package api_comment

import (
	"1chanserver/internal/models"
	"1chanserver/internal/models/api_error"
	"1chanserver/internal/utils/utils_db"
	"1chanserver/internal/utils/utils_handler"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
	"strconv"
	"time"
)

func New(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)
	threadID := c.Param("threadID")
	if threadID == "" {
		c.Error(api_error.NewFromStr("missing thread id", http.StatusBadRequest))
		return
	}

	threadIDInt, err := strconv.Atoi(threadID)
	if err != nil {
		c.Error(api_error.NewFromStr("invalid thread id", http.StatusBadRequest))
	}

	var commentRequest map[string]string
	err = c.ShouldBindJSON(&commentRequest)
	if err != nil {
		c.Error(api_error.NewFromStr("invalid obj", http.StatusBadRequest))
		return
	}

	comment := commentRequest["comment"]

	query := "INSERT INTO comments (thread_id, user_id, comment) VALUES ($1, $2, $3)"
	_, err = db.Exec(query, threadIDInt, userID, comment)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusCreated)
}

func Edit(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)
	commentID := c.Param("commentID")
	if commentID == "" {
		c.Error(api_error.NewFromStr("missing comment id", http.StatusBadRequest))
		return
	}

	comment, err := utils_handler.GetObj[map[string]string](c)
	if err != nil {
		c.Error(api_error.NewFromStr("invalid obj", http.StatusBadRequest))
		return
	}

	query := `
	UPDATE comments
	SET comment = $1, updated_date = $2
	WHERE id = $3 AND user_id = $4 
	`

	_, err = db.Exec(query, comment["comment"], time.Now().UTC(), commentID, userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

func List() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)

		threadID := c.Param("threadID")
		if threadID == "" {
			c.Error(api_error.NewFromStr("missing thread id", http.StatusBadRequest))
			return
		}

		page := c.DefaultQuery("p", "1")
		pageInt, err := strconv.Atoi(page)
		if err != nil {
			c.Error(api_error.NewFromStr("invalid page", http.StatusBadRequest))
			return
		}
		//sort := c.DefaultQuery("sort", "likes")

		query := `
			SELECT 
				c.id, u.username, c.comment, c.creation_date, 
				c.updated_date, c.like_count, c.dislike_count
			FROM users u, comments c
			WHERE
				c.thread_id = $1 AND c.user_id = u.id
			ORDER BY
				c.like_count DESC
			LIMIT $2 OFFSET $3
			`

		comments, err := utils_db.FetchAll[models.CommentView](db, query,
			threadID, models.DEFAULT_PAGE_SIZE, (pageInt-1)*models.DEFAULT_PAGE_SIZE)
		if err != nil {
			c.Error(err)
			return
		}

		commentsCount, err := utils_db.GetTotalRecordNo(db, "SELECT COUNT(*) FROM comments WHERE thread_id = $1", threadID)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusOK, models.PaginatedResponse[models.CommentView]{
			Response: comments,
			Pagination: models.Pagination{
				CurrentPage: pageInt,
				PageSize:    models.DEFAULT_PAGE_SIZE,
				LastPage:    commentsCount,
			},
		})

	}

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
			c.Error(api_error.NewFromErr(err, http.StatusBadRequest))
			return
		}

		var columnName string
		switch tableName {
		case "user_thread_likes":
			columnName = "thread_id"
		case "user_comment_likes":
			columnName = "comment_id"
		}

		isLiked, err := utils_db.FetchOne[int](
			db,
			fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE user_id = $1 AND %s = $2", tableName, columnName),
			userID, objID)

		switch isLiked {
		case 0:
			_, err := db.Exec(fmt.Sprintf(
				"INSERT INTO %s (user_id, %s, variant) VALUES ($1, $2, $3)", tableName, columnName),
				userID, objID, v)
			if err != nil {
				c.Error(err)
				return
			}
		case 1:
			likeVariant, err := utils_db.FetchOne[int](db,
				fmt.Sprintf("SELECT variant FROM %s WHERE user_id = $1 AND %s = $2", tableName, columnName),
				userID, objID)
			if err != nil {
				c.Error(err)
				return
			}

			switch likeVariant {
			case 0:
				var query string
				if v == 1 {
					log.Printf("is disliked, now updating to like.")
					query = fmt.Sprintf("UPDATE %s SET variant = 1 WHERE user_id = $1 AND %s = $2", tableName, columnName)
				} else {
					log.Printf("already disliked, now canceling dislike")
					query = fmt.Sprintf("DELETE FROM %s WHERE user_id = $1 AND %s = $2", tableName, columnName)
				}

				_, err := db.Exec(query, userID, objID)
				if err != nil {
					c.Error(err)
					return
				}

			case 1:
				var query string
				if v == 0 {
					log.Printf("is disliked, now updating to like.")
					query = fmt.Sprintf("UPDATE %s SET variant = 0 WHERE user_id = $1 AND %s = $2", tableName, columnName)
				} else {
					log.Printf("already liked, now canceling like.")
					query = fmt.Sprintf("DELETE FROM %s WHERE user_id = $1 AND %s = $2", tableName, columnName)
				}

				_, err := db.Exec(query, userID, objID)
				if err != nil {
					c.Error(err)
					return
				}
			}
		}

		c.Status(http.StatusOK)
	}
}

func View() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)
		commentID := c.Param("commentID")
		if commentID == "" {
			c.Error(api_error.NewFromStr("missing comment id", http.StatusBadRequest))
			return
		}

		query := `
		SELECT * FROM comments WHERE id = $1
		`

		comment, err := utils_db.FetchOne[models.CommentView](db, query, commentID)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusOK, comment)
	}
}
