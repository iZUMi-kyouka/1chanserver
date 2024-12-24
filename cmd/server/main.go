package main

import (
	"1chanserver/internal/api/api_dev"
	"1chanserver/internal/api/api_thread"
	"1chanserver/internal/api/api_user"
	"1chanserver/internal/database"
	_ "1chanserver/internal/database"
	"1chanserver/internal/middleware"
	"fmt"
	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("Starting server...")
	database.InitDB()

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Set("db", database.DB)
		c.Next()
	})

	{
		v1 := r.Group("/api/v1")
		v1.GET("/healthcheck", api_dev.HealthCheck)
		v1.GET("/authcheck", middleware.AuthMiddleware(), api_dev.AuthCheck)
		users := v1.Group("/users")
		users.POST("/register", api_user.Register)
		users.POST("/login", api_user.Login)
		threads := v1.Group("/threads")
		threads.POST("/new", middleware.AuthMiddleware(), api_thread.New)
		threads.GET("/view/:threadID", api_thread.View)
	}

	r.Run()
}
