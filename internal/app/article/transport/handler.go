package transport

import (
	"context"
	"encoding/json"
	"time"

	blogv1 "github.com/Yuvraj02/md-protos/proto/blog/v1"
	"github.com/marketing-digest/pkg/grpcx"

	"github.com/marketing-digest/blog-service/internal/app/article"
	articlemodels "github.com/marketing-digest/blog-service/internal/app/article/models"
	"github.com/marketing-digest/blog-service/internal/app/blog"
	blogmodels "github.com/marketing-digest/blog-service/internal/app/blog/models"
)

// Handler is the single gRPC adapter for BlogService (blogs + articles + comments).
type Handler struct {
	blogv1.UnimplementedBlogServiceServer
	blogs    *blog.Service
	articles *article.Service
}

func NewHandler(blogs *blog.Service, articles *article.Service) *Handler {
	return &Handler{blogs: blogs, articles: articles}
}

func (h *Handler) Ping(_ context.Context, req *blogv1.PingRequest) (*blogv1.PingResponse, error) {
	msg := "pong"
	if req != nil && req.GetMessage() != "" {
		msg = req.GetMessage()
	}
	return &blogv1.PingResponse{Message: msg, Service: "blog-service"}, nil
}

func (h *Handler) ListBlogs(ctx context.Context, _ *blogv1.ListBlogsRequest) (*blogv1.ListBlogsResponse, error) {
	list, err := h.blogs.List(ctx)
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	out := &blogv1.ListBlogsResponse{}
	for _, b := range list {
		out.Blogs = append(out.Blogs, toProtoBlog(b))
	}
	return out, nil
}

func (h *Handler) GetBlog(ctx context.Context, req *blogv1.GetBlogRequest) (*blogv1.GetBlogResponse, error) {
	b, err := h.blogs.Get(ctx, req.GetBlogId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.GetBlogResponse{Blog: toProtoBlog(b)}, nil
}

func (h *Handler) CreateBlog(ctx context.Context, req *blogv1.CreateBlogRequest) (*blogv1.CreateBlogResponse, error) {
	b, err := h.blogs.Create(ctx, req.GetName(), req.GetDescription(), req.GetOwnerId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.CreateBlogResponse{Blog: toProtoBlog(b)}, nil
}

func (h *Handler) UpdateBlog(ctx context.Context, req *blogv1.UpdateBlogRequest) (*blogv1.UpdateBlogResponse, error) {
	// Empty string means "leave this field alone" (proto3 has no optional presence here).
	var name, desc, cover, slug *string
	if req.GetName() != "" {
		v := req.GetName()
		name = &v
	}
	if req.GetDescription() != "" {
		v := req.GetDescription()
		desc = &v
	}
	if req.GetCoverImage() != "" {
		v := req.GetCoverImage()
		cover = &v
	}
	if req.GetSlug() != "" {
		v := req.GetSlug()
		slug = &v
	}
	b, err := h.blogs.Update(ctx, req.GetBlogId(), name, desc, cover, slug)
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.UpdateBlogResponse{Blog: toProtoBlog(b)}, nil
}

func (h *Handler) UpvoteBlog(ctx context.Context, req *blogv1.UpvoteBlogRequest) (*blogv1.UpvoteBlogResponse, error) {
	b, err := h.blogs.Upvote(ctx, req.GetBlogId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.UpvoteBlogResponse{Blog: toProtoBlog(b)}, nil
}

func (h *Handler) ListArticles(ctx context.Context, req *blogv1.ListArticlesRequest) (*blogv1.ListArticlesResponse, error) {
	list, err := h.articles.ListByBlog(ctx, req.GetBlogId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	out := &blogv1.ListArticlesResponse{}
	for _, a := range list {
		out.Articles = append(out.Articles, toProtoArticle(a))
	}
	return out, nil
}

func (h *Handler) GetArticle(ctx context.Context, req *blogv1.GetArticleRequest) (*blogv1.GetArticleResponse, error) {
	a, err := h.articles.Get(ctx, req.GetBlogId(), req.GetArticleId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.GetArticleResponse{Article: toProtoArticle(a)}, nil
}

func (h *Handler) CreateArticle(ctx context.Context, req *blogv1.CreateArticleRequest) (*blogv1.CreateArticleResponse, error) {
	a, err := h.articles.Create(ctx, req.GetBlogId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.CreateArticleResponse{Article: toProtoArticle(a)}, nil
}

func (h *Handler) UpdateArticle(ctx context.Context, req *blogv1.UpdateArticleRequest) (*blogv1.UpdateArticleResponse, error) {
	// Saves a never-published DRAFT article (not a live published piece).
	patch := article.UpdatePatch{}
	if req.GetTitle() != "" || req.GetContentJson() != "" {
		t := req.GetTitle()
		patch.Title = &t
	}
	if req.GetExcerpt() != "" {
		e := req.GetExcerpt()
		patch.Excerpt = &e
	}
	if len(req.GetTags()) > 0 {
		patch.HasTags = true
		patch.Tags = req.GetTags()
	}
	if req.GetMedia() != nil {
		img, vid := req.Media.GetImageUrl(), req.Media.GetVideoUrl()
		patch.ImageURL = &img
		patch.VideoURL = &vid
	}
	if req.GetContentJson() != "" {
		c := req.GetContentJson()
		patch.ContentJSON = &c
	}
	if req.GetReadingTimeMinutes() > 0 {
		m := int(req.GetReadingTimeMinutes())
		patch.ReadingTimeMinutes = &m
	}
	a, err := h.articles.Update(ctx, req.GetBlogId(), req.GetArticleId(), patch)
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.UpdateArticleResponse{Article: toProtoArticle(a)}, nil
}

func (h *Handler) DeleteArticle(ctx context.Context, req *blogv1.DeleteArticleRequest) (*blogv1.DeleteArticleResponse, error) {
	if err := h.articles.Delete(ctx, req.GetBlogId(), req.GetArticleId()); err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.DeleteArticleResponse{}, nil
}

func (h *Handler) UpvoteArticle(ctx context.Context, req *blogv1.UpvoteArticleRequest) (*blogv1.UpvoteArticleResponse, error) {
	a, err := h.articles.Upvote(ctx, req.GetBlogId(), req.GetArticleId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.UpvoteArticleResponse{Article: toProtoArticle(a)}, nil
}

// PublishArticle turns a never-published DRAFT into PUBLISHED.
func (h *Handler) PublishArticle(ctx context.Context, req *blogv1.PublishArticleRequest) (*blogv1.PublishArticleResponse, error) {
	a, err := h.articles.PublishArticle(ctx, req.GetBlogId(), req.GetArticleId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.PublishArticleResponse{Article: toProtoArticle(a)}, nil
}

// EnsureDraft creates or reuses the edit-draft for a PUBLISHED article.
func (h *Handler) EnsureDraft(ctx context.Context, req *blogv1.EnsureDraftRequest) (*blogv1.EnsureDraftResponse, error) {
	d, err := h.articles.EnsureDraft(ctx, req.GetBlogId(), req.GetArticleId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.EnsureDraftResponse{Draft: toProtoDraft(d, req.GetBlogId())}, nil
}

func (h *Handler) GetDraft(ctx context.Context, req *blogv1.GetDraftRequest) (*blogv1.GetDraftResponse, error) {
	d, err := h.articles.GetDraft(ctx, req.GetBlogId(), req.GetArticleId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.GetDraftResponse{Draft: toProtoDraft(d, req.GetBlogId())}, nil
}

func (h *Handler) UpdateDraft(ctx context.Context, req *blogv1.UpdateDraftRequest) (*blogv1.UpdateDraftResponse, error) {
	patch := article.DraftPatch{}
	if req.GetTitle() != "" || req.GetContentJson() != "" {
		t := req.GetTitle()
		patch.Title = &t
	}
	if req.GetExcerpt() != "" {
		e := req.GetExcerpt()
		patch.Excerpt = &e
	}
	if len(req.GetTags()) > 0 {
		patch.HasTags = true
		patch.Tags = req.GetTags()
	}
	if req.GetMedia() != nil {
		img, vid := req.Media.GetImageUrl(), req.Media.GetVideoUrl()
		patch.ImageURL = &img
		patch.VideoURL = &vid
	}
	if req.GetContentJson() != "" {
		c := req.GetContentJson()
		patch.ContentJSON = &c
	}
	if req.GetReadingTimeMinutes() > 0 {
		m := int(req.GetReadingTimeMinutes())
		patch.ReadingTimeMinutes = &m
	}
	d, err := h.articles.UpdateDraft(ctx, req.GetBlogId(), req.GetArticleId(), patch)
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.UpdateDraftResponse{Draft: toProtoDraft(d, req.GetBlogId())}, nil
}

// PublishDraft copies draft onto the live article and deletes the draft.
func (h *Handler) PublishDraft(ctx context.Context, req *blogv1.PublishDraftRequest) (*blogv1.PublishDraftResponse, error) {
	a, err := h.articles.PublishDraft(ctx, req.GetBlogId(), req.GetArticleId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.PublishDraftResponse{Article: toProtoArticle(a)}, nil
}

func (h *Handler) ListPublishedArticles(ctx context.Context, _ *blogv1.ListPublishedArticlesRequest) (*blogv1.ListPublishedArticlesResponse, error) {
	list, err := h.articles.ListPublished(ctx)
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	out := &blogv1.ListPublishedArticlesResponse{}
	for _, a := range list {
		out.Articles = append(out.Articles, toProtoArticle(a))
	}
	return out, nil
}

func (h *Handler) ListTrendingArticles(ctx context.Context, _ *blogv1.ListTrendingArticlesRequest) (*blogv1.ListTrendingArticlesResponse, error) {
	list, err := h.articles.ListTrending(ctx)
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	out := &blogv1.ListTrendingArticlesResponse{}
	for _, a := range list {
		out.Articles = append(out.Articles, toProtoArticle(a))
	}
	return out, nil
}

func (h *Handler) GetPublishedArticle(ctx context.Context, req *blogv1.GetPublishedArticleRequest) (*blogv1.GetPublishedArticleResponse, error) {
	a, err := h.articles.GetPublished(ctx, req.GetArticleId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.GetPublishedArticleResponse{Article: toProtoArticle(a)}, nil
}

func (h *Handler) ListRelatedArticles(ctx context.Context, req *blogv1.ListRelatedArticlesRequest) (*blogv1.ListRelatedArticlesResponse, error) {
	list, err := h.articles.ListRelated(ctx, req.GetArticleId(), int(req.GetLimit()))
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	out := &blogv1.ListRelatedArticlesResponse{}
	for _, a := range list {
		out.Articles = append(out.Articles, toProtoArticle(a))
	}
	return out, nil
}

func (h *Handler) RecordArticleView(ctx context.Context, req *blogv1.RecordArticleViewRequest) (*blogv1.RecordArticleViewResponse, error) {
	if err := h.articles.RecordView(ctx, req.GetArticleId()); err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.RecordArticleViewResponse{}, nil
}

func (h *Handler) RecordArticleRead(ctx context.Context, req *blogv1.RecordArticleReadRequest) (*blogv1.RecordArticleReadResponse, error) {
	if err := h.articles.RecordRead(ctx, req.GetArticleId()); err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.RecordArticleReadResponse{}, nil
}

func (h *Handler) ListComments(ctx context.Context, req *blogv1.ListCommentsRequest) (*blogv1.ListCommentsResponse, error) {
	list, err := h.articles.ListComments(ctx, req.GetArticleId())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	out := &blogv1.ListCommentsResponse{}
	for _, c := range list {
		out.Comments = append(out.Comments, toProtoComment(c))
	}
	return out, nil
}

func (h *Handler) CreateComment(ctx context.Context, req *blogv1.CreateCommentRequest) (*blogv1.CreateCommentResponse, error) {
	c, err := h.articles.CreateComment(ctx, req.GetArticleId(), req.GetName(), req.GetEmail(), req.GetWebsite(), req.GetBody())
	if err != nil {
		return nil, grpcx.ToStatus(err)
	}
	return &blogv1.CreateCommentResponse{Comment: toProtoComment(c)}, nil
}

func toProtoBlog(b *blogmodels.Blog) *blogv1.Blog {
	if b == nil {
		return nil
	}
	return &blogv1.Blog{
		Id: b.ID, Name: b.Name, Slug: b.Slug, Description: b.Description,
		CoverImage: b.CoverImage, OwnerId: b.OwnerID, Upvotes: b.Upvotes,
		Stats: &blogv1.BlogStats{
			TotalViews: b.TotalViews, ArticleCount: int32(b.ArticleCount),
			TotalReadingTimeMinutes: int32(b.TotalReadingTimeMinutes),
		},
		CreatedAt: b.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: b.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toProtoArticle(a *articlemodels.Article) *blogv1.Article {
	if a == nil {
		return nil
	}
	var tags []string
	_ = json.Unmarshal([]byte(a.TagsJSON), &tags)
	out := &blogv1.Article{
		Id: a.ID, BlogId: a.BlogID, Title: a.Title, Excerpt: a.Excerpt,
		Media: &blogv1.ArticleMedia{ImageUrl: a.ImageURL, VideoUrl: a.VideoURL},
		Tags: tags, ContentJson: a.ContentJSON, Status: string(a.Status),
		Upvotes: a.Upvotes, Views: a.Views, ViewsThisWeek: a.ViewsThisWeek, Reads: a.Reads,
		ReadingTimeMinutes: int32(a.ReadingTimeMinutes),
		HasDraft:  a.HasDraft,
		CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: a.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if a.PublishedAt != nil {
		out.PublishedAt = a.PublishedAt.UTC().Format(time.RFC3339)
	}
	if a.LastSaved != nil {
		out.LastSaved = a.LastSaved.UTC().Format(time.RFC3339)
	}
	for i := range a.Comments {
		out.Comments = append(out.Comments, toProtoComment(&a.Comments[i]))
	}
	return out
}

func toProtoDraft(d *articlemodels.Draft, blogID string) *blogv1.Draft {
	if d == nil {
		return nil
	}
	var tags []string
	_ = json.Unmarshal([]byte(d.TagsJSON), &tags)
	return &blogv1.Draft{
		Id: d.ID, ArticleId: d.ArticleID, BlogId: blogID,
		Title: d.Title, Excerpt: d.Excerpt,
		Media: &blogv1.ArticleMedia{ImageUrl: d.ImageURL, VideoUrl: d.VideoURL},
		Tags: tags, ContentJson: d.ContentJSON,
		ReadingTimeMinutes: int32(d.ReadingTimeMinutes),
		CreatedAt: d.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: d.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toProtoComment(c *articlemodels.Comment) *blogv1.Comment {
	if c == nil {
		return nil
	}
	return &blogv1.Comment{
		Id: c.ID, ArticleId: c.ArticleID, Name: c.Name, Email: c.Email,
		Website: c.Website, Body: c.Body,
		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339),
	}
}
