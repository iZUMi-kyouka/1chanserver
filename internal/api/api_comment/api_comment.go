package api_comment

import (
	"1chanserver/internal/utils/utils_handler"
	"github.com/gin-gonic/gin"
)

func New(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)

}
