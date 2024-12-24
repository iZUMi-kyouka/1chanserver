package models

import (
	"github.com/google/uuid"
	"time"
)

type Thread struct {
	ID           int64     `json:"id" db:"id"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	Title        string    `json:"title" db:"title"`
	OriginalPost string    `json:"original_post" db:"original_post"`
	CreationDate time.Time `json:"creation_date" db:"creation_date"`
	UpdatedDate  time.Time `json:"update_date" db:"update_date"`
	LikeCount    int       `json:"like_count" db:"like_count"`
	ViewCount    int       `json:"view_count" db:"view_count"`
}

type ThreadPagination struct {
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	ThreadCount int `json:"thread_count"`
}

type ThreadResponse struct {
	Threads     []Thread   `json:"threads"`
	Paginations Pagination `json:"paginations"`
}
