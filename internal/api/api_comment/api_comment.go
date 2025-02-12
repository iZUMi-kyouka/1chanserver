package api_comment

import (
	"1chanserver/internal/models"
	"1chanserver/internal/models/api_error"
	"1chanserver/internal/utils/utils_db"
	"1chanserver/internal/utils/utils_handler"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
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
	var commentID int

	query := "INSERT INTO comments (thread_id, user_id, comment) VALUES ($1, $2, $3) RETURNING id"
	err = db.QueryRowx(query, threadIDInt, userID, comment).Scan(&commentID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id": commentID,
	})
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

		order := c.DefaultQuery("order", "desc")
		sort_by := c.DefaultQuery("sort_by", "likes")
		sortParamDB, err := utils_db.SortCriteriaToDBColumnWithAlias(sort_by, "c")
		if err != nil {
			c.Error(err)
			return
		}

		if order != "desc" && order != "asc" {
			c.Error(api_error.NewFromStr("invalid order", http.StatusBadRequest))
			return
		}

		threadID := c.Param("threadID")
		if threadID == "" {
			c.Error(api_error.NewFromStr("missing thread id", http.StatusBadRequest))
			return
		}

		page := c.DefaultQuery("page", "1")
		pageInt, err := strconv.Atoi(page)
		if err != nil {
			c.Error(api_error.NewFromStr("invalid page", http.StatusBadRequest))
			return
		}
		//sort := c.DefaultQuery("sort", "likes")

		query := fmt.Sprintf(`
			SELECT 
				c.id, u.username, up.profile_picture_path, c.comment, c.creation_date, 
				c.updated_date, c.like_count, c.dislike_count
			FROM comments c
			JOIN users u ON c.user_id = u.id
			JOIN user_profiles up ON c.user_id = up.id
			WHERE
				c.thread_id = $1
			ORDER BY
				%s %s
			LIMIT $2 OFFSET $3
			`, sortParamDB, order)

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
				PageSize:    min(len(comments), models.DEFAULT_PAGE_SIZE),
				LastPage:    commentsCount/models.DEFAULT_PAGE_SIZE + 1,
			},
		})

	}

}

func Delete(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)
	commentID := c.Param("commentID")

	if commentID == "" {
		c.Error(api_error.NewFromStr("missing comment id", http.StatusBadRequest))
		return
	}

	query := "DELETE FROM comments WHERE id = $1 AND user_id = $2"
	_, err := db.Exec(query, commentID, userID)
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
					//log.Printf("is disliked, now updating to like.")
					query = fmt.Sprintf("UPDATE %s SET variant = 1 WHERE user_id = $1 AND %s = $2", tableName, columnName)
				} else {
					//log.Printf("already disliked, now canceling dislike")
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
					//log.Printf("is disliked, now updating to like.")
					query = fmt.Sprintf("UPDATE %s SET variant = 0 WHERE user_id = $1 AND %s = $2", tableName, columnName)
				} else {
					//log.Printf("already liked, now canceling like.")
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
