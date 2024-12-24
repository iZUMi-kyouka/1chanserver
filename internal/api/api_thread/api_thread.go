package api_thread

import (
	"1chanserver/internal/models"
	"1chanserver/internal/utils/utils_db"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"net/http"
)

func New(c *gin.Context) {
	db := c.MustGet("db").(*sqlx.DB)
	userID := c.MustGet("UserID").(uuid.UUID)

	var newThread models.Thread

	if err := c.ShouldBindJSON(&newThread); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("failed to create a new thread: %s", err.Error())})
		return
	}

	err := utils_db.InsertThread(&newThread, userID, db)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("failed to create a new thread: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": fmt.Sprintf("successfully created a new thread."),
	})
}

func View(c *gin.Context) {

}

func Search(c *gin.Context) {

}

func Edit(c *gin.Context) {

}

func Delete(c *gin.Context) {

}
