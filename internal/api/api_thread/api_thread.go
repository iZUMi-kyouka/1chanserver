package api_thread

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

	// Ensure all custom tags are valid and present. If not, add it.
	lowerCustomTags := make([]string, len(newThread.CustomTags))
	for i := 0; i < len(newThread.CustomTags); i++ {
		lowerCustomTags[i] = strings.ToLower(newThread.CustomTags[i])
	}

	for i := 0; i < len(lowerCustomTags); i++ {
		if lowerCustomTags[i] == "" || !utils_handler.CheckAllowedSymbols(lowerCustomTags[i]) {
			c.Error(api_error.NewFromStr("invalid custom tag", http.StatusBadRequest))
			return
		}

		customTagAlreadyPresent, err := utils_db.FetchOne[int](db, "SELECT COUNT(*) FROM custom_tags WHERE tag = $1", lowerCustomTags[i])
		if err != nil {
			c.Error(api_error.NewFromStr("failed to check custom tag", http.StatusBadRequest))
			return
		}

		if customTagAlreadyPresent == 0 {
			_, err = tx.Exec("INSERT INTO custom_tags(tag) VALUES($1)", lowerCustomTags[i])
			if err != nil {
				c.Error(err)
				return
			}
		}
	}

	var threadID int
	err = tx.QueryRowx(
		"INSERT INTO threads(user_id, title, original_post, like_count, view_count) VALUES ($1, $2, $3, $4, $5) RETURNING id;",
		userID,
		newThread.Title,
		newThread.OriginalPost,
		0,
		0,
	).Scan(&threadID)
	if err != nil {
		c.Error(err)
		return
	}

	for i := 0; i < len(lowerCustomTags); i++ {
		var customTagID int
		err = tx.Get(&customTagID, "SELECT id FROM custom_tags WHERE tag = $1", lowerCustomTags[i])
		if err != nil {
			c.Error(api_error.NewFromStr("failed to check custom tag", http.StatusBadRequest))
			return
		}

		_, err = tx.Exec("INSERT INTO thread_custom_tags(thread_id, custom_tag_id) VALUES($1, $2)", threadID, customTagID)
		if err != nil {
			c.Error(err)
			return
		}
	}

	for i := 0; i < len(newThread.Tags); i++ {
		_, err = tx.Exec("INSERT INTO thread_tags(thread_id, tag_id) VALUES($1, $2)",
			threadID,
			newThread.Tags[i].Id)
	}

	c.JSON(http.StatusCreated, gin.H{
		"id": threadID,
	})
}

func View(page int) gin.HandlerFunc {
	return func(c *gin.Context) {
		threadID, err := strconv.Atoi(c.Param("threadID"))
		if err != nil {
			c.Error(api_error.NewFromErr(err, http.StatusBadRequest))
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
		comments, err := utils_db.FetchAll[models.CommentView](
			db, "SELECT c.*, u.username, up.profile_picture_path FROM comments c, users u JOIN user_profiles up ON u.id = up.id WHERE thread_id = $1 AND c.user_id = u.id ORDER BY like_count LIMIT $2 OFFSET $3", threadID, models.DEFAULT_PAGE_SIZE, (page-1)*models.DEFAULT_PAGE_SIZE)

		if err != nil {
			c.Error(err)
			return
		}

		threadQuery := `
			SELECT 
				t.*, u.username, up.profile_picture_path,
				array_to_string(array_agg(DISTINCT ct.tag) FILTER (WHERE ct.tag IS NOT NULL), ',') AS custom_tags,
				array_to_string(array_agg(DISTINCT dt.id) FILTER (WHERE dt.id IS NOT NULL), ',') AS tags
			FROM threads t
			JOIN users u ON u.id = t.user_id
			JOIN user_profiles up ON up.id = u.id
			LEFT JOIN thread_custom_tags tct ON tct.thread_id = t.id
			LEFT JOIN custom_tags ct ON ct.id = tct.custom_tag_id
			LEFT JOIN thread_tags tt ON tt.thread_id = t.id
			LEFT JOIN tags dt ON dt.id = tt.tag_id
			WHERE t.id = $1 AND t.user_id = u.id
			GROUP BY t.id, u.username, up.profile_picture_path
			`
		thread, err := utils_db.FetchOne[models.ThreadView](
			db, threadQuery, threadID)
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

		_, err = db.Exec("UPDATE threads SET view_count = view_count + 1 WHERE id = $1", threadID)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusOK, models.ThreadViewResponse{
			Thread: thread,
			Comments: models.PaginatedResponse[models.CommentView]{
				Response: comments,
				Pagination: models.Pagination{
					CurrentPage: page,
					LastPage:    totalComments/models.DEFAULT_PAGE_SIZE + 1,
					PageSize:    min(len(comments), models.DEFAULT_PAGE_SIZE),
				},
			},
		})
	}
}

func Search() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)

		// Get any specified query parameters
		threadReqQuery, err := utils_handler.GetThreadReqQuery(c, "", "relevance")
		if err != nil {
			c.Error(err)
			return
		}

		searchQuery := threadReqQuery["q"]
		whereClauseSearchQuery := func() string {
			if searchQuery != "" {
				return "t.search_vector @@ to_tsquery('english', $3)"
			} else {
				return ""
			}
		}()

		whereClauseSearchCountQuery := func() string {
			if searchQuery != "" {
				return "t.search_vector @@ to_tsquery('english', $1)"
			} else {
				return ""
			}
		}()

		tags := threadReqQuery["tags"].([]string)
		customTags := threadReqQuery["custom_tags"].([]string)
		customTagIDs, err := utils_db.GetCustomTagID(db, customTags)
		if err != nil {
			c.Error(err)
			return
		}

		pageInt := threadReqQuery["page"].(int)
		sortBy := threadReqQuery["sort_by"].(string)
		order := threadReqQuery["order"].(string)

		// Building the WHERE clauses dynamically
		whereClausesMainQuery := []string{}
		whereClausesCountQuery := []string{}
		if whereClauseSearchQuery != "" {
			whereClausesMainQuery = append(whereClausesMainQuery, whereClauseSearchQuery)
		}
		if whereClauseSearchCountQuery != "" {
			whereClausesCountQuery = append(whereClausesCountQuery, whereClauseSearchCountQuery)
		}

		if len(tags) > 0 {
			whereClausesMainQuery = append(whereClausesMainQuery, fmt.Sprintf("tt.tag_id IN %s", utils_db.ToInQueryForm[string](tags)))
			whereClausesCountQuery = append(whereClausesCountQuery, fmt.Sprintf("tt.tag_id IN %s", utils_db.ToInQueryForm[string](tags)))

		}
		if len(customTagIDs) > 0 {
			whereClausesMainQuery = append(whereClausesMainQuery, fmt.Sprintf("tct.custom_tag_id IN %s", utils_db.ToInQueryForm[int](customTagIDs)))
			whereClausesCountQuery = append(whereClausesCountQuery, fmt.Sprintf("tct.custom_tag_id IN %s", utils_db.ToInQueryForm[int](customTagIDs)))
		}

		whereClauseMainQuery := func() string {
			if searchQuery == "" && len(customTagIDs) == 0 && len(tags) == 0 {
				return ""
			} else {
				return fmt.Sprintf("WHERE %s", strings.Join(whereClausesMainQuery, " AND "))
			}
		}()

		whereClauseCountQuery := func() string {
			if searchQuery == "" && len(customTagIDs) == 0 && len(tags) == 0 {
				return ""
			} else {
				return fmt.Sprintf("WHERE %s", strings.Join(whereClausesCountQuery, " AND "))
			}
		}()

		customTagsJoinClause := func() string {
			if len(customTags) > 0 {
				return `
					LEFT JOIN thread_custom_tags tct ON tct.thread_id = t.id
					LEFT JOIN custom_tags ct ON ct.id = tct.custom_tag_id`
			} else {
				return ""
			}
		}()

		tagsJoinClause := func() string {
			if len(tags) > 0 {
				return `
				LEFT JOIN thread_tags tt ON tt.thread_id = t.id
				LEFT JOIN tags dt ON dt.id = tt.tag_id`
			} else {
				return ""
			}
		}()

		rankSelectClause := func() string {
			if searchQuery != "" {
				return "t.id, ts_rank(t.search_vector, to_tsquery('english', $3)) AS rank"
			} else {
				return "t.id"
			}
		}()

		rankGroupByClause := func() string {
			if searchQuery != "" {
				return "GROUP BY t.id, u.username, up.profile_picture_path, rt.rank"
			} else {
				return "GROUP BY t.id, u.username, up.profile_picture_path"
			}
		}()

		// Construct the main query
		query := fmt.Sprintf(`
			WITH ranked_threads AS (
				SELECT %s
				FROM threads t
				JOIN users u ON t.user_id = u.id
				JOIN user_profiles up ON t.user_id = up.id
				%s
				%s
				%s
			)
			SELECT 	
				t.*, 
				u.username, 
				up.profile_picture_path,
				-- Aggregate the tags into arrays, ignoring NULLs
				array_to_string(array_agg(DISTINCT ct.tag) FILTER (WHERE ct.tag IS NOT NULL), ',') AS custom_tags,
				array_to_string(array_agg(DISTINCT dt.id) FILTER (WHERE dt.id IS NOT NULL), ',') AS tags		
			FROM threads t
			JOIN users u ON t.user_id = u.id
			JOIN user_profiles up ON t.user_id = up.id
			LEFT JOIN thread_tags tt ON tt.thread_id = t.id
			LEFT JOIN tags dt ON dt.id = tt.tag_id
			LEFT JOIN thread_custom_tags tct ON tct.thread_id = t.id
			LEFT JOIN custom_tags ct ON ct.id = tct.custom_tag_id
			JOIN ranked_threads rt ON rt.id = t.id
			%s
			ORDER BY %s %s
			LIMIT $1 OFFSET $2
			`,
			rankSelectClause, customTagsJoinClause, tagsJoinClause, whereClauseMainQuery, rankGroupByClause, sortBy, order)
		// Construct the count query
		countQuery := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM (
				SELECT DISTINCT t.id
				FROM threads t
				JOIN users u ON t.user_id = u.id
				JOIN user_profiles up ON t.user_id = up.id
				%s
				%s
				%s
			) AS unique_threads`, customTagsJoinClause, tagsJoinClause, whereClauseCountQuery)

		// Fetch threads based on the query
		log.Println(query)
		log.Println(countQuery)

		threadList, err := func() ([]models.ThreadView, error) {
			if searchQuery != "" {
				return utils_db.FetchAll[models.ThreadView](db, query, models.DEFAULT_PAGE_SIZE, (pageInt-1)*models.DEFAULT_PAGE_SIZE, searchQuery)
			} else {
				return utils_db.FetchAll[models.ThreadView](db, query, models.DEFAULT_PAGE_SIZE, (pageInt-1)*models.DEFAULT_PAGE_SIZE)
			}
		}()

		if err != nil {
			c.Error(api_error.New(err, http.StatusInternalServerError, query))
			return
		}

		// Truncate the post content
		for i := 0; i < len(threadList); i++ {
			if len(threadList[i].OriginalPost) > 200 {
				threadList[i].OriginalPost = threadList[i].OriginalPost[:200] + "..."
			}
		}

		// Get total number of rows for the current query for pagination
		threadCount, err := func() (int, error) {
			if searchQuery != "" {
				return utils_db.FetchOne[int](db, countQuery, searchQuery)
			} else {
				return utils_db.FetchOne[int](db, countQuery)
			}
		}()

		if err != nil {
			c.Error(api_error.New(err, http.StatusInternalServerError, countQuery))
			return
		}

		// Send response
		c.JSON(http.StatusOK, models.PaginatedResponse[models.ThreadView]{
			Response: threadList,
			Pagination: models.Pagination{
				CurrentPage: pageInt,
				LastPage:    threadCount/models.DEFAULT_PAGE_SIZE + 1,
				PageSize:    min(len(threadList), models.DEFAULT_PAGE_SIZE),
			},
		})
	}
}

func List() gin.HandlerFunc {
	return func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)

		// Get any specified query parameters
		threadReqQuery, err := utils_handler.GetThreadReqQuery(c, "t", "views")
		if err != nil {
			c.Error(err)
			return
		}

		tags := threadReqQuery["tags"].([]string)
		customTags := threadReqQuery["custom_tags"].([]string)
		customTagIDs, err := utils_db.GetCustomTagID(db, customTags)
		if err != nil {
			c.Error(err)
			return
		}

		pageInt := threadReqQuery["page"].(int)
		sortBy := threadReqQuery["sort_by"].(string)
		order := threadReqQuery["order"].(string)

		var query string
		var countQuery string

		// Build the query according to listing parameters
		if len(tags) > 0 && len(customTagIDs) > 0 {
			query = fmt.Sprintf(`
				SELECT DISTINCT
					t.*, u.username, up.profile_picture_path
				FROM threads t
				JOIN users u ON t.user_id = u.id
				JOIN user_profiles up ON t.user_id = up.id 
				JOIN thread_tags tt ON t.id = tt.thread_id
				JOIN thread_custom_tags tct ON t.id = tct.thread_id
				WHERE tt.tag_id IN %s AND tct.custom_tag_id IN %s
				ORDER BY %s %s
				LIMIT $1 OFFSET $2`, utils_db.ToInQueryForm[string](tags), utils_db.ToInQueryForm[int](customTagIDs), sortBy, order)
			countQuery = fmt.Sprintf(`
				SELECT COUNT(*)
				FROM threads t
				JOIN users u ON t.user_id = u.id
				JOIN user_profiles up ON t.user_id = up.id 
				JOIN thread_tags tt ON t.id = tt.thread_id
				JOIN thread_custom_tags tct ON t.id = tct.thread_id
				WHERE tt.tag_id IN %s AND tct.custom_tag_id IN %s`, utils_db.ToInQueryForm[string](tags), utils_db.ToInQueryForm[int](customTagIDs))
		} else if len(tags) > 0 && len(customTagIDs) == 0 {
			query = fmt.Sprintf(`
				SELECT DISTINCT
					t.*, u.username, up.profile_picture_path
				FROM threads t
				JOIN users u ON t.user_id = u.id
				JOIN user_profiles up ON t.user_id = up.id 
				JOIN thread_tags tt ON t.id = tt.thread_id
				WHERE tt.tag_id IN %s
				ORDER BY %s %s
				LIMIT $1 OFFSET $2`, utils_db.ToInQueryForm[string](tags), sortBy, order)
			countQuery = fmt.Sprintf(`
				SELECT COUNT(*)
				FROM threads t
				JOIN users u ON t.user_id = u.id
				JOIN user_profiles up ON t.user_id = up.id 
				JOIN thread_tags tt ON t.id = tt.thread_id
				WHERE tt.tag_id IN %s`, utils_db.ToInQueryForm[string](tags))
		} else if len(tags) == 0 && len(customTagIDs) > 0 {
			query = fmt.Sprintf(`
				SELECT DISTINCT
					t.*, u.username, up.profile_picture_path
				FROM threads t
				JOIN users u ON t.user_id = u.id
				JOIN user_profiles up ON t.user_id = up.id 
				JOIN thread_custom_tags tct ON t.id = tct.thread_id
				WHERE tct.custom_tag_id IN %s
				ORDER BY %s %s
				LIMIT $1 OFFSET $2`, utils_db.ToInQueryForm[int](customTagIDs), sortBy, order)
			countQuery = fmt.Sprintf(`
				SELECT COUNT(*)
				FROM threads t
				JOIN users u ON t.user_id = u.id
				JOIN user_profiles up ON t.user_id = up.id 
				JOIN thread_custom_tags tct ON t.id = tct.thread_id
				WHERE tt.tag_id IN %s AND tct.custom_tag_id IN %s`, utils_db.ToInQueryForm[int](customTagIDs))
		} else {
			query = fmt.Sprintf(`
				SELECT DISTINCT
					t.*, u.username,  up.profile_picture_path
				FROM threads t
				JOIN users u ON t.user_id = u.id
				JOIN user_profiles up ON t.user_id = up.id
				ORDER BY %s %s
				LIMIT $1 OFFSET $2`, sortBy, order)
			countQuery = fmt.Sprintf(`
				SELECT COUNT(*)
				FROM threads t
				JOIN users u ON t.user_id = u.id
				JOIN user_profiles up ON t.user_id = up.id`)
		}

		// Fetch threads based on the query
		threadList, err := utils_db.FetchAll[models.ThreadView](db, query, models.DEFAULT_PAGE_SIZE, (pageInt-1)*models.DEFAULT_PAGE_SIZE)
		if err != nil {
			c.Error(api_error.NewFromErr(err, http.StatusInternalServerError))
			return
		}

		// Truncate the post content
		for i := 0; i < len(threadList); i++ {
			if len(threadList[i].OriginalPost) > 200 {
				threadList[i].OriginalPost = threadList[i].OriginalPost[:200] + "..."
			}
		}

		/// Get total number of rows for the current query for pagination
		threadCount, err := utils_db.FetchOne[int](db, countQuery)
		if err != nil {
			c.Error(api_error.NewFromErr(err, http.StatusInternalServerError))
			return
		}

		c.JSON(http.StatusOK, models.PaginatedResponse[models.ThreadView]{
			Response: threadList,
			Pagination: models.Pagination{
				CurrentPage: pageInt,
				LastPage:    threadCount/models.DEFAULT_PAGE_SIZE + 1,
				PageSize:    min(len(threadList), models.DEFAULT_PAGE_SIZE),
			},
		})
	}
}

func Edit(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)

	threadID := c.Param("threadID")
	if threadID == "" {
		c.Error(api_error.NewFromStr("missing thread id", http.StatusBadRequest))
		return
	}

	editedThread, err := utils_handler.GetObj[map[string]string](c)
	if err != nil {
		c.Error(api_error.NewFromErr(err, http.StatusBadRequest))
		return
	}

	query := `
		UPDATE threads 
		SET title = $1, original_post = $2, updated_date = $3 
		WHERE id = $4 AND user_id = $5
	`
	_, err = db.Exec(query, editedThread["title"], editedThread["original_post"], time.Now().UTC(), threadID, userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

func Delete(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)
	threadID := c.Param("threadID")
	if threadID == "" {
		c.Error(api_error.NewFromStr("missing thread id", http.StatusBadRequest))
		return
	}

	_, err := db.Exec(
		"DELETE FROM threads WHERE id = $1 AND user_id = $2",
		threadID, userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

func Report(objectType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		db, userID := utils_handler.GetReqCx(c)
		objID := c.Param("objID")
		objIDInt, err := strconv.Atoi(objID)
		if err != nil {
			c.Error(api_error.NewFromStr("invalid object id", http.StatusBadRequest))
			return
		}

		report, err := utils_handler.GetStringMap(c)
		if err != nil {
			c.Error(api_error.NewFromStr("missing report body", http.StatusBadRequest))
			return
		}

		var query string
		switch objectType {
		case "thread":
			query = `
			INSERT INTO reports(thread_id, reporter_id, report_reason) VALUES($1, $2, $3) RETURNING id
			`
		case "comment":
			query = `
			INSERT INTO reports(comment_id, reporter_id, report_reason) VALUES($1, $2, $3) RETURNING id
			`
		}

		var reportID int
		err = db.QueryRowx(query, objIDInt, userID, report["report_reason"]).Scan(&reportID)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"report_id": reportID,
		})
	}
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

func CreateTag(c *gin.Context) {
	db := c.MustGet("db").(*sqlx.DB)
	tag := c.Query("tag")
	if tag == "" {
		c.Error(api_error.NewFromStr("missing tag", http.StatusBadRequest))
		return
	}

	count, err := utils_db.GetTotalRecordNo(db, "SELECT COUNT(*) FROM tags WHERE tag = $1", tag)
	if err != nil {
		c.Error(err)
		return
	}
	if count != 0 {
		c.Error(api_error.NewFromStr("tag already exists", http.StatusConflict))
		return
	}

	_, err = db.Exec("INSERT INTO tags(tag) VALUES($1)", tag)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusCreated)
}
