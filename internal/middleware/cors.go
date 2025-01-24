package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AddAllowHeaders("Authorization", "Device-ID")
	config.AllowCredentials = true
	config.AllowOrigins = []string{"http://localhost:3000", "https://localhost:3000"} // Change to public address of frontend instance
	return cors.New(config)
}
