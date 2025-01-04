package api_user

import (
	"1chanserver/internal/models"
	"1chanserver/internal/models/api_error"
	"1chanserver/internal/utils/utils_auth"
	"1chanserver/internal/utils/utils_db"
	"1chanserver/internal/utils/utils_handler"
	"github.com/jmoiron/sqlx"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"net/http"
)

func Register(c *gin.Context) {
	db := c.MustGet("db").(*sqlx.DB)

	deviceID := c.GetHeader("Device-ID")
	if deviceID == "" {
		c.Error(api_error.NewFromStr("missing device ID", http.StatusBadRequest))
		return
	}

	newUser, err := utils_handler.GetObj[models.User](c)
	if err != nil {
		c.Error(api_error.NewC(err, http.StatusBadRequest))
		return
	}

	newUser.ID = uuid.New()
	newUser.Password = utils_auth.GenerateArgon2Hash(newUser.Password)

	_, err = db.Exec(
		"INSERT INTO users(id, username, password_hash) VALUES ($1, $2, $3)",
		newUser.ID,
		newUser.Username,
		newUser.Password,
	)

	curTime := time.Now().UTC()

	_, err = db.Exec(
		"INSERT INTO user_profiles(id, creation_date) VALUES ($1, $2)",
		newUser.ID,
		curTime)

	if err != nil {
		c.Error(err)
		return
	}

	// Generate access and refresh tokens
	accessToken, err := utils_auth.GenerateAccessToken(newUser.ID)
	if err != nil {
		c.Error(err)
		return
	}

	refreshToken, err := utils_auth.GenerateRefreshToken(newUser.ID)
	if err != nil {
		c.Error(err)
		return
	}

	hashedRefreshToken := utils_auth.HashRefreshToken(refreshToken)

	log.Printf(
		"Inserting refresh token %s for user %s at time %s",
		hashedRefreshToken,
		newUser.ID,
		time.Now().UTC().Format("2006-01-02 15:04:05"))

	err = utils_db.InsertRefreshToken(&newUser, hashedRefreshToken, time.Now().UTC().Add(utils_auth.JWT_REFRESH_TOKEN_EXPIRATION), deviceID, db)
	if err != nil {
		c.Error(err)
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
	deviceID := c.GetHeader("Device-ID")

	if deviceID == "" {
		c.Error(api_error.NewFromStr("missing device ID", http.StatusBadRequest))
		return
	}

	tx, err := db.Beginx()
	if err != nil {
		c.Error(err)
		return
	}

	defer func() {
		if p := recover(); p != nil {
			err := tx.Rollback()
			if err != nil {
				log.Fatalf("failed to rollback db: %s", err.Error())
			}
			panic(p)
		} else if err != nil {
			err := tx.Rollback()
			if err != nil {
				log.Fatalf("failed to rollback db: %s", err.Error())
			}
		} else {
			err = tx.Commit()
			if err != nil {
				c.Error(err)
			}
		}
	}()

	loginUser, err := utils_handler.GetObj[models.User](c)
	if err != nil {
		c.Error(api_error.NewC(err, http.StatusBadRequest))
		return
	}

	storedUser, err := utils_db.GetUserByUsername(&loginUser.Username, db)
	if err != nil {
		c.Error(err)
		return
	}

	inputPassword := loginUser.Password
	ok := utils_auth.VerifyArgon2Hash(inputPassword, storedUser.Password)

	if !ok {
		c.Error(api_error.New(err, http.StatusUnauthorized, "invalid username or password"))
		return
	}

	accessToken, err := utils_auth.GenerateAccessToken(storedUser.ID)
	if err != nil {
		c.Error(err)
		return
	}

	var refreshToken string
	var hasRefreshToken int
	err = db.Get(&hasRefreshToken,
		"SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1 AND device_id = $2",
		storedUser.ID, deviceID)
	if err != nil {
		c.Error(err)
		return
	}

	if hasRefreshToken != 0 {
		_, err = tx.Exec(
			"DELETE FROM refresh_tokens WHERE user_id = $1 AND device_id = $2",
			storedUser.ID, deviceID)

		if err != nil {
			c.Error(err)
			return
		}
	}

	refreshToken, err = utils_auth.GenerateRefreshToken(storedUser.ID)
	if err != nil {
		c.Error(err)
		return
	}

	hashedRefreshToken := utils_auth.HashRefreshToken(refreshToken)

	_, err = tx.Exec(
		"INSERT INTO refresh_tokens(user_id, token_hash, expiration_date, device_id) VALUES ($1, $2, $3, $4)",
		storedUser.ID,
		hashedRefreshToken,
		time.Now().UTC().Add(utils_auth.JWT_REFRESH_TOKEN_EXPIRATION), deviceID)
	if err != nil {
		c.Error(err)
		return
	}

	_, err = tx.Exec("UPDATE user_profiles SET last_login = $1 WHERE id = $2",
		time.Now().UTC(), storedUser.ID)
	if err != nil {
		c.Error(err)
		return
	}

	c.SetCookie("Refresh-Token", refreshToken, 3600*24*14, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"uuid":     storedUser.ID,
		"username": storedUser.Username,
		"account": gin.H{
			"id":           storedUser.ID,
			"username":     storedUser.Username,
			"access_token": accessToken,
		},
		"refresh_token": refreshToken,
	})
}

func Logout(c *gin.Context) {
	db := c.MustGet("db").(*sqlx.DB)
	refreshToken, err := c.Cookie("Refresh-Token")
	userID := c.MustGet("UserID").(uuid.UUID)
	deviceID := c.GetHeader("Device-ID")

	if err != nil {
		c.Error(api_error.NewC(err, http.StatusBadRequest))
		c.Abort()
		return
	}

	_, err = db.Exec("DELETE FROM refresh_tokens WHERE token_hash = $1 AND user_id = $2 AND device_id = $3",
		refreshToken, userID, deviceID)
	if err != nil {
		c.Error(err)
		c.Abort()
		return
	}

	c.SetCookie("Refresh-Token", "", 0, "/", "*", false, true)
	c.Status(http.StatusOK)
}

func UpdateProfile(c *gin.Context) {
	db := c.MustGet("db").(*sqlx.DB)
	var userProfile models.UserProfile

	err := c.ShouldBindJSON(&userProfile)
	if err != nil {
		c.Error(api_error.New(err, http.StatusBadRequest, "invalid user profile"))
		return
	}

	err = utils_db.EditUserProfile(&userProfile, db)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
	return
}

func Delete(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)
	err := utils_db.DeleteUser(&userID, db)

	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
	return
}
