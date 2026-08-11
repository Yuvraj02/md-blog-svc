# Build from workspace root (parent of backend/ and protos/):
#   docker build -f backend/services/blog-service/Dockerfile -t marketing-digest-blog-service .
FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git

COPY protos ./protos
COPY backend/pkg ./backend/pkg
COPY backend/services/blog-service ./backend/services/blog-service

WORKDIR /src/backend/services/blog-service
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/blog-service ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/blog-service /blog-service
USER nonroot:nonroot
EXPOSE 50051
ENTRYPOINT ["/blog-service"]
