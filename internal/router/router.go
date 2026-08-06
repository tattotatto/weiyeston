// Package router central route registration
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

// Dependencies dependency injection container
type Dependencies struct {
	Config        *config.Config
	DB            *sqlx.DB
	Redis         *redis.Client
	Logger        *zap.Logger
	WechatService *wechat.WechatService
	AIService     *aiservice.AIService
}

// Setup registers all routes and middleware, returns configured gin.Engine
// Order: Recovery -> Logger -> CORS (global middleware)
// Then health check, WeChat callback, H5 pages (no auth)
// Finally admin API group (requires JWT auth)
func Setup(deps *Dependencies) *gin.Engine {
	// Set Gin mode
	gin.SetMode(deps.Config.Server.Mode)

	r := gin.New()

	// ============ Global middleware (in order) ============
	r.Use(middleware.Recovery(deps.Logger))
	r.Use(middleware.Logger(deps.Logger))
	r.Use(middleware.CORS(deps.Config.CORS))

	// ============ Health check ============
	healthHandler := api.NewHealthHandler(deps.DB, deps.Redis)
	r.GET("/api/v1/health", healthHandler.Check)
	r.GET("/api/health", healthHandler.Check)

	// ============ Root path -> admin SPA ============
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/admin") })

	// ============ Test routes (dev/test environment) ============
	if deps.Config.Server.Mode != "release" {
		testGroup := r.Group("/api/v1/__tests__")
		registerTestRoutes(testGroup, deps)
	}

	// ============ WeChat callback (no auth, protected by WeChat signature) ============
	// T3: WeChat third-party platform callback endpoint
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

	// ============ H5 display pages (no auth) ============
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

	// ============ Auth routes (login/register/refresh independent of Auth middleware group) ============
	authHandler := api.NewAuthHandler(deps.DB, deps.Redis, deps.Config.JWT)
	r.POST("/api/v1/auth/login", authHandler.Login)
	r.POST("/api/v1/auth/register", authHandler.Register)

	// Admin and server info handlers
	adminHandler := api.NewAdminHandler(deps.DB)
	serverHandler := api.NewServerHandler()

	// /auth/refresh registered independently - cannot be in Auth middleware group (needs to accept expired tokens)
	r.POST("/api/v1/auth/refresh", authHandler.Refresh)

	// ============ Admin API (JWT auth) ============
	v1 := r.Group("/api/v1")
	v1.Use(middleware.Auth(deps.Config.JWT))
	v1.Use(middleware.Tenant())
	{
		// Auth related (needs JWT)
		v1.GET("/auth/me", authHandler.Me)
		v1.POST("/auth/logout", authHandler.Logout)

		// Server info (needs auth, any role)
		v1.GET("/server/info", serverHandler.GetInfo)

		// Admin routes (requires admin role)
		adminGroup := v1.Group("/admin")
		adminGroup.Use(middleware.RequireRole("admin"))
		{
			adminGroup.GET("/users", adminHandler.ListUsers)
			adminGroup.PUT("/users/:id", adminHandler.UpdateUser)
		}

		// WeChat account management
		var accountRepo *account.Repo
		var accountHandler *api.AccountHandler
		if deps.WechatService != nil {
			cacheClient := cache.New(deps.Redis)
			accountRepo = &account.Repo{DB: deps.DB}
			accountHandler = api.NewAccountHandler(accountRepo, deps.WechatService, cacheClient, deps.Logger)
		}

		if accountHandler != nil {
			// T3 existing
			v1.POST("/accounts/auth-url", accountHandler.GenerateAuthURL)
			// T4 full implementation
			v1.GET("/accounts", accountHandler.List)
			v1.POST("/accounts", accountHandler.Create)

			// Account detail/edit/delete + replies/menu - needs ownership check
			accountOwnership := func() gin.HandlerFunc {
				if accountRepo != nil {
					return middleware.CheckAccountOwnership(accountRepo)
				}
				return func(c *gin.Context) { c.Next() }
			}()

			accountGroup := v1.Group("/accounts")
			accountGroup.Use(accountOwnership)
			{
				accountGroup.GET("/:id", accountHandler.GetByID)
				accountGroup.PUT("/:id", accountHandler.Update)
				accountGroup.DELETE("/:id", accountHandler.Delete)
				// T3 existing - static routes before parameter routes
				accountGroup.GET("/:id/auth-status", accountHandler.GetAuthStatus)
			}
		} else {
			v1.POST("/accounts/auth-url", placeholderJSON)
			v1.GET("/accounts", placeholderJSON)
			v1.POST("/accounts", placeholderJSON)
			v1.GET("/accounts/:id", placeholderJSON)
			v1.PUT("/accounts/:id", placeholderJSON)
			v1.DELETE("/accounts/:id", placeholderJSON)
			v1.GET("/accounts/:id/auth-status", placeholderJSON)
		}

		// T5: Auto-reply rules
		var replyHandler *api.ReplyHandler
		if deps.DB != nil {
			replyRepo := &reply.Repo{DB: deps.DB}
			replyHandler = api.NewReplyHandler(replyRepo, deps.Logger)
		}
		if replyHandler != nil {
			// Reply routes need account ownership check
			replyOwnership := func() gin.HandlerFunc {
				if accountRepo != nil {
					return middleware.CheckAccountOwnership(accountRepo)
				}
				return func(c *gin.Context) { c.Next() }
			}()

			replyGroup := v1.Group("/accounts")
			replyGroup.Use(replyOwnership)
			{
				replyGroup.GET("/:id/replies", replyHandler.List)
				replyGroup.POST("/:id/replies", replyHandler.Create)
			}
			v1.PUT("/replies/:id", replyHandler.Update)
			v1.DELETE("/replies/:id", replyHandler.Delete)
		} else {
			v1.GET("/accounts/:id/replies", placeholderJSON)
			v1.POST("/accounts/:id/replies", placeholderJSON)
			v1.PUT("/replies/:id", placeholderJSON)
			v1.DELETE("/replies/:id", placeholderJSON)
		}

		// T6: WeChat custom menu
		var menuHandler *api.MenuHandler
		if deps.DB != nil {
			menuRepo := api.NewMenuRepo(deps.DB)
			menuHandler = api.NewMenuHandler(menuRepo, deps.Logger)
		}
		if menuHandler != nil {
			// Menu routes need account ownership check
			menuOwnership := func() gin.HandlerFunc {
				if accountRepo != nil {
					return middleware.CheckAccountOwnership(accountRepo)
				}
				return func(c *gin.Context) { c.Next() }
			}()

			menuGroup := v1.Group("/accounts")
			menuGroup.Use(menuOwnership)
			{
				menuGroup.GET("/:id/menu", menuHandler.GetMenu)
				menuGroup.POST("/:id/menu", menuHandler.SaveDraft)
				menuGroup.PUT("/:id/menu/publish", menuHandler.Publish)
				menuGroup.DELETE("/:id/menu", menuHandler.DeleteDraft)
			}
		} else {
			v1.GET("/accounts/:id/menu", placeholderJSON)
			v1.POST("/accounts/:id/menu", placeholderJSON)
			v1.PUT("/accounts/:id/menu/publish", placeholderJSON)
			v1.DELETE("/accounts/:id/menu", placeholderJSON)
		}

		// T12: Micro-website CMS
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

		// T14: Voting
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

		// T7: Material management
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

		// T10: Template system
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

		// T11: AI integration
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

		// T13: Quick news
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

		// 服务器信息

		// 管理员路由（需要 admin 角色）
		admin := v1.Group("/admin")
		admin.Use(middleware.RequireRole("admin"))
		{
			admin.GET("/users", adminHandler.ListUsers)
			admin.PUT("/users/:id", adminHandler.UpdateUser)
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
	// ============ Upload files ============
	r.Static("/uploads", "./uploads")
	return r
}

// placeholderJSON placeholder JSON handler for unimplemented handlers (T0 phase)
func placeholderJSON(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": nil,
	})
}

// registerTestRoutes registers test routes only in non-release mode
func registerTestRoutes(r *gin.RouterGroup, deps *Dependencies) {
	// Database connection status
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

	// Config desensitization endpoint
	r.GET("/config", func(c *gin.Context) {
		// Return desensitized config
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

	// Test data load/clear placeholder
	r.POST("/fixtures/load", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "fixtures load placeholder"})
	})
	r.POST("/fixtures/clear", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "fixtures clear placeholder"})
	})
}
