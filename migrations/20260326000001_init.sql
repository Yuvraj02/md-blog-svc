-- Blog Service initial schema (articles).
-- Forward-only: do not edit after release; add a new migration instead.

CREATE TABLE IF NOT EXISTS "articles" (
  "id" uuid PRIMARY KEY,
  "author_id" uuid NOT NULL,
  "title" varchar(500) NOT NULL,
  "excerpt" text NOT NULL DEFAULT '',
  "content" text NOT NULL DEFAULT '',
  "status" varchar(32) NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS "idx_articles_author_id" ON "articles" ("author_id");
CREATE INDEX IF NOT EXISTS "idx_articles_status" ON "articles" ("status");
