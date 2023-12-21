package server

import (
	"fmt"
	app "goserv/src/app"
	cfg "goserv/src/configuration"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

func RunServer(config *cfg.Properties) {
	// Create Gin router
	//gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	//
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3000"},
		AllowMethods: []string{
			"GET",
			"HEAD",
			"POST",
			"PUT",
			"DELETE",
			"OPTIONS",
			"PATCH"},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"Authorization",
			"Cache-Control",
			"Access-Control-Allow-Origin",
			"access-control-allow-headers",
			"Origin",
			"User-Agent",
			"Referrer",
			"Host",
			"Token"},
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
	handlerAuth := NewAuthHandler(config)
	handlerS3 := NewS3Handler(config, clientS3)
	handlerExternal := NewExternalHandler(config)

	// Register Routes
	router.GET("/health", handlerAuth.GetHealth)
	router.GET("/", handlerAuth.Root)
	router.GET("/login", handlerAuth.Login)
	router.GET("/singin", handlerAuth.Singin)
	router.GET("/logout", handlerAuth.Logout)
	router.GET("/callback", handlerAuth.Callback)
	router.GET("/account", handlerAuth.Account)
	router.GET("/images", handlerS3.GetImageList)
	router.GET("/tracks", handlerS3.GetAudioList)
	router.POST("/image", handlerS3.PostImage)
	router.DELETE("/images", handlerS3.DeleteImage)
	router.NoRoute(func(ctx *gin.Context) { ctx.JSON(http.StatusNotFound, gin.H{}) })
	// Simple group: v2
	ml := router.Group("/ml")
	{
		ml.POST("/image", handlerExternal.SendImageToML)
		ml.POST("/ts", handlerExternal.SendTSToML)
		ml.POST("/track", handlerExternal.SendTrackToML)
		ml.POST("/melody", handlerExternal.SendMelodyToML)
		ml.POST("/message", handlerExternal.SendMessageToML)
	}

	pprof.Register(router)
	// Start the server
	router.Run(fmt.Sprintf(":%s", config.Server.Port))
}
