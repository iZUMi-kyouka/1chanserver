package api_user

import (
	"1chanserver/internal/models"
	"1chanserver/internal/models/api_error"
	"1chanserver/internal/utils/utils_auth"
	"1chanserver/internal/utils/utils_db"
	"1chanserver/internal/utils/utils_handler"
	"database/sql"
	"errors"
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
		c.Error(api_error.NewFromErr(err, http.StatusBadRequest))
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
	if err != nil {
		if utils_db.CheckDuplicateError(err) {
			c.Error(api_error.NewFromStr("username already exists", http.StatusConflict))
			return
		} else {
			c.Error(err)
			return
		}
	}

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

	c.SetCookie("Refresh-Token", refreshToken, 3600*24*14, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{
		"uuid": newUser.ID,
		"account": gin.H{
			"id":           newUser.ID,
			"username":     newUser.Username,
			"access_token": accessToken,
		},
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

	defer utils_db.HandleTxRollback(tx, &err, c)

	loginUser, err := utils_handler.GetObj[models.User](c)
	if err != nil {
		c.Error(api_error.NewFromErr(err, http.StatusBadRequest))
		return
	}

	storedUser, err := utils_db.GetUserByUsername(&loginUser.Username, db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.Error(api_error.NewFromStr("invalid username or password", http.StatusUnauthorized))
			return
		}

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

	query := `
	SELECT * FROM user_profiles WHERE id = $1
	`
	userProfile, err := utils_db.FetchOne[models.UserProfile](db, query, storedUser.ID)
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

	c.SetCookie("Refresh-Token", refreshToken, 3600*24*14, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{
		"uuid":     storedUser.ID,
		"username": storedUser.Username,
		"account": gin.H{
			"id":           storedUser.ID,
			"username":     storedUser.Username,
			"access_token": accessToken,
		},
		"refresh_token": refreshToken,
		"profile":       userProfile,
	})
}

func Logout(c *gin.Context) {
	db := c.MustGet("db").(*sqlx.DB)
	refreshToken, err := c.Cookie("Refresh-Token")
	userID := c.MustGet("UserID").(uuid.UUID)
	deviceID := c.GetHeader("Device-ID")

	if err != nil {
		c.Error(api_error.NewFromErr(err, http.StatusBadRequest))
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
	db, userID := utils_handler.GetReqCx(c)

	newProfile, err := utils_handler.GetStringMap(c)
	if err != nil {
		c.Error(api_error.New(err, http.StatusBadRequest, "invalid user profile"))
		return
	}

	query := `
	UPDATE user_profiles 
	SET biodata = $1, email = $2
	WHERE id = $3
	`

	_, err = db.Exec(query, newProfile["biodata"], newProfile["email"], userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
	return
}

func GetProfile(isOwner bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userID uuid.UUID
		var db *sqlx.DB

		if isOwner {
			db, userID = utils_handler.GetReqCx(c)
		} else {
			db = c.MustGet("db").(*sqlx.DB)
		}

		var query string
		var profile models.UserProfile
		var err error

		username := c.Param("username")
		if isOwner {
			query = "SELECT * FROM user_profiles WHERE id = $1"
			profile, err = utils_db.FetchOne[models.UserProfile](
				db, query, userID)
		} else {
			query = "SELECT * FROM user_profiles up, users u WHERE u.id = up.id AND u.username = $1"
			profile, err = utils_db.FetchOne[models.UserProfile](
				db, query, username)
		}

		if err != nil {
			c.Error(err)
			return
		}

		c.JSON(http.StatusOK, profile)
	}
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

func Likes(c *gin.Context) {
	db, userID := utils_handler.GetReqCx(c)
	query := `
		SELECT utl.thread_id, utl.variant
		FROM users u, user_thread_likes utl
		WHERE u.id = $1 AND utl.user_id = u.id
	`

	rows, err := db.Queryx(query, userID)
	if err != nil {
		c.Error(err)
		return
	}

	defer rows.Close()

	likes := make(map[string]map[int]int)
	likes["threads"] = make(map[int]int)
	likes["comments"] = make(map[int]int)

	for rows.Next() {
		var threadID int
		var variant int
		if err := rows.Scan(&threadID, &variant); err != nil {
			c.Error(err)
			return
		}
		likes["threads"][threadID] = variant
	}

	query = `
		SELECT ucl.comment_id, ucl.variant
		FROM users u, user_comment_likes ucl
		WHERE u.id = $1 AND ucl.user_id = u.id
	`

	rows, err = db.Queryx(query, userID)

	for rows.Next() {
		var commentID int
		var variant int
		if err := rows.Scan(&commentID, &variant); err != nil {
			c.Error(err)
			return
		}
		likes["comments"][commentID] = variant
	}

	c.JSON(http.StatusOK, likes)
}

func Threads() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, userID := utils_handler.GetReqCx(c)

		query := `
		SELECT id FROM threads WHERE user_id = $1
		`

		rows, err := db.Queryx(query, userID)
		if err != nil {
			c.Error(err)
			return
		}

		threadIDMap := make(map[int]int)

		for rows.Next() {
			var threadID int
			if err := rows.Scan(&threadID); err != nil {
				c.Error(err)
				return
			}
			threadIDMap[threadID] = 0
		}

		c.JSON(http.StatusOK, threadIDMap)
	}
}

func Comments() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, userID := utils_handler.GetReqCx(c)

		query := `
		SELECT id FROM comments WHERE user_id = $1
		`
		rows, err := db.Queryx(query, userID)
		if err != nil {
			c.Error(err)
			return
		}

		commentIDMap := make(map[int]int)

		for rows.Next() {
			var commentID int
			if err := rows.Scan(&commentID); err != nil {
				c.Error(err)
				return
			}
			commentIDMap[commentID] = 0
		}

		c.JSON(http.StatusOK, commentIDMap)
	}
}

func UpdatePassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, userID := utils_handler.GetReqCx(c)

		var request map[string]string
		if err := c.ShouldBindJSON(&request); err != nil {
			c.Error(api_error.NewFromErr(err, http.StatusBadRequest))
			return
		}

		oldPassword := request["old_password"]
		newPassword := request["new_password"]
		if newPassword == "" {
			c.Error(api_error.NewFromErr(nil, http.StatusBadRequest))
			return
		}

		storedPasswordHash, err := utils_db.FetchOne[string](db, "SELECT password_hash FROM users WHERE id = $1", userID)
		if err != nil {
			c.Error(err)
			return
		}

		if utils_auth.VerifyArgon2Hash(oldPassword, storedPasswordHash) {
			newPasswordHash := utils_auth.GenerateArgon2Hash(newPassword)
			_, err = db.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", newPasswordHash, userID)
			if err != nil {
				c.Error(err)
				return
			}
		} else {
			c.Status(http.StatusForbidden)
			return
		}

		c.Status(http.StatusOK)
	}
}
