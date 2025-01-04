package api_thread

import (
	"1chanserver/internal/models"
	"1chanserver/internal/models/api_error"
	"1chanserver/internal/utils/utils_db"
	"1chanserver/internal/utils/utils_handler"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"net/http"
	"strconv"
	"time"
)

func New(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)

	newThread, err := utils_handler.GetObj[models.Thread](c)
	if err != nil {
		c.Error(api_error.NewFromStr("invalid object", http.StatusBadRequest))
		return
	}

	_, err = db.Exec(
		"INSERT INTO threads(user_id, title, original_post, like_count, view_count) VALUES ($1, $2, $3, $4, $5)",
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

func Search(page int) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)

		searchQuery := c.Param("searchQuery")
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

		query := fmt.Sprintf(
			"SELECT id, user_id, title, original_post, creation_date, updated_date, like_count, dislike_count, view_count, ts_rank(search_vector, to_tsquery('english', $1)) AS rank FROM threads WHERE search_vector @@ to_tsquery('english', $1) ORDER BY rank LIMIT $2 OFFSET $3")

		threadList, err := utils_db.FetchAll[models.Thread](
			db, query, searchQuery, models.DEFAULT_PAGE_SIZE, (page-1)*100)
		if err != nil {
			c.Error(err)
			return
		}

		threadCount, err := utils_db.FetchOne[int](
			db, "SELECT COUNT(*) FROM "+
				"(SELECT id FROM threads WHERE search_vector @@ to_tsquery('english', $1))", searchQuery)
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

func List(page int) gin.HandlerFunc {
	return func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)

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

		var queryOrder string
		switch rank := c.Param("rank"); rank {
		case "top":
			queryOrder = "like_count DESC"
		case "latest":
			queryOrder = "updated_date DESC"
		case "oldest":
			queryOrder = "updated_date ASC"
		case "":
			queryOrder = "like_count DESC"
		default:
			c.Error(api_error.NewC(errors.New("invalid sort parameter"), http.StatusBadRequest))
		}

		query := fmt.Sprintf("SELECT * FROM threads ORDER BY %s LIMIT $1 OFFSET $2", queryOrder)

		threadList, err := utils_db.FetchAll[models.Thread](db, query, models.DEFAULT_PAGE_SIZE, (page-1)*100)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		for i := 0; i < len(threadList); i++ {
			threadList[i].OriginalPost = threadList[i].OriginalPost[:200]
		}

		threadCount, err := utils_db.FetchOne[int](db, "SELECT COUNT(*) FROM threads")
		if err != nil {
			c.Status(http.StatusInternalServerError)
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
