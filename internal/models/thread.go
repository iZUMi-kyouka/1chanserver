package models

import (
	"github.com/google/uuid"
	"time"
)

type Thread struct {
	ID              int        `json:"id" db:"id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	ChannelID       *int64     `json:"channel_id" db:"channel_id"`
	Title           string     `json:"title" db:"title"`
	OriginalPost    string     `json:"original_post" db:"original_post"`
	CreationDate    time.Time  `json:"creation_date" db:"creation_date"`
	UpdatedDate     *time.Time `json:"updated_date" db:"updated_date"`
	LastCommentDate *time.Time `json:"last_comment_date" db:"last_comment_date""`
	LikeCount       int        `json:"like_count" db:"like_count"`
	ViewCount       int        `json:"view_count" db:"view_count"`
}

type ThreadView struct {
	ID              int        `json:"id" db:"id"`
	Username        string     `json:"username" db:"username"`
	UserProfilePath *string    `json:"user_profile_path" db:"profile_picture_path"`
	Channel         *string    `json:"channel" db:"channel"`
	Title           string     `json:"title" db:"title"`
	OriginalPost    string     `json:"original_post" db:"original_post"`
	CreationDate    time.Time  `json:"creation_date" db:"creation_date"`
	UpdatedDate     *time.Time `json:"updated_date" db:"updated_date"`
	LastCommentDate *time.Time `json:"last_comment_date" db:"last_comment_date"`
	LikeCount       int        `json:"like_count" db:"like_count"`
	DislikeCount    int        `json:"dislike_count" db:"dislike_count"`
	CommentCount    int        `json:"comment_count" db:"comment_count"`
	ViewCount       int        `json:"view_count" db:"view_count"`
	TagIDs          *string    `json:"tags" db:"tags"`               // For default tag IDs
	CustomTagNames  *string    `json:"custom_tags" db:"custom_tags"` // For custom tag names
}

type ThreadRequest struct {
	Title        string   `json:"title"`
	OriginalPost string   `json:"original_post"`
	Tags         []Tag    `json:"tags"`
	CustomTags   []string `json:"custom_tags"`
}

func (t *Thread) IsOwnedBy(userID *uuid.UUID) bool {
	return &t.UserID == userID
}

type ThreadPagination struct {
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	ThreadCount int `json:"thread_count"`
}

type ThreadListResponse struct {
	Threads     []ThreadView `json:"threads"`
	Paginations Pagination   `json:"paginations"`
}

type ThreadViewResponse struct {
	Thread   ThreadView                     `json:"thread"`
	Comments PaginatedResponse[CommentView] `json:"comments"`
}
