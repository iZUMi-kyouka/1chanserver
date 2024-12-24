package utils_db

import (
	"1chanserver/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"time"
)

func GetUserByUsername(username *string, db *sqlx.DB) (models.User, error) {
	var user models.User
	err := db.Get(&user, "SELECT * FROM users WHERE username = $1", *username)
	return user, err
}

func GetUserByUserID(userID *uuid.UUID, db *sqlx.DB) (models.User, error) {
	var user models.User
	err := db.Get(&user, "SELECT * FROM users WHERE id = $1", (*userID).String())
	return user, err
}

func InsertUser(user *models.User, db *sqlx.DB) error {
	_, err := db.Exec(
		"INSERT INTO users(id, username, password_hash) VALUES ($1, $2, $3)",
		user.ID,
		user.Username,
		user.Password,
	)

	curTime := time.Now()

	_, err = db.Exec(
		"INSERT INTO user_profiles(id, creation_date, last_login) VALUES ($1, $2, $3)",
		user.ID,
		curTime,
		curTime,
	)

	return err
}

func InsertRefreshToken(user *models.User, refreshTokenHash string, expirationDate time.Time, db *sqlx.DB) error {
	_, err := db.Exec(
		"INSERT INTO refresh_tokens(user_id, token_hash, expiration_date) VALUES ($1, $2, $3)",
		user.ID,
		refreshTokenHash,
		expirationDate,
	)
	return err
}

func InsertThread(thread *models.Thread, userID uuid.UUID, db *sqlx.DB) error {
	curTime := time.Now()

	_, err := db.Exec(
		"INSERT INTO threads(user_id, title, original_post, creation_date, updated_date, like_count, view_count) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		userID,
		thread.Title,
		thread.OriginalPost,
		curTime,
		curTime,
		0,
		0,
	)

	return err
}
