package api_comment

import (
	"1chanserver/internal/models"
	"1chanserver/internal/models/api_error"
	"1chanserver/internal/utils/utils_db"
	"1chanserver/internal/utils/utils_handler"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

func New(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)
	comment, err := utils_handler.GetObj[models.Comment](c)
	if err != nil {
		c.Error(api_error.NewFromStr("invalid obj", http.StatusBadRequest))
		return
	}

	comment.UserID = userID
	query := "INSERT INTO comments (thread_id, user_id, comment) VALUES (:thread_id, :user_id, :comment))"
	_, err = db.NamedExec(query, comment)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusCreated)
}

func Edit(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)
	comment, err := utils_handler.GetObj[models.Comment](c)
	if err != nil {
		c.Error(api_error.NewFromStr("invalid obj", http.StatusBadRequest))
		return
	}

	comment.UserID = userID
	comment.UpdatedDate = time.Now()
	query := "UPDATE comments SET " +
		"comment = :comment, " +
		"updated_date = :updated_date " +
		"WHERE id = :id AND user_id = :user_id"
	_, err = db.NamedExec(query, comment)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

func Delete(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)
	comment, err := utils_handler.GetObj[models.Comment](c)
	if err != nil {
		c.Error(api_error.NewFromStr("invalid obj", http.StatusBadRequest))
		return
	}

	comment.UserID = userID
	query := "DELETE FROM comments WHERE id = :id AND user_id = :user_id"
	_, err = db.NamedExec(query, comment)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

func HandleLikeDislike(v int, tableName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		db, userID := utils_handler.GetReqCx(c)
		objID, err := strconv.Atoi(c.Param("objID"))
		if err != nil {
			c.Error(api_error.NewC(err, http.StatusBadRequest))
			return
		}

		isLiked, err := utils_db.FetchOne[int](
			db,
			fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE user_id = $1 AND comment_id = $2", tableName),
			userID, objID)

		switch isLiked {
		case 0:
			_, err := db.Exec(fmt.Sprintf("INSERT INTO %s (user_id, comment_id, variant) VALUES ($1, $2, $3)", tableName),
				userID, objID, v)
			if err != nil {
				c.Error(err)
				return
			}
		case 1:
			likeVariant, err := utils_db.FetchOne[int](db,
				fmt.Sprintf("SELECT variant FROM %s WHERE user_id = $1 AND comment_id = $2", tableName),
				userID, objID)
			if err != nil {
				c.Error(err)
				return
			}

			switch likeVariant {
			case 0:
				_, err := db.Exec(
					fmt.Sprintf("INSERT INTO %s (user_id, comment_id, variant) VALUES ($1, $2, $3)", tableName),
					userID, objID, 1-v)
				if err != nil {
					c.Error(err)
					return
				}
			case 1:
				_, err := db.Exec(
					fmt.Sprintf("DELETE FROM %s WHERE user_id = $1 AND comment_id = $2", tableName), userID, objID)
				if err != nil {
					c.Error(err)
					return
				}
			}
		}

		c.Status(http.StatusOK)
	}
}
