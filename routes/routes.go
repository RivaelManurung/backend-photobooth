package routes

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"backendphotobooth/handlers"
	"backendphotobooth/middleware"
	"backendphotobooth/services"
	"backendphotobooth/utils"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter initializes and returns the gin router with all routes configured
func SetupRouter(cfg *config.Config, 
	authHandler *handlers.AuthHandler,
	templateHandler *handlers.TemplateHandler,
	templateAdminHandler *handlers.TemplateAdminHandler,
	photoHandler *handlers.PhotoHandler,
	paymentHandler *handlers.PaymentHandler,
	goPayHandler *handlers.GoPayHandler,
	adminHandler *handlers.AdminHandler,
	sessionHandler *handlers.SessionHandler,
	searchHandler *handlers.SearchHandler,
	promoHandler *handlers.PromoHandler,
	docsHandler *handlers.DocsHandler,
	wsHub *services.Hub) *gin.Engine {

	// Create router
	router := gin.New()

	// 1. Global Recovery & Error Handling
	router.Use(middleware.RecoveryMiddleware())
	router.Use(middleware.ErrorHandlerMiddleware())

	// 2. Security Headers Middleware
	router.Use(middleware.SecurityMiddleware())

	// 2. CORS Configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Server.AllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 3. Global Logging & Rate Limiting
	router.Use(middleware.ZapLogger(utils.Logger))
	router.Use(middleware.RateLimitMiddleware(200))

	// Serve static files
	router.Static("/uploads", "./uploads")

	// Health check (Enhanced with DB check)
	router.GET("/health", func(c *gin.Context) {
		dbStatus := "connected"
		sqlDB, err := database.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "disconnected"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"database":  dbStatus,
			"message":   "Photo Booth Backend is healthy",
			"timestamp": time.Now(),
			"version":   "1.1.0",
			"env":       cfg.Server.Environment,
		})
	})

	// Documentation
	router.GET("/docs", docsHandler.GetSwaggerUI)
	router.GET("/swagger", docsHandler.GetSwaggerUI)
	router.GET("/api-docs", docsHandler.GetSwaggerUI)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Documentation endpoints
		v1.GET("/docs", docsHandler.GetSwaggerUI)
		v1.GET("/docs/swagger.json", docsHandler.GetSwaggerJSON)
		
		// Public routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
		}

		v1.POST("/photos/strip-public", photoHandler.UploadPublicStrip)

		// Templates
		templates := v1.Group("/templates")
		templates.Use(middleware.OptionalAuthMiddleware(cfg))
		{
			templates.GET("", templateHandler.GetTemplates)
			templates.GET("/categories", templateHandler.GetTemplateCategories)
			templates.GET("/:id", templateHandler.GetTemplate)
			templates.POST("/:id/usage", templateHandler.IncrementTemplateUsage)
		}

		v1.GET("/pricing", paymentHandler.GetPricingPlans)

		// Search
		search := v1.Group("/search")
		search.Use(middleware.OptionalAuthMiddleware(cfg))
		{
			search.GET("", searchHandler.GlobalSearch)
			search.GET("/templates", searchHandler.SearchTemplates)
			search.GET("/suggestions", searchHandler.GetSearchSuggestions)
			search.GET("/popular", searchHandler.GetPopularSearches)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			// Profile
			profile := protected.Group("/profile")
			{
				profile.GET("", authHandler.GetProfile)
				profile.PUT("", authHandler.UpdateProfile)
				profile.POST("/change-password", authHandler.ChangePassword)
			}

			// Photos
			photos := protected.Group("/photos")
			{
				photos.POST("", photoHandler.UploadPhoto)
				photos.GET("", photoHandler.GetPhotos)
				photos.GET("/:id", photoHandler.GetPhoto)
				photos.DELETE("/:id", photoHandler.DeletePhoto)
				photos.GET("/:id/download", photoHandler.DownloadPhoto)
				photos.POST("/:id/favorite", photoHandler.ToggleFavorite)
				photos.POST("/strip", photoHandler.CreatePhotoStrip)
			}

			// Orders
			orders := protected.Group("/orders")
			{
				orders.POST("/subscription", paymentHandler.CreateSubscriptionOrder)
				orders.GET("", paymentHandler.GetOrders)
				orders.GET("/:id", paymentHandler.GetOrder)
				orders.POST("/:id/cancel", paymentHandler.CancelOrder)
			}

			// Payments
			payment := protected.Group("/payment")
			{
				payment.POST("/qris/create", goPayHandler.CreateQRISPayment)
				payment.GET("/qris/:order_id", goPayHandler.GetQRISPayment)
				payment.GET("/qris/:order_id/status", goPayHandler.CheckQRISStatus)
				payment.POST("/qris/:order_id/cancel", goPayHandler.CancelQRISPayment)
			}

			// Sessions
			sessions := protected.Group("/sessions")
			{
				sessions.POST("", sessionHandler.CreateSession)
				sessions.GET("", sessionHandler.GetUserSessions)
				sessions.GET("/:session_id", sessionHandler.GetSession)
				sessions.PUT("/:session_id", sessionHandler.UpdateSession)
				sessions.POST("/:session_id/end", sessionHandler.EndSession)
				sessions.POST("/:session_id/extend", sessionHandler.ExtendSession)
				sessions.GET("/:session_id/photos", sessionHandler.GetSessionPhotos)
				sessions.DELETE("/:session_id", sessionHandler.DeleteSession)
			}

			// Promo
			promo := protected.Group("/promo")
			{
				promo.POST("/validate", promoHandler.ValidatePromoCode)
			}
		}

		// Admin routes
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware(cfg))
		admin.Use(middleware.AdminMiddleware())
		{
			admin.GET("/dashboard/stats", adminHandler.GetDashboardStats)
			admin.GET("/health", adminHandler.GetSystemHealth)

			adminUsers := admin.Group("/users")
			{
				adminUsers.GET("", adminHandler.GetAllUsers)
				adminUsers.GET("/:id", adminHandler.GetUser)
				adminUsers.PUT("/:id/status", adminHandler.UpdateUserStatus)
				adminUsers.DELETE("/:id", adminHandler.DeleteUser)
			}

			adminTemplates := admin.Group("/templates")
			{
				adminTemplates.GET("", templateAdminHandler.GetAllTemplates)
				adminTemplates.POST("", templateAdminHandler.CreateTemplate)
				adminTemplates.PUT("/:id", templateAdminHandler.UpdateTemplate)
				adminTemplates.DELETE("/:id", templateAdminHandler.DeleteTemplate)
			}

			adminPromo := admin.Group("/promo")
			{
				adminPromo.POST("", promoHandler.CreatePromoCode)
				adminPromo.GET("", promoHandler.GetPromoCodes)
			}
		}

		// Webhooks
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/stripe", paymentHandler.WebhookStripe)
			webhooks.POST("/midtrans", paymentHandler.WebhookMidtrans)
			webhooks.POST("/gopay", goPayHandler.GoPayCallback)
		}

		// WebSocket
		v1.GET("/ws", func(c *gin.Context) {
			services.ServeWs(wsHub, c.Writer, c.Request)
		})
	}

	// 404 handler
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Route not found",
		})
	})

	return router
}
