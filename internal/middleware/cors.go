package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"os"
)

func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AddAllowHeaders("Authorization", "Device-ID")
	config.AllowCredentials = true
	deploymentEnv := os.Getenv("DEPLOYMENT_ENV")
	if deploymentEnv == "cloud" {
		config.AllowOrigins = []string{"https://onechan.xyz"}
	} else if deploymentEnv == "local" {
		config.AllowOrigins = []string{"http://localhost:3000", "https://localhost:3000"} // Change to public address of frontend instance
	}
	return cors.New(config)
}
