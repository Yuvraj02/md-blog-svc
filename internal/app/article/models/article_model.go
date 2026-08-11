package models

import "time"

// Status is the single source of truth for article lifecycle.
type Status string

const (
	StatusDraft     Status = "DRAFT"
	StatusPublished Status = "PUBLISHED"
)

// Article is the canonical public (or never-published) piece.
type Article struct {
	ID                 string
	BlogID             string
	Title              string
	Excerpt            string
	ImageURL           string
	VideoURL           string
	TagsJSON           string
	ContentJSON        string
	Status             Status
	Upvotes             int64
	Views              int64
	ViewsThisWeek      int64
	Reads              int64
	ReadingTimeMinutes int
	PublishedAt        *time.Time
	LastSaved          *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Comments           []Comment
	HasDraft           bool // studio hint: an edit-draft exists
}

// Draft is the in-progress revision of a PUBLISHED article (at most one per article).
type Draft struct {
	ID                 string
	ArticleID          string
	Title              string
	Excerpt            string
	ImageURL           string
	VideoURL           string
	TagsJSON           string
	ContentJSON        string
	ReadingTimeMinutes int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Comment is a reader comment on an article.
type Comment struct {
	ID        string
	ArticleID string
	Name      string
	Email     string
	Website   string
	Body      string
	CreatedAt time.Time
}
