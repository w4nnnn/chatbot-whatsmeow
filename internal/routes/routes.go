package routes

import (
	"chatbot-whatsmeow/internal/handlers"
	"chatbot-whatsmeow/internal/services"
	"chatbot-whatsmeow/internal/websocket"

	"github.com/gin-gonic/gin"
)

func SetupRouter(service *services.WhatsMeowService) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1"})

	router.GET("/ws", websocket.WSHandler)
	router.POST("/start", handlers.StartHandler(service))
	router.POST("/stop", handlers.StopHandler(service))
	router.POST("/logout", handlers.LogoutHandler(service))
	router.GET("/status", handlers.StatusHandler(service))

	return router
}