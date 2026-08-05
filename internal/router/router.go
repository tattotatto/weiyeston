// Package router 路由注册集中管理
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/weiyeston/weiyeston-v2/internal/config"
	"github.com/weiyeston/weiyeston-v2/internal/handler/api"
	"github.com/weiyeston/weiyeston-v2/internal/handler/wx"
	"github.com/weiyeston/weiyeston-v2/internal/middleware"
	"github.com/weiyeston/weiyeston-v2/internal/repository/account"
	"github.com/weiyeston/weiyeston-v2/internal/repository/reply"
	"github.com/weiyeston/weiyeston-v2/internal/service/wechat"


	"github.com/weiyeston/weiyeston-v2/internal/cache"

	aiservice "github.com/weiyeston/weiyeston-v2/internal/service/ai"
)

// Dependencies 依赖注入容器
// 集中管理路由所需要的所有外部依赖
type Dependencies struct {
	Config        *config.Config
	DB            *sqlx.DB
	Redis         *redis.Client
	Logger        *zap.Logger
	WechatService *wechat.WechatService
	AIService     *aiservice.AIService
}

// Setup 注册所有路由和中间件，返回配置好的 gin.Engine
// 注册顺序：Recovery → Logger → CORS（全局中间件）
// 然后是健康检查、微信回调、H5 页面（无需认证）
// 最后是管理后台 API 组（需要 JWT 认证）
func Setup(deps *Dependencies) *gin.Engine {
	// 设置 Gin 模式
	gin.SetMode(deps.Config.Server.Mode)

	r := gin.New()

	// ============ 全局中间件（按顺序） ============
	r.Use(middleware.Recovery(deps.Logger))
	r.Use(middleware.Logger(deps.Logger))
	r.Use(middleware.CORS(deps.Config.CORS))

	// ============ 健康检查 ============
	healthHandler := api.NewHealthHandler(deps.DB, deps.Redis)
	r.GET("/api/v1/health", healthHandler.Check)

	// ============ 根路径 → 管理后台 ============
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/admin") })

	// ============ 测试接口注册（开发/测试环境） ============
	if deps.Config.Server.Mode != "release" {
		testGroup := r.Group("/api/v1/__tests__")
		registerTestRoutes(testGroup, deps)
	}

	// ============ 微信回调（无需认证，由微信签名验证保护） ============
	// T3: 微信第三方平台回调端点
	var componentHandler *wx.ComponentHandler
	if deps.WechatService != nil {
		componentHandler = wx.NewComponentHandler(deps.WechatService, deps.Logger)
		r.POST("/wx/component/callback", componentHandler.HandleComponentCallback)
	} else {
		r.POST("/wx/component/callback", func(c *gin.Context) {
			c.String(http.StatusOK, "wx component callback placeholder")
		})
	}

	r.GET("/wx/callback/:app_id", func(c *gin.Context) {
		c.String(http.StatusOK, "wx callback verify placeholder")
	})
	r.POST("/wx/callback/:app_id", func(c *gin.Context) {
		c.String(http.StatusOK, "wx callback message placeholder")
	})

	// ============ H5 展示页（无需认证） ============
	// Build H5 handlers (need VoteRepo for vote submit)
	var voteH5Handler *api.VoteHandler
	if deps.DB != nil {
		voteRepo := api.NewVoteRepo(deps.DB)
		voteH5Handler = api.NewVoteHandler(voteRepo, deps.Logger)
	}

	r.GET("/h5/article/:id", func(c *gin.Context) {
		c.String(http.StatusOK, "h5 article placeholder")
	})
	r.GET("/h5/news/:account_id", func(c *gin.Context) {
		c.String(http.StatusOK, "h5 news placeholder")
	})
	r.GET("/h5/vote/:id", func(c *gin.Context) {
		c.String(http.StatusOK, "h5 vote placeholder")
	})
	if voteH5Handler != nil {
		r.POST("/h5/vote/:id/submit", voteH5Handler.SubmitVote)
	} else {
		r.POST("/h5/vote/:id/submit", func(c *gin.Context) {
			c.String(http.StatusOK, "h5 vote submit placeholder")
		})
	}

	// ============ 认证路由（登录/刷新独立于 Auth 中间件组） ============
	authHandler := api.NewAuthHandler(deps.DB, deps.Redis, deps.Config.JWT)
	r.POST("/api/v1/auth/login", authHandler.Login)

	// /auth/refresh 独立注册 — 不能放在 Auth 中间件组内（需接收过期 token）
	r.POST("/api/v1/auth/refresh", authHandler.Refresh)

	// ============ 管理后台 API（JWT 认证） ============
	v1 := r.Group("/api/v1")
	v1.Use(middleware.Auth(deps.Config.JWT))
	v1.Use(middleware.Tenant())
	{
		// 认证相关（需要 JWT）
		v1.GET("/auth/me", authHandler.Me)
		v1.POST("/auth/logout", authHandler.Logout)

		// 公众号管理
		var accountHandler *api.AccountHandler
		if deps.WechatService != nil {
			cacheClient := cache.New(deps.Redis)
			accountRepo := &account.Repo{DB: deps.DB}
			accountHandler = api.NewAccountHandler(accountRepo, deps.WechatService, cacheClient, deps.Logger)
		}

		if accountHandler != nil {
			// T3 已有
			v1.POST("/accounts/auth-url", accountHandler.GenerateAuthURL)
			// T4 完整实现
			v1.GET("/accounts", accountHandler.List)
			v1.POST("/accounts", accountHandler.Create)
			v1.GET("/accounts/:id", accountHandler.GetByID)
			v1.PUT("/accounts/:id", accountHandler.Update)
			v1.DELETE("/accounts/:id", accountHandler.Delete)
			// T3 已有 — 注意静态路由在参数路由之前
			v1.GET("/accounts/:id/auth-status", accountHandler.GetAuthStatus)
		} else {
			v1.POST("/accounts/auth-url", placeholderJSON)
			v1.GET("/accounts/:id/auth-status", placeholderJSON)
			v1.GET("/accounts", placeholderJSON)
			v1.POST("/accounts", placeholderJSON)
			v1.GET("/accounts/:id", placeholderJSON)
			v1.PUT("/accounts/:id", placeholderJSON)
			v1.DELETE("/accounts/:id", placeholderJSON)
		}

		// T5: 自动回复规则
		var replyHandler *api.ReplyHandler
		if deps.DB != nil {
			replyRepo := &reply.Repo{DB: deps.DB}
			replyHandler = api.NewReplyHandler(replyRepo, deps.Logger)
		}
		if replyHandler != nil {
			v1.GET("/accounts/:id/replies", replyHandler.List)
			v1.POST("/accounts/:id/replies", replyHandler.Create)
			v1.PUT("/replies/:id", replyHandler.Update)
			v1.DELETE("/replies/:id", replyHandler.Delete)
		} else {
			v1.GET("/accounts/:id/replies", placeholderJSON)
			v1.POST("/accounts/:id/replies", placeholderJSON)
			v1.PUT("/replies/:id", placeholderJSON)
			v1.DELETE("/replies/:id", placeholderJSON)
		}

		// T6: 微信自定义菜单
		var menuHandler *api.MenuHandler
		if deps.DB != nil {
			menuRepo := api.NewMenuRepo(deps.DB)
			menuHandler = api.NewMenuHandler(menuRepo, deps.Logger)
		}
		if menuHandler != nil {
			v1.GET("/accounts/:id/menu", menuHandler.GetMenu)
			v1.POST("/accounts/:id/menu", menuHandler.SaveDraft)
			v1.PUT("/accounts/:id/menu/publish", menuHandler.Publish)
			v1.DELETE("/accounts/:id/menu", menuHandler.DeleteDraft)
		} else {
			v1.GET("/accounts/:id/menu", placeholderJSON)
			v1.POST("/accounts/:id/menu", placeholderJSON)
			v1.PUT("/accounts/:id/menu/publish", placeholderJSON)
			v1.DELETE("/accounts/:id/menu", placeholderJSON)
		}

		// T12: 微官网 CMS
		var cmsHandler *api.CMSHandler
		if deps.DB != nil {
			cmsRepo := &api.CMSRepoDB{DB: deps.DB}
			cmsHandler = api.NewCMSHandler(cmsRepo, deps.Logger)
		}
		if cmsHandler != nil {
			v1.GET("/cms/channels", cmsHandler.ListChannels)
			v1.POST("/cms/channels", cmsHandler.CreateChannel)
			v1.PUT("/cms/channels/:id", cmsHandler.UpdateChannel)
			v1.DELETE("/cms/channels/:id", cmsHandler.DeleteChannel)
			v1.GET("/cms/articles", cmsHandler.ListArticles)
			v1.POST("/cms/articles", cmsHandler.CreateArticle)
			v1.GET("/cms/articles/:id", cmsHandler.GetArticle)
			v1.PUT("/cms/articles/:id", cmsHandler.UpdateArticle)
			v1.DELETE("/cms/articles/:id", cmsHandler.DeleteArticle)
			v1.GET("/cms/articles/:id/preview", cmsHandler.PreviewArticle)
		} else {
			v1.GET("/cms/channels", placeholderJSON)
			v1.POST("/cms/channels", placeholderJSON)
			v1.PUT("/cms/channels/:id", placeholderJSON)
			v1.DELETE("/cms/channels/:id", placeholderJSON)
			v1.GET("/cms/articles", placeholderJSON)
			v1.POST("/cms/articles", placeholderJSON)
			v1.GET("/cms/articles/:id", placeholderJSON)
			v1.PUT("/cms/articles/:id", placeholderJSON)
			v1.DELETE("/cms/articles/:id", placeholderJSON)
			v1.GET("/cms/articles/:id/preview", placeholderJSON)
		}

		// T14: 投票
		var voteHandler *api.VoteHandler
		if deps.DB != nil {
			voteRepo := api.NewVoteRepo(deps.DB)
			voteHandler = api.NewVoteHandler(voteRepo, deps.Logger)
		}
		if voteHandler != nil {
			v1.GET("/votes", voteHandler.ListVotes)
			v1.POST("/votes", voteHandler.CreateVote)
			v1.GET("/votes/:id", voteHandler.GetVote)
			v1.PUT("/votes/:id", voteHandler.UpdateVote)
			v1.DELETE("/votes/:id", voteHandler.DeleteVote)
			v1.GET("/votes/:id/results", voteHandler.GetResults)
		} else {
			v1.GET("/votes", placeholderJSON)
			v1.POST("/votes", placeholderJSON)
			v1.GET("/votes/:id", placeholderJSON)
			v1.PUT("/votes/:id", placeholderJSON)
			v1.DELETE("/votes/:id", placeholderJSON)
			v1.GET("/votes/:id/results", placeholderJSON)
		}

		// T7: 素材管理
		var materialHandler *api.MaterialHandler
		if deps.DB != nil {
			materialRepo := api.NewMaterialRepo(deps.DB)
			uploadDir := deps.Config.Upload.LocalPath
			if uploadDir == "" {
				uploadDir = "./uploads"
			}
			materialHandler = api.NewMaterialHandler(materialRepo, uploadDir, deps.Logger)
		}
		if materialHandler != nil {
			v1.GET("/materials", materialHandler.List)
			v1.POST("/materials/upload", materialHandler.Upload)
			v1.GET("/materials/:id", materialHandler.GetByID)
			v1.DELETE("/materials/:id", materialHandler.Delete)
		} else {
			v1.GET("/materials", placeholderJSON)
			v1.POST("/materials/upload", placeholderJSON)
			v1.GET("/materials/:id", placeholderJSON)
			v1.DELETE("/materials/:id", placeholderJSON)
		}

		// T10: 模板系统
		var templateHandler *api.TemplateHandler
		if deps.DB != nil {
			templateHandler = api.NewTemplateHandler(deps.DB)
		}
		if templateHandler != nil {
			v1.GET("/templates", templateHandler.ListSystemTemplates)
			v1.POST("/templates", templateHandler.SaveTemplate)
		} else {
			v1.GET("/templates", placeholderJSON)
			v1.POST("/templates", placeholderJSON)
		}

		// T11: AI 集成
		var aiHandler *api.AIHandler
		if deps.AIService != nil {
			aiHandler = api.NewAIHandler(deps.AIService)
		}
		if aiHandler != nil {
			v1.POST("/ai/write", aiHandler.Write)
			v1.POST("/ai/layout", aiHandler.Layout)
			v1.POST("/ai/proofread", aiHandler.Proofread)
		} else {
			v1.POST("/ai/write", placeholderJSON)
			v1.POST("/ai/layout", placeholderJSON)
			v1.POST("/ai/proofread", placeholderJSON)
		}

		// T13: 快讯
		var newsHandler *api.NewsHandler
		if deps.DB != nil {
			newsRepo := api.NewNewsRepo(deps.DB)
			newsHandler = api.NewNewsHandler(newsRepo, deps.Logger)
		}
		if newsHandler != nil {
			v1.GET("/news", newsHandler.ListNews)
			v1.POST("/news", newsHandler.CreateNews)
			v1.GET("/news/:id", newsHandler.GetNews)
			v1.PUT("/news/:id", newsHandler.UpdateNews)
			v1.DELETE("/news/:id", newsHandler.DeleteNews)
			v1.GET("/quicknews/users", newsHandler.ListUsers)
		} else {
			v1.GET("/news", placeholderJSON)
			v1.POST("/news", placeholderJSON)
			v1.GET("/news/:id", placeholderJSON)
			v1.PUT("/news/:id", placeholderJSON)
			v1.DELETE("/news/:id", placeholderJSON)
			v1.GET("/quicknews/users", placeholderJSON)
		}
	}


	// ============ 管理后台 SPA ============
	r.GET("/admin", func(c *gin.Context) { c.File("./web/admin/index.html") })
	r.GET("/admin/*filepath", func(c *gin.Context) { c.File("./web/admin/" + c.Param("filepath")) })
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if p == "/admin" || (len(p) > 6 && p[:7] == "/admin/") {
			c.File("./web/admin/index.html")
			return
		}
		http.NotFound(c.Writer, c.Request)
	})
	// ============ 上传文件 ============
	r.Static("/uploads", "./uploads")
	return r
}

// placeholderJSON 占位 JSON 处理函数（T0 阶段用于未实现的 handler）
func placeholderJSON(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": nil,
	})
}

// registerTestRoutes 仅在非 release 模式下注册测试接口
func registerTestRoutes(r *gin.RouterGroup, deps *Dependencies) {
	// 数据库连接状态
	r.GET("/db/status", func(c *gin.Context) {
		errStr := ""
		dbAlive := true
		if deps.DB != nil {
			if err := deps.DB.Ping(); err != nil {
				dbAlive = false
				errStr = err.Error()
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"db_alive": dbAlive,
			"error":    errStr,
		})
	})

	// 配置脱敏端点
	r.GET("/config", func(c *gin.Context) {
		// 返回脱敏后的配置
		safe := *deps.Config
		safe.Database.Password = "***"
		safe.Redis.Password = "***"
		safe.JWT.Secret = "***"
		safe.Wechat.ComponentAppSecret = "***"
		safe.Wechat.EncodingAESKey = "***"
		safe.Upload.S3Key = "***"
		safe.Upload.S3Secret = "***"
		safe.AI.APIKey = "***"
		c.JSON(http.StatusOK, safe)
	})

	// 测试数据加载/清理占位
	r.POST("/fixtures/load", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "fixtures load placeholder"})
	})
	r.POST("/fixtures/clear", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "fixtures clear placeholder"})
	})
}
