-- Replace the tiny boilerplate schema with blogs + articles + comments.
-- Safe for local/dev: drops the old articles table.

DROP TABLE IF EXISTS "articles";

CREATE TABLE IF NOT EXISTS "blogs" (
  "id" text PRIMARY KEY,
  "name" text NOT NULL,
  "slug" text NOT NULL UNIQUE,
  "description" text NOT NULL DEFAULT '',
  "cover_image" text NOT NULL DEFAULT '',
  "owner_id" text NOT NULL,
  "upvotes" bigint NOT NULL DEFAULT 0,
  "total_views" bigint NOT NULL DEFAULT 0,
  "article_count" integer NOT NULL DEFAULT 0,
  "total_reading_time_minutes" integer NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS "articles" (
  "id" text PRIMARY KEY,
  "blog_id" text NOT NULL REFERENCES "blogs" ("id") ON DELETE CASCADE,
  "title" text NOT NULL DEFAULT '',
  "excerpt" text NOT NULL DEFAULT '',
  "image_url" text NOT NULL DEFAULT '',
  "video_url" text NOT NULL DEFAULT '',
  "tags_json" text NOT NULL DEFAULT '[]',
  "content_json" text NOT NULL DEFAULT '[]',
  "is_draft" boolean NOT NULL DEFAULT true,
  "upvotes" bigint NOT NULL DEFAULT 0,
  "views" bigint NOT NULL DEFAULT 0,
  "views_this_week" bigint NOT NULL DEFAULT 0,
  "reads" bigint NOT NULL DEFAULT 0,
  "reading_time_minutes" integer NOT NULL DEFAULT 1,
  "published_at" timestamptz NULL,
  "last_saved" timestamptz NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS "idx_articles_blog_id" ON "articles" ("blog_id");
CREATE INDEX IF NOT EXISTS "idx_articles_published" ON "articles" ("is_draft", "published_at");

CREATE TABLE IF NOT EXISTS "comments" (
  "id" text PRIMARY KEY,
  "article_id" text NOT NULL REFERENCES "articles" ("id") ON DELETE CASCADE,
  "name" text NOT NULL,
  "email" text NOT NULL,
  "website" text NOT NULL DEFAULT '',
  "body" text NOT NULL,
  "created_at" timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS "idx_comments_article_id" ON "comments" ("article_id");

-- Seed so Home / Discover work out of the box.
INSERT INTO "blogs" (
  "id", "name", "slug", "description", "cover_image", "owner_id",
  "upvotes", "total_views", "article_count", "total_reading_time_minutes",
  "created_at", "updated_at"
) VALUES
(
  'blog-1', 'Marketing Digest', 'marketing-digest',
  'Marketing stories, brand voice, and growth notes.', '',
  'user-1', 128, 8420, 2, 14, NOW(), NOW()
),
(
  'blog-2', 'Growth Lab', 'growth-lab',
  'Experiments in acquisition and retention.', '',
  'user-1', 54, 2100, 1, 6, NOW(), NOW()
)
ON CONFLICT ("id") DO NOTHING;

INSERT INTO "articles" (
  "id", "blog_id", "title", "excerpt", "image_url", "video_url",
  "tags_json", "content_json", "is_draft", "upvotes", "views",
  "views_this_week", "reads", "reading_time_minutes",
  "published_at", "last_saved", "created_at", "updated_at"
) VALUES
(
  'article-1', 'blog-1', 'The Architecture of Silence',
  'Why minimalism is scaling the digital world.', '/protest.jpg', '',
  '["design","minimalism"]',
  '[{"id":"b1","type":"paragraph","children":[{"type":"text","text":"Marketing work rewards clarity more than volume."}]},{"id":"b2","type":"heading","level":2,"children":[{"type":"text","text":"Start with voice, not volume"}]},{"id":"b3","type":"paragraph","children":[{"type":"text","text":"Brand voice is a set of constraints: what we always say and what we never say."}]}]',
  false, 42, 1200, 380, 210, 8,
  NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 day', NOW() - INTERVAL '10 days', NOW() - INTERVAL '1 day'
),
(
  'article-2', 'blog-1', 'Brand voice in 2026',
  'How marketers keep a consistent tone across channels.', '/protest.jpg', '',
  '["branding"]',
  '[{"id":"b1","type":"paragraph","children":[{"type":"text","text":"Consistent tone beats clever one-offs."}]}]',
  false, 19, 640, 120, 88, 6,
  NOW() - INTERVAL '5 days', NOW() - INTERVAL '4 days', NOW() - INTERVAL '20 days', NOW() - INTERVAL '4 days'
),
(
  'article-3', 'blog-2', 'A/B tests that actually ship',
  'Keep experiments small and decisive.', '/protest.jpg', '',
  '["growth"]',
  '[{"id":"b1","type":"paragraph","children":[{"type":"text","text":"Ship small tests. Decide fast."}]}]',
  false, 7, 210, 95, 40, 6,
  NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day', NOW() - INTERVAL '8 days', NOW() - INTERVAL '1 day'
),
(
  'article-4', 'blog-1', 'Untitled draft', '', '', '',
  '[]',
  '[{"id":"b1","type":"paragraph","children":[{"type":"text","text":"Still cooking…"}]}]',
  true, 0, 0, 0, 0, 1,
  NULL, NOW(), NOW(), NOW()
)
ON CONFLICT ("id") DO NOTHING;

INSERT INTO "comments" ("id", "article_id", "name", "email", "website", "body", "created_at")
VALUES (
  'c-1', 'article-1', 'Aisha Khan', 'aisha@example.com', '',
  'This framing of silence as strategy really stuck with me — sharing with my team.',
  NOW() - INTERVAL '1 day'
)
ON CONFLICT ("id") DO NOTHING;
