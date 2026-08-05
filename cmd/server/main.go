// Package main 微盈通 V2 服务入口
// 启动流程：config → logger → DB → Redis → WechatService → router → graceful shutdown
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/weiyeston/weiyeston-v2/internal/cache"
	"github.com/weiyeston/weiyeston-v2/internal/config"
	"github.com/weiyeston/weiyeston-v2/internal/database"
	"github.com/weiyeston/weiyeston-v2/internal/repository/account"
	"github.com/weiyeston/weiyeston-v2/internal/repository/ticket"
	"github.com/weiyeston/weiyeston-v2/internal/router"
	"github.com/weiyeston/weiyeston-v2/internal/service/wechat"
)

func main() {
	// 1. 解析命令行参数
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	flag.Parse()

	// 2. 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 3. 初始化日志
	logger := initLogger(cfg)
	zap.ReplaceGlobals(logger)
	defer logger.Sync()

	// 4. 初始化数据库连接池
	db, err := initDB(cfg)
	if err != nil {
		logger.Fatal("数据库连接失败", zap.Error(err))
	}
	defer db.Close()

	// 4.1 执行数据库迁移
	if err := database.RunMigrations(db, "migrations"); err != nil {
		logger.Fatal("数据库迁移失败", zap.Error(err))
	}
	logger.Info("数据库迁移完成")

	// 5. 初始化 Redis 连接
	rdb, err := initRedis(cfg)
	if err != nil {
		logger.Fatal("Redis 连接失败", zap.Error(err))
	}
	defer rdb.Close()

	// 6. 初始化微信服务（T3 新增）
	cacheClient := cache.New(rdb)
	ticketRepo := &ticket.Repo{DB: db}
	accountRepo := &account.Repo{DB: db}
	wechatService := wechat.NewWechatService(
		cfg, cacheClient, db, logger, accountRepo, ticketRepo,
	)

	// 6.1 启动 component token 后台刷新 goroutine（带优雅关闭）
	ctx, cancelRefresh := context.WithCancel(context.Background())
	defer cancelRefresh()
	wechatService.StartComponentTokenRefresher(ctx)

	// 6.2 启动时自检（异步，5秒后执行）
	go startupSelfCheck(cfg, wechatService, logger)

	// 7. 组装依赖 → 路由注册
	deps := &router.Dependencies{
		Config:        cfg,
		DB:            db,
		Redis:         rdb,
		Logger:        logger,
		WechatService: wechatService,
	}
	engine := router.Setup(deps)

	// 8. 启动 HTTP 服务
	readTimeout := cfg.Server.ReadTimeout
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}
	writeTimeout := cfg.Server.WriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = 60 * time.Second
	}
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// 9. 优雅关闭
	go func() {
		logger.Info("服务启动", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务异常退出", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 停止 token 刷新 goroutine
	wechatService.StopComponentTokenRefresher()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("服务关闭异常", zap.Error(err))
	}
	logger.Info("服务已安全退出")
}

// startupSelfCheck 启动时自检：验证第三方平台配置是否有效
func startupSelfCheck(cfg *config.Config, wechatService *wechat.WechatService, logger *zap.Logger) {
	time.Sleep(5 * time.Second) // 等待服务完全启动
	ctx := context.Background()

	// 检查是否有 component_verify_ticket
	ticket, err := wechatService.GetComponentVerifyTicket(ctx)
	if err != nil || ticket == "" {
		logger.Warn("未检测到 component_verify_ticket，component_access_token 将无法获取。"+
			"请确认微信开放平台已将服务器地址配置为: "+cfg.Wechat.ServerURL+"/wx/component/callback",
		)
		return
	}

	// 尝试获取 component_access_token
	token, err := wechatService.GetComponentAccessToken(ctx)
	if err != nil || token == "" {
		logger.Error("获取 component_access_token 失败，请检查配置", zap.Error(err))
		return
	}

	logger.Info("微信第三方平台初始化成功，component_access_token 已就绪")
}

// initLogger 初始化 zap 日志器
func initLogger(cfg *config.Config) *zap.Logger {
	var level zapcore.Level
	switch cfg.Log.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	var encoderConfig zapcore.EncoderConfig
	if cfg.Log.Format == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if cfg.Log.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	var writeSyncer zapcore.WriteSyncer
	switch cfg.Log.Output {
	case "file":
		if cfg.Log.File != "" {
			file, err := os.OpenFile(cfg.Log.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Printf("无法打开日志文件 %s: %v，回退到 stdout", cfg.Log.File, err)
				writeSyncer = zapcore.AddSync(os.Stdout)
			} else {
				writeSyncer = zapcore.AddSync(file)
			}
		} else {
			writeSyncer = zapcore.AddSync(os.Stdout)
		}
	default:
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(encoder, writeSyncer, level)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

// initDB 初始化 PostgreSQL 连接池
func initDB(cfg *config.Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	return db, nil
}

// initRedis 初始化 Redis 连接
func initRedis(cfg *config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}

	return rdb, nil
}
