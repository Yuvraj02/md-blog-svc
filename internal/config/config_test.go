package config_test

import (
	"testing"

	"github.com/marketing-digest/blog-service/internal/config"
)

func TestLoadOK(t *testing.T) {
	t.Setenv("DATABASE_HOST", "localhost")
	t.Setenv("DATABASE_NAME", "marketing_digest_blog")
	t.Setenv("DATABASE_USER", "blog")
	t.Setenv("DATABASE_PASSWORD", "secret")
	t.Setenv("GRPC_PORT", "50052")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Service != "blog-service" {
		t.Fatalf("service=%s", cfg.Service)
	}
}
