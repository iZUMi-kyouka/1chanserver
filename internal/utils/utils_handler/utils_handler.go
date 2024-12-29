package utils_handler

import (
	"1chanserver/internal/models/api_error"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"strconv"
)

func GetReqCx(c *gin.Context) (*sqlx.DB, uuid.UUID) {
	return c.MustGet("DB").(*sqlx.DB), c.MustGet("UserID").(uuid.UUID)
}

func GetObj[T any](c *gin.Context) (T, error) {
	var obj T
	err := c.ShouldBindJSON(&obj)
	return obj, err
}

func GetReqPage(c *gin.Context) (int, error) {
	pageStr := c.Param("page")
	if pageStr == "" {
		return -1, api_error.MissingPageReq
	}

	pageInt, err := strconv.Atoi(pageStr)
	if err != nil {
		return -1, api_error.InvalidPageReq
	}

	return pageInt, nil
}
