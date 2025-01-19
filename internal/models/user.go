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

type UserForAuth struct {
	Username string `db:"username" json:"username" binding:"required"`
	Password string `db:"password" json:"password" binding:"required"`
	DeviceID string `db:"device_id" json:"device_id" binding:"required"`
}

func (u *User) IsOwnedBy(userID *uuid.UUID) bool {
	return &u.ID == userID
}

type UserProfile struct {
	ID                 uuid.UUID   `db:"id" json:"id"`
	ProfilePicturePath *string     `db:"profile_picture_path" json:"profile_picture_path"`
	Biodata            string      `db:"biodata" json:"biodata"`
	Email              *string     `db:"email" json:"email"`
	PostCount          int         `db:"post_count" json:"post_count"`
	CommentCount       int         `db:"comment_count" json:"comment_count"`
	PreferredLang      AppLanguage `db:"preferred_lang" json:"preferred_lang"`
	PreferredTheme     AppTheme    `db:"preferred_theme" json:"preferred_theme"`
	CreationDate       time.Time   `db:"creation_date" json:"creation_date"`
	LastLogin          time.Time   `db:"last_login" json:"last_login"`
}

type UserAccountResponse struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Username    string    `db:"username" json:"username"`
	AccessToken string    `db:"access_token" json:"access_token"`
}

type UserForResponse struct {
	Account UserAccountResponse `json:"account"`
	Profile UserProfile         `json:"profile"`
}

func (u *UserProfile) IsOwnedBy(userID *uuid.UUID) bool {
	return &u.ID == userID
}
