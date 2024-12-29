package models

import (
	"github.com/google/uuid"
	"time"
)

type Thread struct {
	ID              int64      `json:"id" db:"id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	ChannelID       *int64     `json:"channel_id" db:"channel_id"`
	Title           string     `json:"title" db:"title"`
	OriginalPost    string     `json:"original_post" db:"original_post"`
	CreationDate    time.Time  `json:"creation_date" db:"creation_date"`
	UpdatedDate     *time.Time `json:"update_date" db:"update_date"`
	LastCommentDate *time.Time `json:"last_comment_date" db:"last_comment_date""`
	LikeCount       int        `json:"like_count" db:"like_count"`
	ViewCount       int        `json:"view_count" db:"view_count"`
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
	Threads     []Thread   `json:"threads"`
	Paginations Pagination `json:"paginations"`
}

type ThreadViewResponse struct {
	Thread   Thread          `json:"thread"`
	Comments CommentResponse `json:"comments_response"`
}
