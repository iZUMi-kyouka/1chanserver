package api_files

import (
	"1chanserver/internal/models/api_error"
	"1chanserver/internal/routes"
	"1chanserver/internal/utils/utils_handler"
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
			c.Error(api_error.NewFromErr(err, http.StatusBadRequest))
			return
		}

		filenameSplits := strings.Split(file.Filename, ".")
		fileName := uuid.New().String()

		if len(filenameSplits) == 1 {
			c.Error(api_error.NewFromStr("invalid image", http.StatusBadRequest))
			return
		}

		fileFormat := filenameSplits[len(filenameSplits)-1]
		filePath := fmt.Sprintf("./public/uploads/%s.%s", fileName, fileFormat)
		err = c.SaveUploadedFile(file, filePath)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"url": fmt.Sprintf("%s%s/%s.%s", routes.BaseURL, "/files", fileName, fileFormat),
		})
	}
}

func UploadProfilePicture() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, userID := utils_handler.GetReqCx(c)

		file, err := c.FormFile("profile_picture")
		if err != nil {
			c.Error(api_error.NewFromErr(err, http.StatusBadRequest))
			return
		}

		filenameSplits := strings.Split(file.Filename, ".")
		fileName := uuid.New().String()

		if len(filenameSplits) == 1 {
			c.Error(api_error.NewFromStr("invalid image", http.StatusBadRequest))
			return
		}

		fileFormat := filenameSplits[len(filenameSplits)-1]
		filePath := fmt.Sprintf("./public/uploads/profile_pictures/%s.%s", fileName, fileFormat)

		err = c.SaveUploadedFile(file, filePath)
		if err != nil {
			c.Error(err)
			return
		}

		query := `
		UPDATE user_profiles
		SET profile_picture_path = $1
		WHERE id = $2
		`

		_, err = db.Exec(query, fmt.Sprintf("%s.%s", fileName, fileFormat), userID)
		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"url": fmt.Sprintf("%s.%s", fileName, fileFormat),
		})
	}
}
