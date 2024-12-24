package models

import (
	"github.com/google/uuid"
	"time"
)

type User struct {
	ID       uuid.UUID `db:"id" json:"id"`
	Username string    `db:"username" json:"username" binding:"required"`
	Password string    `db:"password_hash" json:"password" binding:"required"`
}

type UserProfile struct {
	ID               uuid.UUID   `db:"id" json:"id"`
	ProfilePhotoPath string      `db:"profile_photo_path" json:"profile_photo_path"`
	Biodata          string      `db:"biodata" json:"biodata"`
	Email            string      `db:"email" json:"email"`
	PostCount        int         `db:"post_count" json:"post_count"`
	CommentCount     int         `db:"comment_count" json:"comment_count"`
	PreferredLang    AppLanguage `db:"preferred_lang" json:"preferred_lang"`
	PreferredTheme   AppTheme    `db:"preferred_theme" json:"preferred_theme"`
	CreationDate     time.Time   `db:"creation_date" json:"creation_date"`
	LastLogin        time.Time   `db:"last_login" json:"last_login"`
}
