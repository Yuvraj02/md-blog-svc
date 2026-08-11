package blog

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/marketing-digest/pkg/errorsx"

	"github.com/marketing-digest/blog-service/internal/app/blog/models"
)

// Store persists blogs.
type Store interface {
	List(ctx context.Context) ([]*models.Blog, error)
	Get(ctx context.Context, id string) (*models.Blog, error)
	Create(ctx context.Context, b *models.Blog) error
	Update(ctx context.Context, b *models.Blog) error
	BumpArticleStats(ctx context.Context, blogID string, articleDelta, readingDelta int) error
}

// Service handles blog create / list / update / upvote.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) List(ctx context.Context) ([]*models.Blog, error) {
	return s.store.List(ctx)
}

func (s *Service) Get(ctx context.Context, id string) (*models.Blog, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errorsx.ErrInvalidArgument
	}
	return s.store.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, name, description, ownerID string) (*models.Blog, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Untitled blog"
	}
	if ownerID == "" {
		ownerID = "user-1"
	}
	now := time.Now().UTC()
	b := &models.Blog{
		ID:          uuid.NewString(),
		Name:        name,
		Slug:        slugify(name),
		Description: strings.TrimSpace(description),
		OwnerID:     ownerID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.Create(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) Update(ctx context.Context, id string, name, description, coverImage, slug *string) (*models.Blog, error) {
	b, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		b.Name = strings.TrimSpace(*name)
	}
	if description != nil {
		b.Description = strings.TrimSpace(*description)
	}
	if coverImage != nil {
		b.CoverImage = strings.TrimSpace(*coverImage)
	}
	if slug != nil && strings.TrimSpace(*slug) != "" {
		b.Slug = slugify(*slug)
	}
	b.UpdatedAt = time.Now().UTC()
	if err := s.store.Update(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) Upvote(ctx context.Context, id string) (*models.Blog, error) {
	b, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	b.Upvotes++
	b.UpdatedAt = time.Now().UTC()
	if err := s.store.Update(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "untitled"
	}
	return out
}
