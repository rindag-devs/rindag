package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "rindag/docs"
	"rindag/handler"
	"rindag/middleware"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	ginlogrus "github.com/toorop/gin-logrus"
)

// @title       RinDAG API
// @version     dev
// @description API for RinDAG service.
// @basePath    /

// @securityDefinitions.apikey ApiKeyAuth
// @in                         header
// @name                       X-API-KEY
// @description                API key that needs to be passed as part of the request in order to be authorized.

func setupRouter() *gin.Engine {
	r := gin.New()

	r.Use(ginlogrus.Logger(log.StandardLogger()), gin.Recovery())
	r.Use(gin.Recovery())

	r.GET("/ping", handler.HandlePing)
	r.POST("/login", handler.HandleLogin)

	authorized := r.Group("/")
	authorized.Use(middleware.JWTMiddleware())
	{
		authorized.DELETE("/logout", handler.HandleLogout)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return r
}

func main() {
	router := setupRouter()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Error listening")
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server with a timeout of 10 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.WithError(err).Fatal("Server shutdown failed")
	}
	log.Info("Server exiting")
}
