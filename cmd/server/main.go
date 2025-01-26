package main

import (
	"1chanserver/internal/api/api_comment"
	"1chanserver/internal/api/api_dev"
	"1chanserver/internal/api/api_files"
	"1chanserver/internal/api/api_notification"
	"1chanserver/internal/api/api_thread"
	"1chanserver/internal/api/api_token"
	"1chanserver/internal/api/api_user"
	"1chanserver/internal/database"
	_ "1chanserver/internal/database"
	"1chanserver/internal/middleware"
	"1chanserver/internal/models/api_error"
	"1chanserver/internal/routes"
	"1chanserver/internal/utils/utils_auth"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"log"
	"os"
	"time"
)

func main() {

	fmt.Println("Starting server...")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	routes.BaseAPI = os.Getenv("BASE_API")
	routes.BaseURL = os.Getenv("BASE_URL")
	routes.APIRoot = routes.BaseURL + routes.BaseAPI

	secureCookieEnabled := os.Getenv("SECURE_COOKIE")
	if secureCookieEnabled == "true" {
		api_token.SecureCookieEnabled = true
	} else if secureCookieEnabled == "false" {
		api_token.SecureCookieEnabled = false
	} else {
		panic("invalid SECURE_COOKIE environment variable")
	}

	utils_auth.JWT_SECRET_KEY = []byte(os.Getenv("JWT_SECRET_KEY"))

	// Initialise database
	database.InitDB()

	// Use middlewares
	r := gin.Default()
	r.Use(middleware.CORS())
	r.Use(
		middleware.PanicRecovery(),
		middleware.RequestIDProvider(),
		middleware.ErrorLogging(),
		middleware.ErrorHandler())
	r.Use(middleware.DBProvider(database.DB))

	// Use routes
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

		// User routes
		users := v1.Group("/users")
		{
			usersAuth := users.Group("/", middleware.Auth())
			{
				usersAuth.GET("/logout", api_user.Logout)
				usersAuth.GET("/likes", api_user.Likes)
				usersAuth.GET("/profile", api_user.GetProfile(true))
				usersAuth.POST("/profile", api_user.UpdateProfile)
				usersAuth.GET("/threads", api_user.Threads())
				usersAuth.GET("/comments", api_user.Comments())
				usersAuth.POST("/update_password", api_user.UpdatePassword())
				usersAuth.POST("/profile/picture", api_files.UploadProfilePicture())
			}

			users.POST("/login", api_user.Login)
			users.POST("/register", api_user.Register)
			users.GET("/profile/:username", api_user.GetProfile(false))
			users.GET("/refresh_new", api_token.RefreshToken("first"))
			users.GET("/refresh", api_token.RefreshToken("continue"))
		}

		// Thread routes
		threads := v1.Group("/threads")
		{
			threadsAuth := threads.Group("/", middleware.Auth())
			{
				threadsAuth.POST("/new", api_thread.New)
				threadsAuth.PUT("/like/:objID", api_comment.HandleLikeDislike(1, "user_thread_likes"))
				threadsAuth.PUT("/dislike/:objID", api_comment.HandleLikeDislike(0, "user_thread_likes"))
				threadsAuth.PATCH("/edit/:threadID", api_thread.Edit)
				threadsAuth.DELETE("/:threadID", api_thread.Delete)
				threadsAuth.POST("/report/:objID", api_thread.Report("thread"))
			}

			threads.GET("/:threadID", api_thread.View(1))
			threads.GET("/:threadID/:page", api_thread.View(1))
			threads.GET("/list", api_thread.List())
			threads.GET("/search", api_thread.Search())
			threads.GET("/tags", api_thread.Tags)

		}

		// Comment routes
		comments := v1.Group("/comments")
		{
			commentsAuth := comments.Group("/", middleware.Auth())
			{
				commentsAuth.POST("/new/:threadID", api_comment.New)
				commentsAuth.PATCH("/edit/:commentID", api_comment.Edit)
				commentsAuth.DELETE("/:commentID", api_comment.Delete)
				commentsAuth.PUT("/like/:objID", api_comment.HandleLikeDislike(1, "user_comment_likes"))
				commentsAuth.PUT("/dislike/:objID", api_comment.HandleLikeDislike(0, "user_comment_likes"))
				commentsAuth.POST("/report/:objID", api_thread.Report("comment"))
			}

			comments.GET("/:commentID", api_comment.View())
			comments.GET("/thread/:threadID", api_comment.List())
		}

		// Tags route
		tags := v1.Group("/tags")
		{
			tagsAuth := tags.Group("/", middleware.Auth())
			{
				tagsAuth.POST("/new", api_thread.CreateTag)
			}
		}

		// NOT FULLY IMPLEMENTED: Notifications route
		notifications := v1.Group("/notifications")
		{
			notifications.GET("/notifications", api_notification.GetGlobalNotifications())
		}

		// File upload route
		upload := v1.Group("/upload")
		{
			//uploadAuth := upload.Group("/", middleware.Auth())
			upload.POST("/image", api_files.Upload("image"))

		}

		// File server routes
		r.Static("/files", "./public/uploads")
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
