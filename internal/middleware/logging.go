package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log"
)

func RequestIDProvider() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID, err := uuid.NewUUID()
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}

		ip := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path

		log.Printf("request %s >> %s | %s | %s |",
			requestID, ip, method, path)

		c.Set("RequestID", requestID.String())
	}
}

func ErrorLogging() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Next()

		requestID := c.MustGet("RequestID").(string)
		status := c.Writer.Status()
		ip := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		bg := 42
		if status >= 400 && status <= 599 {
			bg = 41
		}

		bgStart := fmt.Sprintf("\033[1;%dm", bg)
		reset := "\033[0m"

		for i := 0; i < len(c.Errors); i++ {
			log.Printf("request %s >> %s%d%s | %s | %s | %s | \033[1;31merror: %s\033[0m",
				requestID, bgStart, status, reset, ip, method, path, c.Errors[i].Err.Error())
		}
	}
}
