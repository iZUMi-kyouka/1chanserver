package main

import (
	"1chanserver/internal/api/api_comment"
	"1chanserver/internal/api/api_dev"
	"1chanserver/internal/api/api_thread"
	"1chanserver/internal/api/api_token"
	"1chanserver/internal/api/api_user"
	"1chanserver/internal/database"
	_ "1chanserver/internal/database"
	"1chanserver/internal/middleware"
	"1chanserver/internal/models/api_error"
	"errors"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"log"
	"time"
)

func main() {
	fmt.Println("Starting server...")
	database.InitDB()

	r := gin.Default()

	config := cors.DefaultConfig()
	config.AddAllowHeaders("Authorization", "Device-ID")
	config.AllowCredentials = true
	config.AllowOrigins = []string{"http://localhost:3000"}
	r.Use(cors.New(config))

	r.Use(
		middleware.PanicRecovery(),
		middleware.RequestIDProvider(),
		middleware.ErrorLogging(),
		middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		c.Set("db", database.DB)
		c.Next()
	})

	{
		v1 := r.Group("/api/v1")
		v1.GET("/healthcheck", api_dev.HealthCheck)
		v1.GET("/authcheck", middleware.Auth(), api_dev.AuthCheck)
		v1.GET("/error", func(c *gin.Context) {
			err := api_error.New(errors.New("not implemented"), 500, "unexpected error")
			err2 := api_error.New(errors.New("not implemented 2"), 500, "unexpected error")
			c.Error(err)
			c.Error(err2)
			return
		})
		v1.GET("/reflect/:required/*optional", api_dev.ReflectPath)

		users := v1.Group("/users")
		{
			usersAuth := users.Group("/", middleware.Auth())
			{
				usersAuth.GET("/logout", api_user.Logout)

			}

			users.POST("/login", api_user.Login)
			users.POST("/register", api_user.Register)
			users.GET("/refresh_new", api_token.RefreshToken("first"))
			users.GET("/refresh", api_token.RefreshToken("continue"))
		}

		threads := v1.Group("/threads")
		{
			threadsAuth := threads.Group("/", middleware.Auth())
			{
				threadsAuth.POST("/new", api_thread.New)
				threadsAuth.POST("/like/:objID", api_comment.HandleLikeDislike(1, "user_thread_likes"))
				threadsAuth.POST("/dislike/:objID", api_comment.HandleLikeDislike(0, "user_thread_likes"))
				threadsAuth.PATCH("/edit", api_thread.Edit)
				threadsAuth.DELETE("/delete/:threadID", api_thread.Delete)
			}

			threads.GET("/view/:threadID", api_thread.View(1))
			threads.GET("/view/:threadID/:page", api_thread.View(1))
			threads.GET("/list/:rank", api_thread.
				List(1))
			threads.GET("/list/:rank/:page", api_thread.List(1))
			threads.GET("/search/:searchQuery", api_thread.Search(1))
			threads.GET("/search/:searchQuery/:page", api_thread.Search(1))
		}

		comments := v1.Group("/comments")
		{
			commentsAuth := comments.Group("/", middleware.Auth())
			{
				commentsAuth.POST("/new", api_comment.New)
				commentsAuth.PATCH("/edit", api_comment.Edit)
				commentsAuth.DELETE("/delete", api_comment.Delete)
				commentsAuth.POST("/like/:objID", api_comment.HandleLikeDislike(1, "user_comment_likes"))
				commentsAuth.POST("/dislike/:objID", api_comment.HandleLikeDislike(0, "user_comment_likes"))
			}

			comments.GET("/")
		}

	}

	stop := make(chan struct{})
	go func() {
		log.Println("started background task: cleanup expired refresh token every 6 hours")
		cleanupExpiredRefreshToken(database.DB, stop)
	}()

	defer func() {
		close(stop)
		err := database.DB.Close()
		log.Fatalf("failed to close db: %s", err.Error())
	}()

	r.Run()
}

func cleanupExpiredRefreshToken(db *sqlx.DB, stop chan struct{}) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	query := "DELETE FROM refresh_tokens WHERE expiration_date < NOW()"

	for {
		select {
		case <-stop:
			log.Println("[cleanupExpiredRefreshToken] stopping...")
			return
		case <-ticker.C:
			_, err := db.Exec(query)
			if err != nil {
				log.Printf("[cleanupExpiredRefreshToken] failed to cleanup expired refresh token: %s\n", err.Error())
			} else {
				log.Println("[cleanupExpiredRefreshToken] successfully cleaned up expired refresh token")
			}

		}
	}
}
