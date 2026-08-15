// Package grpcx provides shared gRPC interceptors and helpers.
package grpcx

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/marketing-digest/pkg/errorsx"
	"github.com/marketing-digest/pkg/logger"
)

const requestIDKey = "x-request-id"

// UnaryServerInterceptors returns the default interceptor chain.
func UnaryServerInterceptors(log *slog.Logger) grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(
		requestIDInterceptor(),
		loggingInterceptor(log),
		recoveryInterceptor(log),
	)
}

func requestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		rid := requestIDFromIncoming(ctx)
		ctx = context.WithValue(ctx, requestIDContextKey{}, rid)
		return handler(ctx, req)
	}
}

func loggingInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		rid := RequestID(ctx)
		reqLog := log.With("request_id", rid, "method", info.FullMethod)
		ctx = logger.WithContext(ctx, reqLog)

		resp, err := handler(ctx, req)

		attrs := []any{
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if err != nil {
			reqLog.Error("rpc finished", append(attrs, "error", err.Error())...)
		} else {
			reqLog.Info("rpc finished", attrs...)
		}
		return resp, err
	}
}

func recoveryInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered", "method", info.FullMethod, "panic", r)
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

type requestIDContextKey struct{}

// RequestID extracts the request id from context.
func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDContextKey{}).(string); ok && v != "" {
		return v
	}
	return ""
}

func requestIDFromIncoming(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(requestIDKey); len(vals) > 0 && vals[0] != "" {
			return vals[0]
		}
	}
	return uuid.NewString()
}

// ToStatus maps domain/sentinel errors to gRPC status codes.
// Never leaks internal/database details to clients.
func ToStatus(err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok {
		return st.Err()
	}
	switch {
	case errors.Is(err, errorsx.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, errorsx.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, "invalid argument")
	case errors.Is(err, errorsx.ErrUnauthenticated):
		return status.Error(codes.Unauthenticated, "unauthenticated")
	case errors.Is(err, errorsx.ErrPermissionDenied):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, errorsx.ErrConflict):
		return status.Error(codes.AlreadyExists, "conflict")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
