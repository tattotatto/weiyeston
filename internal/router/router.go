// Package router central route registration
package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/weiyeston/weiyeston-v2/internal/cache"
	"github.com/weiyeston/weiyeston-v2/internal/config"
	"github.com/weiyeston/weiyeston-v2/internal/handler/api"
	"github.com/weiyeston/weiyeston-v2/internal/handler/wx"
	"github.com/weiyeston/weiyeston-v2/internal/middleware"
	"github.com/weiyeston/weiyeston-v2/internal/repository/account"
	"github.com/weiyeston/weiyeston-v2/internal/repository/reply"
	aiservice "github.com/weiyeston/weiyeston-v2/internal/service/ai"
	"github.com/weiyeston/weiyeston-v2/internal/service/wechat"
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

// Setup registers all routes and middleware, returns configured gin.Engine.
// Route groups (in registration order):
//  1. Public — health, WeChat callback, H5 pages (no auth)
//  2. Auth — login, register, refresh (no JWT middleware)
//  3. API v1 — JWT + Tenant middleware, then sub-groups for each feature
//  4. Admin — RequireRole("admin") within API v1
//  5. SPA — static file serving for the admin frontend
func Setup(deps *Dependencies) *gin.Engine {
	// Set Gin mode
	gin.SetMode(deps.Config.Server.Mode)

	r := gin.New()

	// ============ Global middleware (in order) ============
	r.Use(middleware.Recovery(deps.Logger))
	r.Use(middleware.Logger(deps.Logger))
	r.Use(middleware.CORS(deps.Config.CORS))

	// ============ Group 1: Public routes (no auth) ============

	// Health check
	healthHandler := api.NewHealthHandler(deps.DB, deps.Redis)
	r.GET("/api/health", healthHandler.Check)
	r.GET("/api/v1/health", healthHandler.Check)

	// Root → SPA landing page
	r.GET("/", func(c *gin.Context) { c.File("./web/admin/index.html") })

	// Test routes (dev/test environment only)
	if deps.Config.Server.Mode != "release" {
		testGroup := r.Group("/api/v1/__tests__")
		registerTestRoutes(testGroup, deps)
	}

	// WeChat third-party platform callback (protected by WeChat signature, not JWT)
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

	// H5 display pages (no auth)
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

	// ============ Group 2: Auth routes (login/register/refresh — no JWT middleware) ============
	authHandler := api.NewAuthHandler(deps.DB, deps.Redis, deps.Config.JWT)
	r.POST("/api/v1/auth/login", authHandler.Login)
	r.POST("/api/v1/auth/register", authHandler.Register)

	// /auth/refresh is independent — must accept expired tokens, so it cannot be inside the Auth group
	r.POST("/api/v1/auth/refresh", authHandler.Refresh)

	// ============ Group 3: API v1 (JWT + Tenant middleware) ============
	adminHandler := api.NewAdminHandler(deps.DB)
	serverHandler := api.NewServerHandler()

	v1 := r.Group("/api/v1")
	v1.Use(middleware.Auth(deps.Config.JWT))
	v1.Use(middleware.Tenant())
	{
		// -- Auth (needs valid JWT) --
		v1.GET("/auth/me", authHandler.Me)
		v1.POST("/auth/logout", authHandler.Logout)

		// -- Server info (any authenticated user) --
		v1.GET("/server/info", serverHandler.GetInfo)

		// -- Group 4: Admin routes (RequireRole("admin")) --
		adminGroup := v1.Group("/admin")
		adminGroup.Use(middleware.RequireRole("admin"))
		{
			adminGroup.GET("/users", adminHandler.ListUsers)
			adminGroup.PUT("/users/:id", adminHandler.UpdateUser)
		}

		// -- WeChat account management --
		var accountRepo *account.Repo
		var accountHandler *api.AccountHandler
		if deps.WechatService != nil {
			cacheClient := cache.New(deps.Redis)
			accountRepo = &account.Repo{DB: deps.DB}
			accountHandler = api.NewAccountHandler(accountRepo, deps.WechatService, cacheClient, deps.Logger)
		}

		if accountHandler != nil {
			v1.POST("/accounts/auth-url", accountHandler.GenerateAuthURL)
			v1.GET("/accounts", accountHandler.List)
			v1.POST("/accounts", accountHandler.Create)

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

		// -- Auto-reply rules --
		var replyHandler *api.ReplyHandler
		if deps.DB != nil {
			replyRepo := &reply.Repo{DB: deps.DB}
			replyHandler = api.NewReplyHandler(replyRepo, deps.Logger)
		}
		if replyHandler != nil {
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

		// -- WeChat custom menu --
		var menuHandler *api.MenuHandler
		if deps.DB != nil {
			menuRepo := api.NewMenuRepo(deps.DB)
			menuHandler = api.NewMenuHandler(menuRepo, deps.Logger)
		}
		if menuHandler != nil {
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

		// -- Micro-website CMS --
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

		// -- Voting --
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

		// -- Material management --
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

		// -- Template system --
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

		// -- AI integration --
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

		// -- Quick news --
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

	// ============ Group 5: SPA static files ============
	r.GET("/admin", func(c *gin.Context) { c.File("./web/admin/index.html") })
	r.Static("/admin/assets", "./web/admin/assets")
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		// API/微信/H5路径不做SPA回退，返回标准404
		if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/wx/") || strings.HasPrefix(p, "/h5/") {
			http.NotFound(c.Writer, c.Request)
			return
		}
		// 其他所有路径走SPA（React Router处理）
		c.File("./web/admin/index.html")
	})

	// Static uploads
	r.Static("/uploads", "./uploads")

	return r
}

// placeholderJSON placeholder handler for routes whose real handler cannot be constructed
// (e.g. when DB / external service dependency is nil).
func placeholderJSON(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": nil,
	})
}

// registerTestRoutes registers test routes only in non-release mode.
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
