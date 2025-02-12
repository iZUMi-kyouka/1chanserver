package utils_db

import (
	"1chanserver/internal/models"
	"1chanserver/internal/models/api_error"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"log"
	"net/http"
	"strings"
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
		"preferred_lang = :preferred_lang" +
		"preferred_theme = :preferred_theme" +
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

func HandleTxRollback(tx *sqlx.Tx, err *error, c *gin.Context) {
	if p := recover(); p != nil {
		err := tx.Rollback()
		if err != nil {
			log.Fatalf("failed to rollback db: %s", err.Error())
		}
		panic(p)
	} else if *err != nil {
		err := tx.Rollback()
		if err != nil {
			log.Fatalf("failed to rollback db: %s", err.Error())
		}
	} else {
		err := tx.Commit()
		if err != nil {
			c.Error(err)
		}
	}
}

func ToInQueryForm[T any](s []T) string {
	if len(s) == 0 {
		return "()"
	}

	var builder strings.Builder
	builder.WriteString("(")

	for i, v := range s {
		switch v := any(v).(type) {
		case int:
			builder.WriteString(fmt.Sprintf("%d", v))
		case string:
			builder.WriteString(fmt.Sprintf("'%s'", v))
		default:
			panic("unsupported tag identifier type")
		}

		if i < len(s)-1 {
			builder.WriteString(", ")
		}
	}

	builder.WriteString(")")
	return builder.String()
}

func GetTotalRecordNo(db *sqlx.DB, query string, args ...interface{}) (int, error) {
	recordNo, err := FetchOne[int](db, query, args...)
	if err != nil {
		return -1, err
	}

	return recordNo, nil
}

func CheckDuplicateError(err error) bool {
	if err, ok := err.(*pq.Error); ok {
		if err.Code == "23505" {
			return true
		}
	}

	return false
}

func SortCriteriaToDBColumn(s string) (string, error) {
	switch s {
	case "relevance":
		return "rank", nil
	case "views":
		return "view_count", nil
	case "likes":
		return "like_count", nil
	case "dislikes":
		return "dislike_count", nil
	case "date":
		return "creation_date", nil
	default:
		return "", api_error.NewFromStr("invalid sort criteria", http.StatusBadRequest)
	}
}

func SortCriteriaToDBColumnWithAlias(sortCriteria string, tableAlias string) (string, error) {
	switch tableAlias {
	case "":
		return SortCriteriaToDBColumn(sortCriteria)
	default:
		columnName, err := SortCriteriaToDBColumn(sortCriteria)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("%s.%s", tableAlias, columnName), nil
	}
}

func GetCustomTagID(db *sqlx.DB, customTags []string) ([]int, error) {
	if len(customTags) == 0 {
		return []int{}, nil
	}

	customTagIDs := make([]int, len(customTags))
	for _, tag := range customTags {
		customTagID, err := FetchOne[int](db, "SELECT id FROM custom_tags WHERE tag = $1", tag)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}

			return nil, err
		}

		customTagIDs = append(customTagIDs, customTagID)
	}

	return customTagIDs, nil
}
