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
	"github.com/gin-gonic/gin"
	"log"
)

func main() {
	fmt.Println("Starting server...")
	database.InitDB()

	r := gin.Default()
	r.Use(
		middleware.PanicRecovery(),
		middleware.RequestIDProvider(),
		middleware.CORS(),
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
			usersAuth := users.Group("/")
			{
				usersAuth.GET("/placeholder", func(c *gin.Context) {
					c.Status(200)
				})
			}

			users.POST("/login", api_user.Login)
			users.POST("/register", api_user.Register)
			users.GET("/refresh", api_token.RefreshToken())
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
			threads.GET("/list/:rank", api_thread.List(1))
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

	defer func() {
		err := database.DB.Close()
		log.Fatalf("failed to close db: %s", err.Error())
	}()

	r.Run()
}
