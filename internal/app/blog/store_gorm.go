package blog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/marketing-digest/pkg/errorsx"

	"github.com/marketing-digest/blog-service/internal/app/blog/models"
)

type row struct {
	ID                      string    `gorm:"primaryKey"`
	Name                    string    `gorm:"not null"`
	Slug                    string    `gorm:"uniqueIndex;not null"`
	Description             string    `gorm:"not null;default:''"`
	CoverImage              string    `gorm:"column:cover_image;not null;default:''"`
	OwnerID                 string    `gorm:"column:owner_id;not null"`
	Upvotes                  int64     `gorm:"not null;default:0"`
	TotalViews              int64     `gorm:"column:total_views;not null;default:0"`
	ArticleCount            int       `gorm:"column:article_count;not null;default:0"`
	TotalReadingTimeMinutes int       `gorm:"column:total_reading_time_minutes;not null;default:0"`
	CreatedAt               time.Time `gorm:"not null"`
	UpdatedAt               time.Time `gorm:"not null"`
}

func (row) TableName() string { return "blogs" }

type GORMStore struct{ db *gorm.DB }

func NewGORMStore(db *gorm.DB) *GORMStore { return &GORMStore{db: db} }

func (s *GORMStore) List(ctx context.Context) ([]*models.Blog, error) {
	var rows []row
	if err := s.db.WithContext(ctx).Order("updated_at desc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list blogs: %w", err)
	}
	out := make([]*models.Blog, 0, len(rows))
	for i := range rows {
		out = append(out, fromRow(&rows[i]))
	}
	return out, nil
}

func (s *GORMStore) Get(ctx context.Context, id string) (*models.Blog, error) {
	var m row
	err := s.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get blog: %w", err)
	}
	return fromRow(&m), nil
}

func (s *GORMStore) Create(ctx context.Context, b *models.Blog) error {
	m := toRow(b)
	if err := s.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("create blog: %w", err)
	}
	*b = *fromRow(m)
	return nil
}

func (s *GORMStore) Update(ctx context.Context, b *models.Blog) error {
	m := toRow(b)
	if err := s.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("update blog: %w", err)
	}
	*b = *fromRow(m)
	return nil
}

func (s *GORMStore) BumpArticleStats(ctx context.Context, blogID string, articleDelta, readingDelta int) error {
	return s.db.WithContext(ctx).Model(&row{}).Where("id = ?", blogID).Updates(map[string]any{
		"article_count":               gorm.Expr("article_count + ?", articleDelta),
		"total_reading_time_minutes":  gorm.Expr("total_reading_time_minutes + ?", readingDelta),
		"updated_at":                  time.Now().UTC(),
	}).Error
}

func toRow(b *models.Blog) *row {
	return &row{
		ID: b.ID, Name: b.Name, Slug: b.Slug, Description: b.Description,
		CoverImage: b.CoverImage, OwnerID: b.OwnerID, Upvotes: b.Upvotes,
		TotalViews: b.TotalViews, ArticleCount: b.ArticleCount,
		TotalReadingTimeMinutes: b.TotalReadingTimeMinutes,
		CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
	}
}

func fromRow(m *row) *models.Blog {
	return &models.Blog{
		ID: m.ID, Name: m.Name, Slug: m.Slug, Description: m.Description,
		CoverImage: m.CoverImage, OwnerID: m.OwnerID, Upvotes: m.Upvotes,
		TotalViews: m.TotalViews, ArticleCount: m.ArticleCount,
		TotalReadingTimeMinutes: m.TotalReadingTimeMinutes,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}
