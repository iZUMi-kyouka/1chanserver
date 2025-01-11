package api_thread

import (
	"1chanserver/internal/models"
	"1chanserver/internal/models/api_error"
	"1chanserver/internal/utils/utils_db"
	"1chanserver/internal/utils/utils_handler"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func New(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)
	tx, err := db.Beginx()
	if err != nil {
		c.Error(err)
		return
	}

	defer utils_db.HandleTxRollback(tx, &err, c)

	var newThread models.ThreadRequest
	err = c.ShouldBindJSON(&newThread)
	if err != nil {
		c.Error(api_error.NewFromStr("invalid object", http.StatusBadRequest))
		return
	}

	threadID, _ := uuid.NewUUID()

	_, err = tx.Exec(
		"INSERT INTO threads(id, user_id, title, original_post, like_count, view_count) VALUES ($1, $2, $3, $4, $5, $6)",
		threadID,
		userID,
		newThread.Title,
		newThread.OriginalPost,
		0,
		0,
	)
	if err != nil {
		c.Error(err)
		return
	}

	for i := 0; i < len(newThread.Tags); i++ {
		_, err = tx.Exec("INSERT INTO thread_tags(thread_id, tag_id) VALUES($1, $2)",
			threadID,
			newThread.Tags[i].Id)
	}

	c.Status(http.StatusCreated)
}

func View(page int) gin.HandlerFunc {
	return func(c *gin.Context) {
		threadID, err := strconv.Atoi(c.Param("threadID"))
		if err != nil {
			c.Error(api_error.NewC(err, http.StatusBadRequest))
			return
		}

		pageInt, err := utils_handler.GetReqPage(c)
		switch err {
		case api_error.InvalidPageReq:
			c.Error(err)
			return
		case api_error.MissingPageReq:
			page = 1
		default:
			page = pageInt
		}

		db := c.MustGet("db").(*sqlx.DB)
		comments, err := utils_db.FetchAll[models.Comment](
			db, "SELECT * FROM comments WHERE thread_id = $1 ORDER BY like_count LIMIT 100 OFFSET $2", threadID, (page-1)*100)

		if err != nil {
			c.Error(err)
			return
		}

		thread, err := utils_db.FetchOne[models.Thread](
			db, "SELECT * FROM threads WHERE id = $1", threadID)
		if err != nil {
			c.Error(err)
			return
		}

		totalComments, err := utils_db.FetchOne[int](
			db, "SELECT COUNT(*) FROM comments WHERE thread_id = $1", threadID)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusOK, models.ThreadViewResponse{
			Thread: thread,
			Comments: models.CommentResponse{
				Comments: comments,
				Pagination: models.Pagination{
					CurrentPage: page,
					LastPage:    totalComments/100 + 1,
					PageSize:    min(len(comments), models.DEFAULT_PAGE_SIZE),
				},
			},
		})
	}
}

func Search() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)

		query := c.Param("searchQuery")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		tagStrings := strings.Split(c.Query("tag"), ",")
		var tags []int
		for _, tagString := range tagStrings {
			tag, err := strconv.Atoi(strings.TrimSpace(tagString))
			if err != nil {
				c.Error(api_error.NewC(err, http.StatusBadRequest))
				return
			}

			tags = append(tags, tag)
		}

		dbQuery := fmt.Sprintf(
			"SELECT id, user_id, title, original_post, creation_date, updated_date, like_count, dislike_count, view_count, ts_rank(search_vector, to_tsquery('english', $1)) AS rank FROM threads WHERE search_vector @@ to_tsquery('english', $1) ORDER BY rank LIMIT $2 OFFSET $3")

		threadList, err := utils_db.FetchAll[models.ThreadSnippet](
			db, dbQuery, query, models.DEFAULT_PAGE_SIZE, (page-1)*100)
		if err != nil {
			c.Error(err)
			return
		}

		threadCount, err := utils_db.FetchOne[int](
			db, "SELECT COUNT(*) FROM "+
				"(SELECT id FROM threads WHERE search_vector @@ to_tsquery('english', $1))", query)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusOK, models.ThreadListResponse{
			Threads: threadList,
			Paginations: models.Pagination{
				CurrentPage: page,
				LastPage:    threadCount/100 + 1,
				PageSize:    min(len(threadList), models.DEFAULT_PAGE_SIZE),
			},
		})
	}
}

func List() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)

		tag := c.Query("tag")
		page := c.Query("page")
		sort_by := c.Query("sort_by")
		order := c.Query("order")

		// Handle page
		var pageInt int
		if page == "" {
			page = "1"
			pageInt = 1
		} else {
			page, err := strconv.Atoi(page)
			pageInt = page
			if err != nil || page < 1 {
				c.Error(api_error.NewFromStr("invalid page", http.StatusBadRequest))
			}
			return
		}

		// Handle sort by
		switch sort_by {
		case "date":
			sort_by = "t.creation_date"
		case "like":
			sort_by = "t.like_count"
		case "dislike":
			sort_by = "t.dislike_count"
		case "view":
			sort_by = "t.view_count"
		default:
			sort_by = "t.view_count"
		}

		// Handle order
		switch order {
		case "asc":
		case "desc":
		default:
			order = "desc"
		}

		var query string

		if tag != "" {
			query = fmt.Sprintf(`
				SELECT DISTINCT
					t.*, u.username
				FROM threads t
				JOIN users u ON t.user_id = u.id
				JOIN thread_tags tt ON t.id = tt.thread_id
				WHERE tt.tag_id IN (%s)
				ORDER BY %s %s
				LIMIT $1 OFFSET $2`, tag, sort_by, order)
		} else {
			query = fmt.Sprintf(`
				SELECT DISTINCT
					t.*, u.username
				FROM threads t
				JOIN users u ON t.user_id = u.id
				JOIN thread_tags tt ON t.id = tt.thread_id
				ORDER BY %s %s
				LIMIT $1 OFFSET $2`, sort_by, order)
		}

		threadList, err := utils_db.FetchAll[models.ThreadSnippet](db, query, models.DEFAULT_PAGE_SIZE, (pageInt-1)*100)
		if err != nil {
			c.Error(api_error.NewC(err, http.StatusInternalServerError))
			return
		}

		for i := 0; i < len(threadList); i++ {
			if len(threadList[i].OriginalPost) > 200 {
				threadList[i].OriginalPost = threadList[i].OriginalPost[:200]
			}
		}

		threadCount, err := utils_db.FetchOne[int](db, "SELECT COUNT(*) FROM threads")
		if err != nil {
			c.Error(api_error.NewC(err, http.StatusInternalServerError))
			return
		}

		c.JSON(http.StatusOK, models.ThreadListResponse{
			Threads: threadList,
			Paginations: models.Pagination{
				CurrentPage: pageInt,
				LastPage:    threadCount/100 + 1,
				PageSize:    min(len(threadList), models.DEFAULT_PAGE_SIZE),
			},
		})
	}
}

func Edit(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)

	editedThread, err := utils_handler.GetObj[models.Thread](c)
	if err != nil {
		c.Error(api_error.NewC(err, http.StatusBadRequest))
		return
	}

	editedThread.UserID = userID
	curTime := time.Now()
	query := "UPDATE threads SET " +
		"title = :title," +
		"original_post = :original_post," +
		"updated_date = :updated_date" +
		"WHERE id = :id AND user_id = :user_id"
	editedThread.UpdatedDate = &curTime

	_, err = db.NamedExec(query, editedThread)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

func Delete(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)
	threadID := c.Param("thread_id")
	if threadID == "" {
		c.Error(api_error.NewFromStr("missing thread id", http.StatusBadRequest))
		return
	}

	_, err := db.Exec(
		"DELETE FROM threads WHERE id = $1 AND user_id $2 ",
		threadID, userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

func Tags(c *gin.Context) {
	db := c.MustGet("db").(*sqlx.DB)

	tags, err := utils_db.FetchAll[models.Tag](db, "SELECT * FROM tags")
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tags": tags,
	})
}
