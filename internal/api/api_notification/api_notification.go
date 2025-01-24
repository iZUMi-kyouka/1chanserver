package api_notification

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetGlobalNotifications() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Status(http.StatusOK)
	}
}
