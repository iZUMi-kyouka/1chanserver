package middleware

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"runtime/debug"
)

func PanicRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				log.Printf("%s\n", debug.Stack())
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "unexpected server error occurred.",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}
