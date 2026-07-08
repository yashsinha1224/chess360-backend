package main

import (
	"log"
	"os"
	"time"

	"example/hello/auth"
	"example/hello/db"
	"example/hello/game"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] no .env file found, relying on real environment variables")
	}

	if err := db.Init(); err != nil {
		log.Fatalf("[FATAL] failed to connect to database: %v", err)
	}

	requireEnv("GOOGLE_CLIENT_ID")
	requireEnv("GOOGLE_CLIENT_SECRET")
	requireEnv("GOOGLE_REDIRECT_URL")
	requireEnv("JWT_SECRET")
	frontendURL := requireEnv("FRONTEND_URL")

	hub := game.NewHub()
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{frontendURL},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.Static("/static", "./static")

	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"status": "ok"})
	})

	router.GET("/auth/google/login", auth.GoogleLogin)
	router.GET("/auth/google/callback", auth.GoogleCallback)

	router.GET("/ws", game.HandleWebSocket(hub))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on http://localhost:%s\n", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("[FATAL] server failed: %v", err)
	}
}

func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("[FATAL] required env var %s is not set", key)
	}
	return val
}
