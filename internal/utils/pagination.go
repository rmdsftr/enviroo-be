package utils

import "strconv"

type PageMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func ParsePage(pageStr string) int {
	if pageStr == "" {
		return 0
	}
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	return page
}

func Offset(page, limit int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * limit
}

func NewPageMeta(page, limit, total int) *PageMeta {
	if page < 1 {
		page = 1
	}
	totalPages := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}
	return &PageMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}
