package api_files

import (
	"1chanserver/internal/models/api_error"
	"1chanserver/internal/routes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

func Upload(category string) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile(category)
		if err != nil {
			c.Error(api_error.NewC(err, http.StatusBadRequest))
			return
		}

		filenameSplits := strings.Split(file.Filename, ".")
		fileName := uuid.New().String()
		fileFormat := filenameSplits[len(filenameSplits)-1]
		filePath := fmt.Sprintf("./public/uploads/%s.%s", fileName, fileFormat)
		c.SaveUploadedFile(file, filePath)

		c.JSON(http.StatusOK, gin.H{
			"url": fmt.Sprintf("%s%s/%s.%s", routes.BaseURL, "/files", fileName, fileFormat),
		})
	}
}
