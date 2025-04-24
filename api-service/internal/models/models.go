package models

// Article represents a news article
type Article struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Link        string `json:"link"`
	Published   string `json:"published"`
}
