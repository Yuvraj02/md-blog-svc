package article

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/marketing-digest/pkg/errorsx"

	"github.com/marketing-digest/blog-service/internal/app/article/models"
)

type articleRow struct {
	ID                 string     `gorm:"primaryKey"`
	BlogID             string     `gorm:"column:blog_id;index;not null"`
	Title              string     `gorm:"not null;default:''"`
	Excerpt            string     `gorm:"not null;default:''"`
	ImageURL           string     `gorm:"column:image_url;not null;default:''"`
	VideoURL           string     `gorm:"column:video_url;not null;default:''"`
	TagsJSON           string     `gorm:"column:tags_json;not null;default:'[]'"`
	ContentJSON        string     `gorm:"column:content_json;not null;default:'[]'"`
	Status             string     `gorm:"column:status;not null;default:'DRAFT'"`
	Upvotes             int64      `gorm:"not null;default:0"`
	Views              int64      `gorm:"not null;default:0"`
	ViewsThisWeek      int64      `gorm:"column:views_this_week;not null;default:0"`
	Reads              int64      `gorm:"not null;default:0"`
	ReadingTimeMinutes int        `gorm:"column:reading_time_minutes;not null;default:1"`
	PublishedAt        *time.Time `gorm:"column:published_at"`
	LastSaved          *time.Time `gorm:"column:last_saved"`
	CreatedAt          time.Time  `gorm:"not null"`
	UpdatedAt          time.Time  `gorm:"not null"`
}

func (articleRow) TableName() string { return "articles" }

type draftRow struct {
	ID                 string    `gorm:"primaryKey"`
	ArticleID          string    `gorm:"column:article_id;uniqueIndex;not null"`
	Title              string    `gorm:"not null;default:''"`
	Excerpt            string    `gorm:"not null;default:''"`
	ImageURL           string    `gorm:"column:image_url;not null;default:''"`
	VideoURL           string    `gorm:"column:video_url;not null;default:''"`
	TagsJSON           string    `gorm:"column:tags_json;not null;default:'[]'"`
	ContentJSON        string    `gorm:"column:content_json;not null;default:'[]'"`
	ReadingTimeMinutes int       `gorm:"column:reading_time_minutes;not null;default:1"`
	CreatedAt          time.Time `gorm:"not null"`
	UpdatedAt          time.Time `gorm:"not null"`
}

func (draftRow) TableName() string { return "drafts" }

type commentRow struct {
	ID        string    `gorm:"primaryKey"`
	ArticleID string    `gorm:"column:article_id;index;not null"`
	Name      string    `gorm:"not null"`
	Email     string    `gorm:"not null"`
	Website   string    `gorm:"not null;default:''"`
	Body      string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (commentRow) TableName() string { return "comments" }

type GORMStore struct{ db *gorm.DB }

func NewGORMStore(db *gorm.DB) *GORMStore { return &GORMStore{db: db} }

func (s *GORMStore) ListByBlog(ctx context.Context, blogID string) ([]*models.Article, error) {
	var rows []articleRow
	err := s.db.WithContext(ctx).Where("blog_id = ?", blogID).Order("updated_at desc").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list articles: %w", err)
	}
	out := mapArticles(rows)
	ids := make([]string, 0, len(out))
	for _, a := range out {
		ids = append(ids, a.ID)
	}
	has, _ := s.HasDraftMap(ctx, ids)
	for _, a := range out {
		a.HasDraft = has[a.ID]
	}
	return out, nil
}

func (s *GORMStore) Get(ctx context.Context, blogID, articleID string) (*models.Article, error) {
	var m articleRow
	err := s.db.WithContext(ctx).First(&m, "id = ? AND blog_id = ?", articleID, blogID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get article: %w", err)
	}
	a := fromArticleRow(&m)
	_ = s.LoadComments(ctx, a)
	has, _ := s.HasDraftMap(ctx, []string{a.ID})
	a.HasDraft = has[a.ID]
	return a, nil
}

func (s *GORMStore) GetByID(ctx context.Context, articleID string) (*models.Article, error) {
	var m articleRow
	err := s.db.WithContext(ctx).First(&m, "id = ?", articleID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get article by id: %w", err)
	}
	return fromArticleRow(&m), nil
}

func (s *GORMStore) GetPublished(ctx context.Context, articleID string) (*models.Article, error) {
	var m articleRow
	err := s.db.WithContext(ctx).
		First(&m, "id = ? AND status = ?", articleID, string(models.StatusPublished)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get published: %w", err)
	}
	a := fromArticleRow(&m)
	_ = s.LoadComments(ctx, a)
	return a, nil
}

func (s *GORMStore) ListPublished(ctx context.Context) ([]*models.Article, error) {
	var rows []articleRow
	err := s.db.WithContext(ctx).
		Where("status = ?", string(models.StatusPublished)).
		Order("published_at desc").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list published: %w", err)
	}
	return mapArticles(rows), nil
}

func (s *GORMStore) ListTrending(ctx context.Context) ([]*models.Article, error) {
	var rows []articleRow
	err := s.db.WithContext(ctx).
		Where("status = ?", string(models.StatusPublished)).
		Order("views_this_week desc").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list trending: %w", err)
	}
	return mapArticles(rows), nil
}

func (s *GORMStore) Create(ctx context.Context, a *models.Article) error {
	m := toArticleRow(a)
	if err := s.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("create article: %w", err)
	}
	*a = *fromArticleRow(m)
	return nil
}

func (s *GORMStore) Update(ctx context.Context, a *models.Article) error {
	m := toArticleRow(a)
	if err := s.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("update article: %w", err)
	}
	*a = *fromArticleRow(m)
	return nil
}

func (s *GORMStore) Delete(ctx context.Context, blogID, articleID string) error {
	res := s.db.WithContext(ctx).Where("id = ? AND blog_id = ?", articleID, blogID).Delete(&articleRow{})
	if res.Error != nil {
		return fmt.Errorf("delete article: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}
	return nil
}

func (s *GORMStore) ListComments(ctx context.Context, articleID string) ([]*models.Comment, error) {
	var rows []commentRow
	err := s.db.WithContext(ctx).Where("article_id = ?", articleID).Order("created_at desc").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	out := make([]*models.Comment, 0, len(rows))
	for i := range rows {
		out = append(out, fromCommentRow(&rows[i]))
	}
	return out, nil
}

func (s *GORMStore) CreateComment(ctx context.Context, c *models.Comment) error {
	m := toCommentRow(c)
	if err := s.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	*c = *fromCommentRow(m)
	return nil
}

func (s *GORMStore) LoadComments(ctx context.Context, articles ...*models.Article) error {
	for _, a := range articles {
		if a == nil {
			continue
		}
		list, err := s.ListComments(ctx, a.ID)
		if err != nil {
			return err
		}
		a.Comments = make([]models.Comment, 0, len(list))
		for _, c := range list {
			a.Comments = append(a.Comments, *c)
		}
	}
	return nil
}

func (s *GORMStore) GetDraftByArticleID(ctx context.Context, articleID string) (*models.Draft, error) {
	var m draftRow
	err := s.db.WithContext(ctx).First(&m, "article_id = ?", articleID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get draft: %w", err)
	}
	return fromDraftRow(&m), nil
}

func (s *GORMStore) CreateDraft(ctx context.Context, d *models.Draft) error {
	m := toDraftRow(d)
	res := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(m)
	if res.Error != nil {
		return fmt.Errorf("create draft: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		existing, err := s.GetDraftByArticleID(ctx, d.ArticleID)
		if err != nil {
			return err
		}
		*d = *existing
		return nil
	}
	*d = *fromDraftRow(m)
	return nil
}

func (s *GORMStore) UpdateDraft(ctx context.Context, d *models.Draft) error {
	m := toDraftRow(d)
	if err := s.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("update draft: %w", err)
	}
	*d = *fromDraftRow(m)
	return nil
}

func (s *GORMStore) DeleteDraft(ctx context.Context, articleID string) error {
	res := s.db.WithContext(ctx).Where("article_id = ?", articleID).Delete(&draftRow{})
	if res.Error != nil {
		return fmt.Errorf("delete draft: %w", res.Error)
	}
	return nil
}

func (s *GORMStore) HasDraftMap(ctx context.Context, articleIDs []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(articleIDs) == 0 {
		return out, nil
	}
	var rows []draftRow
	if err := s.db.WithContext(ctx).Select("article_id").Where("article_id IN ?", articleIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ArticleID] = true
	}
	return out, nil
}

func (s *GORMStore) PublishDraftTx(ctx context.Context, blogID, articleID string) (*models.Article, error) {
	var result *models.Article
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var art articleRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&art, "id = ? AND blog_id = ?", articleID, blogID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errorsx.ErrNotFound
			}
			return err
		}
		if art.Status != string(models.StatusPublished) {
			return fmt.Errorf("%w: only published articles use edit-drafts", errorsx.ErrInvalidArgument)
		}

		var dr draftRow
		if err := tx.First(&dr, "article_id = ?", articleID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errorsx.ErrNotFound
			}
			return err
		}

		now := time.Now().UTC()
		art.Title = dr.Title
		art.Excerpt = dr.Excerpt
		art.ImageURL = dr.ImageURL
		art.VideoURL = dr.VideoURL
		art.TagsJSON = dr.TagsJSON
		art.ContentJSON = dr.ContentJSON
		art.ReadingTimeMinutes = dr.ReadingTimeMinutes
		art.Status = string(models.StatusPublished)
		art.LastSaved = &now
		art.UpdatedAt = now
		if art.PublishedAt == nil {
			art.PublishedAt = &now
		}
		if err := tx.Save(&art).Error; err != nil {
			return err
		}
		if err := tx.Where("article_id = ?", articleID).Delete(&draftRow{}).Error; err != nil {
			return err
		}
		result = fromArticleRow(&art)
		return nil
	})
	if err != nil {
		return nil, err
	}
	_ = s.LoadComments(ctx, result)
	return result, nil
}

func mapArticles(rows []articleRow) []*models.Article {
	out := make([]*models.Article, 0, len(rows))
	for i := range rows {
		out = append(out, fromArticleRow(&rows[i]))
	}
	return out
}

func toArticleRow(a *models.Article) *articleRow {
	return &articleRow{
		ID: a.ID, BlogID: a.BlogID, Title: a.Title, Excerpt: a.Excerpt,
		ImageURL: a.ImageURL, VideoURL: a.VideoURL, TagsJSON: a.TagsJSON,
		ContentJSON: a.ContentJSON, Status: string(a.Status), Upvotes: a.Upvotes,
		Views: a.Views, ViewsThisWeek: a.ViewsThisWeek, Reads: a.Reads,
		ReadingTimeMinutes: a.ReadingTimeMinutes, PublishedAt: a.PublishedAt,
		LastSaved: a.LastSaved, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

func fromArticleRow(m *articleRow) *models.Article {
	return &models.Article{
		ID: m.ID, BlogID: m.BlogID, Title: m.Title, Excerpt: m.Excerpt,
		ImageURL: m.ImageURL, VideoURL: m.VideoURL, TagsJSON: m.TagsJSON,
		ContentJSON: m.ContentJSON, Status: models.Status(m.Status), Upvotes: m.Upvotes,
		Views: m.Views, ViewsThisWeek: m.ViewsThisWeek, Reads: m.Reads,
		ReadingTimeMinutes: m.ReadingTimeMinutes, PublishedAt: m.PublishedAt,
		LastSaved: m.LastSaved, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		Comments: []models.Comment{},
	}
}

func toDraftRow(d *models.Draft) *draftRow {
	return &draftRow{
		ID: d.ID, ArticleID: d.ArticleID, Title: d.Title, Excerpt: d.Excerpt,
		ImageURL: d.ImageURL, VideoURL: d.VideoURL, TagsJSON: d.TagsJSON,
		ContentJSON: d.ContentJSON, ReadingTimeMinutes: d.ReadingTimeMinutes,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

func fromDraftRow(m *draftRow) *models.Draft {
	return &models.Draft{
		ID: m.ID, ArticleID: m.ArticleID, Title: m.Title, Excerpt: m.Excerpt,
		ImageURL: m.ImageURL, VideoURL: m.VideoURL, TagsJSON: m.TagsJSON,
		ContentJSON: m.ContentJSON, ReadingTimeMinutes: m.ReadingTimeMinutes,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func toCommentRow(c *models.Comment) *commentRow {
	return &commentRow{
		ID: c.ID, ArticleID: c.ArticleID, Name: c.Name, Email: c.Email,
		Website: c.Website, Body: c.Body, CreatedAt: c.CreatedAt,
	}
}

func fromCommentRow(m *commentRow) *models.Comment {
	return &models.Comment{
		ID: m.ID, ArticleID: m.ArticleID, Name: m.Name, Email: m.Email,
		Website: m.Website, Body: m.Body, CreatedAt: m.CreatedAt,
	}
}
