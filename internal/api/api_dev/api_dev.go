package api_dev

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "API OK",
	})
}

func AuthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "You are authorised.",
	})
}

func ReflectPath(c *gin.Context) {
	required := c.Param("required")
	optional := c.Param("optional")

	c.JSON(http.StatusCreated, gin.H{
		"required": required,
		"optional": optional,
	})
}
