package main

import (
	"backend-go/config"
	"backend-go/routes"
	"backend-go/services"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Fatal("Gagal memuat konfigurasi:", err)
	}

	if err := config.LoadAWSConfig(); err != nil {
		log.Fatal("Gagal menginisialisasi AWS:", err)
	}

	if err := config.InitDB(); err != nil {
		log.Fatal("Gagal menginisialisasi database (PostgreSQL):", err)
	}

	if err := config.InitMongoDB(); err != nil {
		log.Fatal("Gagal menginisialisasi database (MongoDB):", err)
	}

	services.InitVoiceServices()

	config.InitLogger()

	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	trustedProxies := []string{"127.0.0.1", "::1"}
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		log.Fatal("Gagal menetapkan proxy tepercaya:", err)
	}

	corsConfig := cors.Config{
		AllowOrigins:     []string{"http://localhost:8501", "http://127.0.0.1:8501"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	r.Use(cors.New(corsConfig))

	// Middleware untuk menangani semua preflight OPTIONS request
	r.Use(func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	sessionKey := []byte(config.SessionSecret)
	if len(sessionKey) == 0 {
		log.Fatal("Session secret key belum dikonfigurasi")
	}
	store := cookie.NewStore(sessionKey)
	store.Options(sessions.Options{Path: "/", MaxAge: 86400, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	r.Use(sessions.Sessions("mysession", store))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")

		if err := config.CloseDB(); err != nil {
			log.Printf("Database shutdown error: %v", err)
		}

		os.Exit(0)
	}()

	routes.SetupRoutes(r)

	serverAddr := ":" + config.Port
	log.Printf("Server starting on %s", serverAddr)
	if err := r.Run(serverAddr); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
