package models

type Pagination struct {
	currentPage int `json:"current_page"`
	lastPage    int `json:"last_page"`
	pageSize    int `json:"page_size"`
}
