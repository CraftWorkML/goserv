package server

import (
	"fmt"
	app "goserv/src/app"
	cfg "goserv/src/configuration"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func RunServer(config *cfg.Properties) {
	// Create Gin router
	//gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	//
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization", "Cache-Control", "Access-Control-Allow-Origin", "access-control-allow-headers", "Origin", "User-Agent", "Referrer", "Host", "Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowOriginFunc:  func(origin string) bool { return true },
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	//
	clientS3, err := app.NewMinioS3Client(
		config.S3.Host,
		config.S3.AccessKey,
		config.S3.SecretKey,
		config.S3.Bucket,
		true)
	if err != nil {
		log.Printf("Error: could not connect to minio %e", err)
	}
	// Instantiate recipe Handler and provide a data store implementation
	handler := NewHandler(config, clientS3)

	// Register Routes
	router.GET("/health", handler.GetHealth)
	router.GET("/", handler.Root)
	router.GET("/login", handler.Login)
	router.GET("/singin", handler.Singin)
	router.GET("/logout", handler.Logout)
	router.GET("/callback", handler.Callback)
	router.GET("/account", handler.Account)
	router.GET("/images", handler.GetImageList)
	router.GET("/tracks", handler.GetAudioList)
	router.POST("/image", handler.PostImage)
	router.POST("/ml/image", handler.SendImageToML)
	router.POST("/ml/ts", handler.SendTSToML)
	router.POST("/ml/track", handler.SendTrackToML)
	router.POST("/ml/melody", handler.SendMelodyToML)
	router.POST("/ml/message", handler.SendMessageToML)

	router.DELETE("/images", handler.DeleteImage)

	router.NoRoute(func(ctx *gin.Context) { ctx.JSON(http.StatusNotFound, gin.H{}) })
	// Start the server
	router.Run(fmt.Sprintf(":%s", config.Server.Port))
}
