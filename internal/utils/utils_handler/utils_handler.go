package utils_handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func GetReqCx(c *gin.Context) (*sqlx.DB, uuid.UUID) {
	return c.MustGet("DB").(*sqlx.DB), c.MustGet("UserID").(uuid.UUID)
}
