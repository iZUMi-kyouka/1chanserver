package utils_db

import (
	"1chanserver/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"time"
)

func FetchOne[T any](db *sqlx.DB, query string, args ...interface{}) (T, error) {
	var result T
	err := db.Get(&result, query, args...)
	return result, err
}

func FetchAll[T any](db *sqlx.DB, query string, args ...interface{}) ([]T, error) {
	var result []T
	err := db.Select(&result, query, args...)
	return result, err
}

func GetUserByUsername(username *string, db *sqlx.DB) (models.User, error) {
	var user models.User
	err := db.Get(&user, "SELECT * FROM users WHERE username = $1", *username)
	return user, err
}

func _GetUserByUserID(userID *uuid.UUID, db *sqlx.DB) (models.User, error) {
	var user models.User
	err := db.Get(&user, "SELECT * FROM users WHERE id = $1", (*userID).String())
	return user, err
}

func EditUserProfile(userProfile *models.UserProfile, db *sqlx.DB) error {
	query := "UPDATE user_profiles SET " +
		"profile_picture_path = :profile_photo_path," +
		"biodata = :biodata," +
		"email = :email," +
		"post_count = :post_count," +
		"comment_count = :comment_count," +
		"preferred_lang = :preferred_lang" +
		"preferred_theme = :preferred_theme" +
		"creation_date = :creation_date" +
		"last_login = :last_login" +
		"WHERE id = :id"
	_, err := db.NamedExec(query, userProfile)
	return err
}

func DeleteUser(userID *uuid.UUID, db *sqlx.DB) error {
	query := "DELETE FROM users WHERE id = $1"
	_, err := db.Exec(query, userID.String())
	return err
}

func InsertRefreshToken(user *models.User, refreshTokenHash string, expirationDate time.Time, devcieID string, db *sqlx.DB) error {
	_, err := db.Exec(
		"INSERT INTO refresh_tokens(user_id, token_hash, expiration_date, device_id) VALUES ($1, $2, $3, $4)",
		user.ID,
		refreshTokenHash,
		expirationDate,
		devcieID)
	return err
}

func DeleteThread(threadID int, db *sqlx.DB) error {
	query := "DELETE FROM threads WHERE id = $1"
	_, err := db.Exec(query, threadID)
	return err
}

func InsertComment(comment *models.Comment, db *sqlx.DB) error {
	query := "INSERT INTO comments(thread_id, user_id, comment) VALUES ($1, $2, $3)"
	_, err := db.Exec(query, comment.ThreadID, comment.UserID, comment.Comment)
	return err
}

func EditComment(comment *models.Comment, db *sqlx.DB) error {
	query := "UPDATE comments SET " +
		"comment = $1, " +
		"updated_date = $2 WHERE id = $3"
	_, err := db.Exec(query, comment.Comment, time.Now(), comment.ID)
	return err
}

func DeleteComment(comment *models.Comment, db *sqlx.DB) error {
	query := "DELETE FROM comments WHERE id = $1"
	_, err := db.Exec(query, comment.ID)
	return err
}
