package models

type Pagination struct {
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	PageSize    int `json:"page_size"`
}

const (
	DEFAULT_PAGE_SIZE = 100
)
