package models

import (
	"github.com/google/uuid"
	"time"
)

type Comment struct {
	ID           int64      `json:"id" db:"id"`
	ThreadID     int        `json:"thread_id" db:"thread_id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	Comment      string     `json:"comment" db:"comment"`
	CreationDate time.Time  `json:"creation_date" db:"creation_date"`
	UpdatedDate  *time.Time `json:"updated_date" db:"updated_date"`
	LikeCount    int        `json:"like_count" db:"like_count"`
	DislikeCount int        `json:"dislike_count" db:"dislike_count"`
}

type CommentView struct {
	ID              int64      `json:"id" db:"id"`
	Username        string     `json:"username" db:"username"`
	UserProfilePath *string    `json:"user_profile_path" db:"profile_picture_path"`
	Comment         string     `json:"comment" db:"comment"`
	CreationDate    time.Time  `json:"creation_date" db:"creation_date"`
	UpdatedDate     *time.Time `json:"updated_date" db:"updated_date"`
	LikeCount       int        `json:"like_count" db:"like_count"`
	DislikeCount    int        `json:"dislike_count" db:"dislike_count"`
}

func (c *Comment) IsOwnedBy(userID *uuid.UUID) bool {
	return &c.UserID == userID
}
