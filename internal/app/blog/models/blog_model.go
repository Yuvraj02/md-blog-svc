package models

import "time"

// Blog is one publication / workspace owned by a user.
type Blog struct {
	ID                       string
	Name                     string
	Slug                     string
	Description              string
	CoverImage               string
	OwnerID                  string
	Upvotes                   int64
	TotalViews               int64
	ArticleCount             int
	TotalReadingTimeMinutes  int
	CreatedAt                time.Time
	UpdatedAt                time.Time
}
