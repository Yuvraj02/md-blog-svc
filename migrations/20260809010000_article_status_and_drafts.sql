-- Article status + separate edit-draft table (one draft per article).

-- Status replaces is_draft boolean.
ALTER TABLE "articles" ADD COLUMN IF NOT EXISTS "status" text NOT NULL DEFAULT 'DRAFT';

UPDATE "articles"
SET "status" = CASE WHEN "is_draft" = true THEN 'DRAFT' ELSE 'PUBLISHED' END
WHERE "status" = 'DRAFT' OR "status" IS NOT NULL;

-- Re-run mapping for rows that already had default DRAFT but is_draft=false
UPDATE "articles"
SET "status" = CASE WHEN "is_draft" = true THEN 'DRAFT' ELSE 'PUBLISHED' END;

ALTER TABLE "articles" DROP COLUMN IF EXISTS "is_draft";

DROP INDEX IF EXISTS "idx_articles_published";
CREATE INDEX IF NOT EXISTS "idx_articles_status_published_at"
  ON "articles" ("status", "published_at");

-- Working copy while a published article is being revised.
CREATE TABLE IF NOT EXISTS "drafts" (
  "id" text PRIMARY KEY,
  "article_id" text NOT NULL UNIQUE REFERENCES "articles" ("id") ON DELETE CASCADE,
  "title" text NOT NULL DEFAULT '',
  "excerpt" text NOT NULL DEFAULT '',
  "image_url" text NOT NULL DEFAULT '',
  "video_url" text NOT NULL DEFAULT '',
  "tags_json" text NOT NULL DEFAULT '[]',
  "content_json" text NOT NULL DEFAULT '[]',
  "reading_time_minutes" integer NOT NULL DEFAULT 1,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS "idx_drafts_article_id" ON "drafts" ("article_id");
