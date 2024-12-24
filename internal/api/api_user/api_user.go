package api_user

import (
	"1chanserver/internal/models"
	"1chanserver/internal/utils/utils_auth"
	"1chanserver/internal/utils/utils_db"
	"fmt"
	"github.com/jmoiron/sqlx"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"net/http"
)

func Register(c *gin.Context) {
	db := c.MustGet("db").(*sqlx.DB)

	var newUser models.User

	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error()})
		return
	}

	newUser.ID = uuid.New()
	newUser.Password = utils_auth.GenerateArgon2Hash(newUser.Password)

	log.Println("New user registered with uuid:", newUser.ID)
	err := utils_db.InsertUser(&newUser, db)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("error creating new user with uuid %s: %s", newUser.ID, err.Error())})
		return
	}

	// Generate access and refresh tokens
	accessToken, err := utils_auth.GenerateAccessToken(newUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("error creating access token for user %s: %s", newUser.ID, err.Error())})
		return
	}

	refreshToken, err := utils_auth.GenerateRefreshToken(newUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("error creating refresh token for user %s: %s", newUser.ID, err.Error())})
		return
	}

	hashedRefreshToken := utils_auth.HashRefreshToken(refreshToken)

	log.Printf(
		"Inserting refresh token %s for user %s at time %s",
		hashedRefreshToken,
		newUser.ID,
		time.Now().Format("2006-01-02 15:04:05"))

	err = utils_db.InsertRefreshToken(&newUser, hashedRefreshToken, time.Now().Add(utils_auth.JWT_REFRESH_TOKEN_EXPIRATION), db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to store the generated refresh token: %s", err.Error()),
		})
		return
	}

	utils_auth.SetAccessAndRefreshToken(c, refreshToken, accessToken)

	c.JSON(http.StatusOK, gin.H{
		"uuid":          newUser.ID,
		"username":      newUser.Username,
		"refresh_token": refreshToken,
		"access_token":  accessToken,
	})
}

func Login(c *gin.Context) {
	db := c.MustGet("db").(*sqlx.DB)

	var loginUser models.User

	if err := c.ShouldBindJSON(&loginUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error()})
		return
	}

	storedUser, err := utils_db.GetUserByUsername(&loginUser.Username, db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	inputPassword := loginUser.Password

	ok := utils_auth.VerifyArgon2Hash(inputPassword, storedUser.Password)

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "invalid username or password",
		})
	}

	accessToken, err := utils_auth.GenerateAccessToken(loginUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error()})
		return
	}

	refreshToken, err := utils_auth.GenerateRefreshToken(loginUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error()})
		return
	}

	hashedRefreshToken := utils_auth.HashRefreshToken(refreshToken)

	err = utils_db.InsertRefreshToken(&loginUser, hashedRefreshToken, time.Now().Add(utils_auth.JWT_REFRESH_TOKEN_EXPIRATION), db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "failed to store the generated refresh token.",
		})
	}

	utils_auth.SetAccessAndRefreshToken(c, refreshToken, accessToken)

	c.JSON(http.StatusOK, gin.H{
		"uuid":          storedUser.ID,
		"username":      storedUser.Username,
		"refresh_token": refreshToken,
		"access_token":  accessToken,
	})
}

func UpdateProfile(c *gin.Context) {

}
