package article

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/marketing-digest/pkg/errorsx"

	"github.com/marketing-digest/blog-service/internal/app/article/models"
	blogapp "github.com/marketing-digest/blog-service/internal/app/blog"
)

// Service covers studio editing, public reading, comments, and edit-drafts.
type Service struct {
	store     Store
	blogStore blogapp.Store
}

func NewService(store Store, blogStore blogapp.Store) *Service {
	return &Service{store: store, blogStore: blogStore}
}

func (s *Service) ListByBlog(ctx context.Context, blogID string) ([]*models.Article, error) {
	if blogID == "" {
		return nil, errorsx.ErrInvalidArgument
	}
	return s.store.ListByBlog(ctx, blogID)
}

func (s *Service) Get(ctx context.Context, blogID, articleID string) (*models.Article, error) {
	if blogID == "" || articleID == "" {
		return nil, errorsx.ErrInvalidArgument
	}
	return s.store.Get(ctx, blogID, articleID)
}

func (s *Service) Create(ctx context.Context, blogID string) (*models.Article, error) {
	if blogID == "" {
		return nil, errorsx.ErrInvalidArgument
	}
	if _, err := s.blogStore.Get(ctx, blogID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	a := &models.Article{
		ID:                 uuid.NewString(),
		BlogID:             blogID,
		TagsJSON:           "[]",
		ContentJSON:        "[]",
		Status:             models.StatusDraft,
		ReadingTimeMinutes: 1,
		LastSaved:          &now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.store.Create(ctx, a); err != nil {
		return nil, err
	}
	_ = s.blogStore.BumpArticleStats(ctx, blogID, 1, 0)
	return a, nil
}

// UpdatePatch is editor save for a never-published DRAFT article only.
type UpdatePatch struct {
	Title              *string
	Excerpt            *string
	Tags               []string
	HasTags            bool
	ImageURL           *string
	VideoURL           *string
	ContentJSON        *string
	ReadingTimeMinutes *int
}

func (s *Service) Update(ctx context.Context, blogID, articleID string, patch UpdatePatch) (*models.Article, error) {
	a, err := s.store.Get(ctx, blogID, articleID)
	if err != nil {
		return nil, err
	}
	// Published articles must be edited via Draft, not patched in place.
	if a.Status == models.StatusPublished {
		return nil, fmtInvalid("published articles must be edited through a draft")
	}
	now := time.Now().UTC()
	applyContentPatch(a, patch, now)
	if err := s.store.Update(ctx, a); err != nil {
		return nil, err
	}
	_ = s.store.LoadComments(ctx, a)
	return a, nil
}

// PublishArticle flips a never-published DRAFT article to PUBLISHED.
func (s *Service) PublishArticle(ctx context.Context, blogID, articleID string) (*models.Article, error) {
	a, err := s.store.Get(ctx, blogID, articleID)
	if err != nil {
		return nil, err
	}
	if a.Status == models.StatusPublished {
		// Idempotent: already live.
		return a, nil
	}
	now := time.Now().UTC()
	a.Status = models.StatusPublished
	if a.PublishedAt == nil {
		a.PublishedAt = &now
	}
	a.UpdatedAt = now
	a.LastSaved = &now
	if err := s.store.Update(ctx, a); err != nil {
		return nil, err
	}
	_ = s.store.LoadComments(ctx, a)
	return a, nil
}

// EnsureDraft creates (or reuses) the one edit-draft for a PUBLISHED article.
func (s *Service) EnsureDraft(ctx context.Context, blogID, articleID string) (*models.Draft, error) {
	a, err := s.store.Get(ctx, blogID, articleID)
	if err != nil {
		return nil, err
	}
	if a.Status != models.StatusPublished {
		return nil, fmtInvalid("only published articles can have an edit draft")
	}
	existing, err := s.store.GetDraftByArticleID(ctx, articleID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, errorsx.ErrNotFound) {
		return nil, err
	}
	now := time.Now().UTC()
	d := &models.Draft{
		ID:                 uuid.NewString(),
		ArticleID:          articleID,
		Title:              a.Title,
		Excerpt:            a.Excerpt,
		ImageURL:           a.ImageURL,
		VideoURL:           a.VideoURL,
		TagsJSON:           a.TagsJSON,
		ContentJSON:        a.ContentJSON,
		ReadingTimeMinutes: a.ReadingTimeMinutes,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.store.CreateDraft(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) GetDraft(ctx context.Context, blogID, articleID string) (*models.Draft, error) {
	if _, err := s.store.Get(ctx, blogID, articleID); err != nil {
		return nil, err
	}
	return s.store.GetDraftByArticleID(ctx, articleID)
}

// DraftPatch is editor save onto an edit-draft.
type DraftPatch struct {
	Title              *string
	Excerpt            *string
	Tags               []string
	HasTags            bool
	ImageURL           *string
	VideoURL           *string
	ContentJSON        *string
	ReadingTimeMinutes *int
}

func (s *Service) UpdateDraft(ctx context.Context, blogID, articleID string, patch DraftPatch) (*models.Draft, error) {
	if _, err := s.store.Get(ctx, blogID, articleID); err != nil {
		return nil, err
	}
	d, err := s.store.GetDraftByArticleID(ctx, articleID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if patch.Title != nil {
		d.Title = *patch.Title
	}
	if patch.Excerpt != nil {
		d.Excerpt = *patch.Excerpt
	}
	if patch.HasTags {
		b, _ := json.Marshal(patch.Tags)
		d.TagsJSON = string(b)
	}
	if patch.ImageURL != nil {
		d.ImageURL = *patch.ImageURL
	}
	if patch.VideoURL != nil {
		d.VideoURL = *patch.VideoURL
	}
	if patch.ContentJSON != nil {
		d.ContentJSON = *patch.ContentJSON
	}
	if patch.ReadingTimeMinutes != nil {
		d.ReadingTimeMinutes = *patch.ReadingTimeMinutes
	}
	d.UpdatedAt = now
	if err := s.store.UpdateDraft(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// PublishDraft copies draft → article and deletes the draft in one transaction.
func (s *Service) PublishDraft(ctx context.Context, blogID, articleID string) (*models.Article, error) {
	return s.store.PublishDraftTx(ctx, blogID, articleID)
}

func (s *Service) Delete(ctx context.Context, blogID, articleID string) error {
	a, err := s.store.Get(ctx, blogID, articleID)
	if err != nil {
		return err
	}
	if err := s.store.Delete(ctx, blogID, articleID); err != nil {
		return err
	}
	_ = s.blogStore.BumpArticleStats(ctx, blogID, -1, -a.ReadingTimeMinutes)
	return nil
}

func (s *Service) Upvote(ctx context.Context, blogID, articleID string) (*models.Article, error) {
	a, err := s.store.Get(ctx, blogID, articleID)
	if err != nil {
		return nil, err
	}
	a.Upvotes++
	a.UpdatedAt = time.Now().UTC()
	if err := s.store.Update(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) ListPublished(ctx context.Context) ([]*models.Article, error) {
	return s.store.ListPublished(ctx)
}

func (s *Service) ListTrending(ctx context.Context) ([]*models.Article, error) {
	return s.store.ListTrending(ctx)
}

func (s *Service) GetPublished(ctx context.Context, articleID string) (*models.Article, error) {
	if articleID == "" {
		return nil, errorsx.ErrInvalidArgument
	}
	return s.store.GetPublished(ctx, articleID)
}

func (s *Service) ListRelated(ctx context.Context, articleID string, limit int) ([]*models.Article, error) {
	if limit <= 0 {
		limit = 3
	}
	current, err := s.store.GetPublished(ctx, articleID)
	if err != nil {
		return nil, err
	}
	all, err := s.store.ListPublished(ctx)
	if err != nil {
		return nil, err
	}
	curTags := decodeTags(current.TagsJSON)
	type scored struct {
		a      *models.Article
		shared int
	}
	var list []scored
	for _, a := range all {
		if a.ID == articleID {
			continue
		}
		shared := 0
		for _, t := range decodeTags(a.TagsJSON) {
			for _, c := range curTags {
				if t == c {
					shared++
				}
			}
		}
		list = append(list, scored{a: a, shared: shared})
	}
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].shared > list[i].shared {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	if len(list) > limit {
		list = list[:limit]
	}
	out := make([]*models.Article, 0, len(list))
	for _, item := range list {
		out = append(out, item.a)
	}
	return out, nil
}

func (s *Service) RecordView(ctx context.Context, articleID string) error {
	a, err := s.store.GetPublished(ctx, articleID)
	if err != nil {
		return err
	}
	a.Views++
	a.ViewsThisWeek++
	a.UpdatedAt = time.Now().UTC()
	return s.store.Update(ctx, a)
}

func (s *Service) RecordRead(ctx context.Context, articleID string) error {
	a, err := s.store.GetPublished(ctx, articleID)
	if err != nil {
		return err
	}
	a.Reads++
	a.UpdatedAt = time.Now().UTC()
	return s.store.Update(ctx, a)
}

func (s *Service) ListComments(ctx context.Context, articleID string) ([]*models.Comment, error) {
	if _, err := s.store.GetByID(ctx, articleID); err != nil {
		return nil, err
	}
	return s.store.ListComments(ctx, articleID)
}

func (s *Service) CreateComment(ctx context.Context, articleID, name, email, website, body string) (*models.Comment, error) {
	name, email, body = strings.TrimSpace(name), strings.TrimSpace(email), strings.TrimSpace(body)
	if name == "" || email == "" || body == "" {
		return nil, errorsx.ErrInvalidArgument
	}
	a, err := s.store.GetPublished(ctx, articleID)
	if err != nil {
		return nil, err
	}
	c := &models.Comment{
		ID: uuid.NewString(), ArticleID: a.ID, Name: name, Email: email,
		Website: strings.TrimSpace(website), Body: body, CreatedAt: time.Now().UTC(),
	}
	if err := s.store.CreateComment(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func applyContentPatch(a *models.Article, patch UpdatePatch, now time.Time) {
	if patch.Title != nil {
		a.Title = *patch.Title
	}
	if patch.Excerpt != nil {
		a.Excerpt = *patch.Excerpt
	}
	if patch.HasTags {
		b, _ := json.Marshal(patch.Tags)
		a.TagsJSON = string(b)
	}
	if patch.ImageURL != nil {
		a.ImageURL = *patch.ImageURL
	}
	if patch.VideoURL != nil {
		a.VideoURL = *patch.VideoURL
	}
	if patch.ContentJSON != nil {
		a.ContentJSON = *patch.ContentJSON
	}
	if patch.ReadingTimeMinutes != nil {
		a.ReadingTimeMinutes = *patch.ReadingTimeMinutes
	}
	a.LastSaved = &now
	a.UpdatedAt = now
}

func decodeTags(raw string) []string {
	if raw == "" {
		return nil
	}
	var tags []string
	_ = json.Unmarshal([]byte(raw), &tags)
	return tags
}

func fmtInvalid(msg string) error {
	return fmt.Errorf("%w: %s", errorsx.ErrInvalidArgument, msg)
}
