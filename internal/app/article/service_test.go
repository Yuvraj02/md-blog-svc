package article_test

import (
	"context"
	"testing"
	"time"

	"github.com/marketing-digest/pkg/errorsx"

	"github.com/marketing-digest/blog-service/internal/app/article"
	"github.com/marketing-digest/blog-service/internal/app/article/models"
	blogmodels "github.com/marketing-digest/blog-service/internal/app/blog/models"
)

type memBlogStore struct {
	blogs map[string]*blogmodels.Blog
}

func (m *memBlogStore) List(context.Context) ([]*blogmodels.Blog, error) { return nil, nil }
func (m *memBlogStore) Get(_ context.Context, id string) (*blogmodels.Blog, error) {
	b, ok := m.blogs[id]
	if !ok {
		return nil, errorsx.ErrNotFound
	}
	return b, nil
}
func (m *memBlogStore) Create(context.Context, *blogmodels.Blog) error { return nil }
func (m *memBlogStore) Update(context.Context, *blogmodels.Blog) error { return nil }
func (m *memBlogStore) BumpArticleStats(context.Context, string, int, int) error {
	return nil
}

type memArticleStore struct {
	articles map[string]*models.Article
	drafts   map[string]*models.Draft
}

func newMem() *memArticleStore {
	return &memArticleStore{
		articles: map[string]*models.Article{},
		drafts:   map[string]*models.Draft{},
	}
}

func (m *memArticleStore) ListByBlog(_ context.Context, blogID string) ([]*models.Article, error) {
	var out []*models.Article
	for _, a := range m.articles {
		if a.BlogID == blogID {
			cp := *a
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (m *memArticleStore) Get(_ context.Context, blogID, articleID string) (*models.Article, error) {
	a, ok := m.articles[articleID]
	if !ok || a.BlogID != blogID {
		return nil, errorsx.ErrNotFound
	}
	cp := *a
	cp.HasDraft = m.drafts[articleID] != nil
	return &cp, nil
}
func (m *memArticleStore) GetByID(_ context.Context, id string) (*models.Article, error) {
	a, ok := m.articles[id]
	if !ok {
		return nil, errorsx.ErrNotFound
	}
	cp := *a
	return &cp, nil
}
func (m *memArticleStore) GetPublished(_ context.Context, id string) (*models.Article, error) {
	a, ok := m.articles[id]
	if !ok || a.Status != models.StatusPublished {
		return nil, errorsx.ErrNotFound
	}
	cp := *a
	return &cp, nil
}
func (m *memArticleStore) ListPublished(context.Context) ([]*models.Article, error) {
	var out []*models.Article
	for _, a := range m.articles {
		if a.Status == models.StatusPublished {
			cp := *a
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (m *memArticleStore) ListTrending(ctx context.Context) ([]*models.Article, error) {
	return m.ListPublished(ctx)
}
func (m *memArticleStore) Create(_ context.Context, a *models.Article) error {
	cp := *a
	m.articles[a.ID] = &cp
	return nil
}
func (m *memArticleStore) Update(_ context.Context, a *models.Article) error {
	if _, ok := m.articles[a.ID]; !ok {
		return errorsx.ErrNotFound
	}
	cp := *a
	m.articles[a.ID] = &cp
	return nil
}
func (m *memArticleStore) Delete(_ context.Context, blogID, articleID string) error {
	a, ok := m.articles[articleID]
	if !ok || a.BlogID != blogID {
		return errorsx.ErrNotFound
	}
	delete(m.articles, articleID)
	delete(m.drafts, articleID)
	return nil
}
func (m *memArticleStore) ListComments(context.Context, string) ([]*models.Comment, error) {
	return nil, nil
}
func (m *memArticleStore) CreateComment(context.Context, *models.Comment) error { return nil }
func (m *memArticleStore) LoadComments(context.Context, ...*models.Article) error {
	return nil
}
func (m *memArticleStore) GetDraftByArticleID(_ context.Context, articleID string) (*models.Draft, error) {
	d, ok := m.drafts[articleID]
	if !ok {
		return nil, errorsx.ErrNotFound
	}
	cp := *d
	return &cp, nil
}
func (m *memArticleStore) CreateDraft(_ context.Context, d *models.Draft) error {
	if existing, ok := m.drafts[d.ArticleID]; ok {
		cp := *existing
		*d = cp
		return nil
	}
	cp := *d
	m.drafts[d.ArticleID] = &cp
	return nil
}
func (m *memArticleStore) UpdateDraft(_ context.Context, d *models.Draft) error {
	if _, ok := m.drafts[d.ArticleID]; !ok {
		return errorsx.ErrNotFound
	}
	cp := *d
	m.drafts[d.ArticleID] = &cp
	return nil
}
func (m *memArticleStore) DeleteDraft(_ context.Context, articleID string) error {
	delete(m.drafts, articleID)
	return nil
}
func (m *memArticleStore) HasDraftMap(_ context.Context, ids []string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, id := range ids {
		out[id] = m.drafts[id] != nil
	}
	return out, nil
}
func (m *memArticleStore) PublishDraftTx(_ context.Context, blogID, articleID string) (*models.Article, error) {
	a, ok := m.articles[articleID]
	if !ok || a.BlogID != blogID {
		return nil, errorsx.ErrNotFound
	}
	d, ok := m.drafts[articleID]
	if !ok {
		return nil, errorsx.ErrNotFound
	}
	now := time.Now().UTC()
	a.Title = d.Title
	a.Excerpt = d.Excerpt
	a.ImageURL = d.ImageURL
	a.VideoURL = d.VideoURL
	a.TagsJSON = d.TagsJSON
	a.ContentJSON = d.ContentJSON
	a.ReadingTimeMinutes = d.ReadingTimeMinutes
	a.Status = models.StatusPublished
	a.UpdatedAt = now
	a.LastSaved = &now
	delete(m.drafts, articleID)
	cp := *a
	return &cp, nil
}

func setupSvc(t *testing.T) (*article.Service, *memArticleStore) {
	t.Helper()
	store := newMem()
	blogs := &memBlogStore{blogs: map[string]*blogmodels.Blog{
		"blog-1": {ID: "blog-1", Name: "Test"},
	}}
	return article.NewService(store, blogs), store
}

func TestCreateArticleIsDraft(t *testing.T) {
	svc, _ := setupSvc(t)
	a, err := svc.Create(context.Background(), "blog-1")
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != models.StatusDraft {
		t.Fatalf("status=%s", a.Status)
	}
}

func TestEnsureDraftIdempotent(t *testing.T) {
	svc, store := setupSvc(t)
	now := time.Now().UTC()
	store.articles["a1"] = &models.Article{
		ID: "a1", BlogID: "blog-1", Title: "Live", ContentJSON: `[{"id":"1"}]`,
		Status: models.StatusPublished, TagsJSON: "[]", PublishedAt: &now,
		CreatedAt: now, UpdatedAt: now,
	}
	d1, err := svc.EnsureDraft(context.Background(), "blog-1", "a1")
	if err != nil {
		t.Fatal(err)
	}
	d2, err := svc.EnsureDraft(context.Background(), "blog-1", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if d1.ID != d2.ID {
		t.Fatalf("expected same draft, got %s vs %s", d1.ID, d2.ID)
	}
	if len(store.drafts) != 1 {
		t.Fatalf("draft count=%d", len(store.drafts))
	}
}

func TestDraftEditLeavesArticleUnchanged(t *testing.T) {
	svc, store := setupSvc(t)
	now := time.Now().UTC()
	store.articles["a1"] = &models.Article{
		ID: "a1", BlogID: "blog-1", Title: "Live", ContentJSON: `[{"t":"old"}]`,
		Status: models.StatusPublished, TagsJSON: "[]", PublishedAt: &now,
		CreatedAt: now, UpdatedAt: now,
	}
	_, _ = svc.EnsureDraft(context.Background(), "blog-1", "a1")
	title := "New title"
	content := `[{"t":"new"}]`
	_, err := svc.UpdateDraft(context.Background(), "blog-1", "a1", article.DraftPatch{
		Title: &title, ContentJSON: &content,
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.articles["a1"].Title != "Live" {
		t.Fatalf("article mutated: %s", store.articles["a1"].Title)
	}
	if store.drafts["a1"].Title != "New title" {
		t.Fatalf("draft=%s", store.drafts["a1"].Title)
	}
}

func TestPublishDraftUpdatesAndDeletes(t *testing.T) {
	svc, store := setupSvc(t)
	now := time.Now().UTC()
	store.articles["a1"] = &models.Article{
		ID: "a1", BlogID: "blog-1", Title: "Live", ContentJSON: `[{"t":"old"}]`,
		Status: models.StatusPublished, TagsJSON: "[]", PublishedAt: &now,
		CreatedAt: now, UpdatedAt: now,
	}
	_, _ = svc.EnsureDraft(context.Background(), "blog-1", "a1")
	title := "Updated"
	_, _ = svc.UpdateDraft(context.Background(), "blog-1", "a1", article.DraftPatch{Title: &title})
	a, err := svc.PublishDraft(context.Background(), "blog-1", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if a.Title != "Updated" || a.Status != models.StatusPublished {
		t.Fatalf("%+v", a)
	}
	if _, ok := store.drafts["a1"]; ok {
		t.Fatal("draft should be deleted")
	}
}

func TestPublishArticleFromDraftStatus(t *testing.T) {
	svc, _ := setupSvc(t)
	a, _ := svc.Create(context.Background(), "blog-1")
	out, err := svc.PublishArticle(context.Background(), "blog-1", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != models.StatusPublished || out.PublishedAt == nil {
		t.Fatalf("%+v", out)
	}
}

func TestUpdatePublishedArticleRejected(t *testing.T) {
	svc, store := setupSvc(t)
	now := time.Now().UTC()
	store.articles["a1"] = &models.Article{
		ID: "a1", BlogID: "blog-1", Title: "Live", Status: models.StatusPublished,
		TagsJSON: "[]", ContentJSON: "[]", PublishedAt: &now, CreatedAt: now, UpdatedAt: now,
	}
	title := "Nope"
	_, err := svc.Update(context.Background(), "blog-1", "a1", article.UpdatePatch{Title: &title})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPublicGetIgnoresDraftArticles(t *testing.T) {
	svc, store := setupSvc(t)
	now := time.Now().UTC()
	store.articles["a1"] = &models.Article{
		ID: "a1", BlogID: "blog-1", Title: "Hidden", Status: models.StatusDraft,
		TagsJSON: "[]", ContentJSON: "[]", CreatedAt: now, UpdatedAt: now,
	}
	_, err := svc.GetPublished(context.Background(), "a1")
	if err == nil {
		t.Fatal("expected not found")
	}
}
