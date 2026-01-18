package handlers

import (
	"context"

	"chatbot-whatsmeow/internal/services"

	"github.com/gin-gonic/gin"
)

func StartHandler(service *services.WhatsMeowService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service.IsConnected() {
			c.JSON(400, gin.H{"error": "Already connected"})
			return
		}
		err := service.Connect()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "connecting"})
	}
}

func StopHandler(service *services.WhatsMeowService) gin.HandlerFunc {
	return func(c *gin.Context) {
		service.Disconnect()
		c.JSON(200, gin.H{"status": "disconnected"})
	}
}

func LogoutHandler(service *services.WhatsMeowService) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := service.Logout(context.Background())
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "logged_out"})
	}
}

func StatusHandler(service *services.WhatsMeowService) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := "disconnected"
		if service.IsConnected() {
			status = "connected"
		}
		loggedIn := service.IsLoggedIn()
		c.JSON(200, gin.H{"status": status, "logged_in": loggedIn})
	}
}