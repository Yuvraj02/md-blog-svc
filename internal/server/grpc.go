package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	blogv1 "github.com/Yuvraj02/md-protos/proto/blog/v1"
	"github.com/marketing-digest/pkg/grpcx"

	"github.com/marketing-digest/blog-service/internal/app/article"
	articletransport "github.com/marketing-digest/blog-service/internal/app/article/transport"
	"github.com/marketing-digest/blog-service/internal/app/blog"
)

// Server owns process-level gRPC lifecycle only.
type Server struct {
	grpc   *grpc.Server
	health *health.Server
	log    *slog.Logger
	addr   string
}

func New(blogSvc *blog.Service, articleSvc *article.Service, log *slog.Logger, port int) *Server {
	s := grpc.NewServer(grpcx.UnaryServerInterceptors(log))
	hs := health.NewServer()

	// One registration: all blog/article/comment RPCs go through this handler.
	blogv1.RegisterBlogServiceServer(s, articletransport.NewHandler(blogSvc, articleSvc))
	healthpb.RegisterHealthServer(s, hs)
	reflection.Register(s)

	hs.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	hs.SetServingStatus("blog.v1.BlogService", healthpb.HealthCheckResponse_NOT_SERVING)

	return &Server{
		grpc:   s,
		health: hs,
		log:    log,
		addr:   fmt.Sprintf(":%d", port),
	}
}

func (s *Server) SetReady(ready bool) {
	status := healthpb.HealthCheckResponse_NOT_SERVING
	if ready {
		status = healthpb.HealthCheckResponse_SERVING
	}
	s.health.SetServingStatus("", status)
	s.health.SetServingStatus("blog.v1.BlogService", status)
}

func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.addr, err)
	}
	s.log.Info("grpc listening", "addr", s.addr)
	return s.grpc.Serve(ln)
}

func (s *Server) GracefulStop(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		s.grpc.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		s.grpc.Stop()
	}
}
