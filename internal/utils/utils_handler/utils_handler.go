package utils_handler

import (
	"1chanserver/internal/models/api_error"
	"1chanserver/internal/utils/utils_db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"net/http"
	"strconv"
	"strings"
	"unicode"
)

func GetReqCx(c *gin.Context) (*sqlx.DB, uuid.UUID) {
	return c.MustGet("db").(*sqlx.DB), c.MustGet("UserID").(uuid.UUID)
}

func GetObj[T any](c *gin.Context) (T, error) {
	var obj T
	err := c.ShouldBindJSON(&obj)
	return obj, err
}

func GetStringMap(c *gin.Context) (map[string]string, error) {
	var obj map[string]string
	obj = make(map[string]string)
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

func CheckAllowedSymbols(s string) bool {
	for _, char := range s {
		if unicode.IsSymbol(char) {
			if char != '_' {
				return false
			}
		}
	}

	return true
}

func GetThreadReqQuery(c *gin.Context, tableAlias string, defaultSortCriteria string) (map[string]interface{}, error) {
	page := c.DefaultQuery("page", "1")
	tags := c.Query("tags")
	customTags := c.Query("custom_tags")
	order := c.DefaultQuery("order", "desc")
	q := c.Query("q")

	reqQuery := make(map[string]interface{})

	reqQuery["q"] = q
	if q == "" {
		defaultSortCriteria = "views"
	}

	sort_by := c.DefaultQuery("sort_by", defaultSortCriteria)

	// Handle page
	var pageInt int
	pageInt, err := strconv.Atoi(page)
	if err != nil || pageInt <= 0 {
		c.Error(api_error.NewFromStr("invalid page", http.StatusBadRequest))
		return reqQuery, api_error.InvalidPageReq
	}

	reqQuery["page"] = pageInt

	// tag here is an array of tag ID
	tagsArray := strings.Split(tags, ",")
	if len(tagsArray) == 1 && tagsArray[0] == "" {
		reqQuery["tags"] = make([]string, 0)
	} else {
		reqQuery["tags"] = tagsArray
	}

	// custom tag here is an array of custom tag name
	customTagsArray := strings.Split(customTags, ",")
	if len(customTagsArray) == 1 && customTagsArray[0] == "" {
		reqQuery["custom_tags"] = make([]string, 0)
	} else {
		reqQuery["custom_tags"] = customTagsArray
	}

	// sort_by is the db column name for the corresponding sort parameter
	reqQuery["sort_by"], err = utils_db.SortCriteriaToDBColumnWithAlias(sort_by, tableAlias)
	if err != nil {
		return reqQuery, err
	}

	switch order {
	case "asc":
		reqQuery["order"] = "asc"
	case "desc":
		reqQuery["order"] = "desc"
	default:
		return reqQuery, api_error.NewFromStr("invalid order parameter", http.StatusBadRequest)
	}

	return reqQuery, nil
}
