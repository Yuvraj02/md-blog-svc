package article

import (
	"context"

	"github.com/marketing-digest/blog-service/internal/app/article/models"
)

// Store persists articles, comments, and edit-drafts.
type Store interface {
	ListByBlog(ctx context.Context, blogID string) ([]*models.Article, error)
	Get(ctx context.Context, blogID, articleID string) (*models.Article, error)
	GetByID(ctx context.Context, articleID string) (*models.Article, error)
	GetPublished(ctx context.Context, articleID string) (*models.Article, error)
	ListPublished(ctx context.Context) ([]*models.Article, error)
	ListTrending(ctx context.Context) ([]*models.Article, error)
	Create(ctx context.Context, a *models.Article) error
	Update(ctx context.Context, a *models.Article) error
	Delete(ctx context.Context, blogID, articleID string) error

	ListComments(ctx context.Context, articleID string) ([]*models.Comment, error)
	CreateComment(ctx context.Context, c *models.Comment) error
	LoadComments(ctx context.Context, articles ...*models.Article) error

	GetDraftByArticleID(ctx context.Context, articleID string) (*models.Draft, error)
	CreateDraft(ctx context.Context, d *models.Draft) error
	UpdateDraft(ctx context.Context, d *models.Draft) error
	DeleteDraft(ctx context.Context, articleID string) error
	// HasDraftMap returns articleID -> true for articles that have an edit-draft.
	HasDraftMap(ctx context.Context, articleIDs []string) (map[string]bool, error)

	// PublishDraftTx copies draft onto article, sets PUBLISHED, deletes draft — atomically.
	PublishDraftTx(ctx context.Context, blogID, articleID string) (*models.Article, error)
}
