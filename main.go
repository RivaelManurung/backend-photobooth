package main

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"backendphotobooth/handlers"
	"backendphotobooth/middleware"
	"backendphotobooth/services"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Set Gin mode
	gin.SetMode(cfg.Server.Environment)

	// Initialize database
	if err := database.InitDatabase(cfg); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize services
	storageService := services.NewStorageService(cfg)
	imageProcessor := services.NewImageProcessor(storageService)
	emailService := services.NewEmailService(cfg)
	goPayQRISService := services.NewGoPayQRISService(cfg)
	templateProcessor := services.NewTemplateProcessor("./uploads/templates")
	
	// Initialize WebSocket Hub (optional, for real-time notifications)
	wsHub := services.NewHub()
	go wsHub.Run()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(cfg)
	templateHandler := handlers.NewTemplateHandler(storageService)
	templateAdminHandler := handlers.NewTemplateAdminHandler(storageService, templateProcessor)
	photoHandler := handlers.NewPhotoHandler(storageService, imageProcessor)
	paymentHandler := handlers.NewPaymentHandler(cfg)
	goPayHandler := handlers.NewGoPayHandler(cfg, goPayQRISService, wsHub)
	adminHandler := handlers.NewAdminHandler()
	sessionHandler := handlers.NewSessionHandler()
	searchHandler := handlers.NewSearchHandler()
	promoHandler := handlers.NewPromoHandler()
	docsHandler := handlers.NewDocsHandler()
	
	// Use email service (example)
	_ = emailService

	// Create router
	router := gin.Default()

	// Middleware
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.RateLimitMiddleware(100)) // 100 requests per minute

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Server.AllowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Serve static files (uploads)
	router.Static("/uploads", "./uploads")

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"message":   "Photo Booth Backend is running",
			"timestamp": time.Now(),
			"version":   "1.0.0",
		})
	})

	// API Documentation
	router.GET("/docs", docsHandler.GetSwaggerUI)
	router.GET("/swagger", docsHandler.GetSwaggerUI)
	router.GET("/api-docs", docsHandler.GetSwaggerUI)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Documentation endpoints
		v1.GET("/docs", docsHandler.GetSwaggerUI)
		v1.GET("/docs/swagger.json", docsHandler.GetSwaggerJSON)
		v1.GET("/docs/json", docsHandler.GetAPIDocs)
		
		// Public routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
		}

		// Templates (public with optional auth)
		templates := v1.Group("/templates")
		templates.Use(middleware.OptionalAuthMiddleware(cfg))
		{
			templates.GET("", templateHandler.GetTemplates)
			templates.GET("/:id", templateHandler.GetTemplate)
			templates.GET("/categories", templateHandler.GetTemplateCategories)
			templates.POST("/:id/usage", templateHandler.IncrementTemplateUsage)
		}

		// Pricing (public)
		v1.GET("/pricing", paymentHandler.GetPricingPlans)

		// Search (public with optional auth)
		search := v1.Group("/search")
		search.Use(middleware.OptionalAuthMiddleware(cfg))
		{
			search.GET("", searchHandler.GlobalSearch)
			search.GET("/templates", searchHandler.SearchTemplates)
			search.GET("/suggestions", searchHandler.GetSearchSuggestions)
			search.GET("/popular", searchHandler.GetPopularSearches)
		}

		// Protected routes (require authentication)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg))
		{
			// User profile
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

			// Orders & Payments
			orders := protected.Group("/orders")
			{
				orders.POST("/subscription", paymentHandler.CreateSubscriptionOrder)
				orders.GET("", paymentHandler.GetOrders)
				orders.GET("/:id", paymentHandler.GetOrder)
				orders.POST("/:id/cancel", paymentHandler.CancelOrder)
			}

			// GoPay QRIS Payment
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

			// Search (authenticated)
			protectedSearch := protected.Group("/search")
			{
				protectedSearch.GET("/photos", searchHandler.SearchPhotos)
			}

			// Promo Codes
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
			// Dashboard
			admin.GET("/dashboard/stats", adminHandler.GetDashboardStats)
			admin.GET("/health", adminHandler.GetSystemHealth)

			// User management
			adminUsers := admin.Group("/users")
			{
				adminUsers.GET("", adminHandler.GetAllUsers)
				adminUsers.GET("/:id", adminHandler.GetUser)
				adminUsers.PUT("/:id/status", adminHandler.UpdateUserStatus)
				adminUsers.DELETE("/:id", adminHandler.DeleteUser)
			}

			// Reports & Analytics
			admin.GET("/reports/revenue", adminHandler.GetRevenueReport)
			admin.GET("/analytics/templates", adminHandler.GetTemplateAnalytics)
			admin.GET("/analytics/growth", adminHandler.GetUserGrowth)
			admin.GET("/export/users", adminHandler.ExportUsers)

			// Template management
			adminTemplates := admin.Group("/templates")
			{
				adminTemplates.GET("", templateAdminHandler.GetAllTemplates)
				adminTemplates.POST("", templateAdminHandler.CreateTemplate)
				adminTemplates.PUT("/:id", templateAdminHandler.UpdateTemplate)
				adminTemplates.DELETE("/:id", templateAdminHandler.DeleteTemplate)
				adminTemplates.POST("/:id/toggle-status", templateAdminHandler.ToggleTemplateStatus)
				adminTemplates.POST("/:id/toggle-featured", templateAdminHandler.ToggleTemplateFeatured)
				adminTemplates.POST("/:id/duplicate", templateAdminHandler.DuplicateTemplate)
				adminTemplates.GET("/categories", templateAdminHandler.GetTemplateCategories)
				adminTemplates.GET("/analytics", templateAdminHandler.GetTemplateAnalytics)
				adminTemplates.POST("/upload-asset", templateHandler.UploadTemplateAsset)
			}

			// Promo Code management
			adminPromo := admin.Group("/promo")
			{
				adminPromo.POST("", promoHandler.CreatePromoCode)
				adminPromo.GET("", promoHandler.GetPromoCodes)
				adminPromo.GET("/:id", promoHandler.GetPromoCode)
				adminPromo.PUT("/:id", promoHandler.UpdatePromoCode)
				adminPromo.DELETE("/:id", promoHandler.DeletePromoCode)
				adminPromo.GET("/:id/usage", promoHandler.GetPromoUsageHistory)
				adminPromo.POST("/:id/toggle", promoHandler.TogglePromoStatus)
			}

			// Search (admin)
			admin.GET("/search/users", searchHandler.SearchUsers)
		}

		// Webhooks (no auth, verified by signature)
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/stripe", paymentHandler.WebhookStripe)
			webhooks.POST("/midtrans", paymentHandler.WebhookMidtrans)
			webhooks.POST("/gopay", goPayHandler.GoPayCallback)
		}

		// WebSocket (optional, for real-time notifications)
		v1.GET("/ws", func(c *gin.Context) {
			services.ServeWs(wsHub, c.Writer, c.Request)
		})
	}

	// 404 handler
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Route not found",
			"path":    c.Request.URL.Path,
			"method":  c.Request.Method,
		})
	})

	// Start server
	port := ":" + cfg.Server.Port
	log.Printf("🚀 Server starting on port %s", port)
	log.Printf("📝 Environment: %s", cfg.Server.Environment)
	log.Printf("🗄️  Database: %s@%s:%s/%s", cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	log.Printf("💾 Storage: %s", cfg.Storage.Provider)
	
	if err := router.Run(port); err != nil {
		log.Fatal("❌ Failed to start server:", err)
	}
}
