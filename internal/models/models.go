package models

type Pagination struct {
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	PageSize    int `json:"page_size"`
}

type PaginatedResponse[T any] struct {
	Pagination Pagination `json:"pagination"`
	Response   []T        `json:"response"`
}

const (
	DEFAULT_PAGE_SIZE = 5
)
