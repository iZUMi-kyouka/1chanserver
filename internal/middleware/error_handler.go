package middleware

import (
	"1chanserver/internal/models/api_error"
	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors[0]
			api_error.ToResponse(c, err.Err)
		}
	}
}
