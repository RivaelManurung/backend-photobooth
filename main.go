package main

import (
	"backendphotobooth/config"
	"backendphotobooth/database"
	"backendphotobooth/handlers"
	"backendphotobooth/routes"
	"backendphotobooth/services"
	"backendphotobooth/utils"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize Logger
	utils.InitLogger(cfg.Server.Environment)
	defer utils.Logger.Sync()

	// Set Gin mode
	gin.SetMode(cfg.Server.Environment)

	// Initialize database
	if err := database.InitDatabase(cfg); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize services
	storageService := services.NewStorageService(cfg)
	imageProcessor := services.NewImageProcessor(storageService)
	goPayQRISService := services.NewGoPayQRISService(cfg)
	templateProcessor := services.NewTemplateProcessor("./uploads/templates")
	redisService := services.NewRedisService(cfg)
	queueService := services.NewQueueService(redisService)

	// Initialize WebSocket Hub
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
	twoFAHandler := handlers.NewTwoFAHandler(cfg)
	auditHandler := handlers.NewAuditHandler()
	docsHandler := handlers.NewDocsHandler()
	wsHandler := handlers.NewWebSocketHandler(wsHub)
	photoHandler.SetQueueService(queueService)

	// Setup Router
	router := routes.SetupRouter(cfg,
		authHandler, templateHandler, templateAdminHandler,
		photoHandler, paymentHandler, goPayHandler,
		adminHandler, sessionHandler, searchHandler,
		promoHandler, twoFAHandler, auditHandler, docsHandler, wsHandler, wsHub,
	)

	// Configure HTTP Server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		log.Printf("🚀 Server starting on port %s", cfg.Server.Port)
		log.Printf("📝 Environment: %s", cfg.Server.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so no need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
