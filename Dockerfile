# Build from workspace root (parent of backend/ and md-protos/):
#   docker build -f backend/services/blog-service/Dockerfile -t marketing-digest-blog .
#
# Same image serves the gRPC process (Deployment) and Atlas migrate (Job).
# Job overrides command to: atlas migrate apply --dir file:///migrations ...

FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git

COPY md-protos ./protos
COPY backend/pkg ./backend/pkg
COPY backend/services/blog-service ./backend/services/blog-service

WORKDIR /src/backend/services/blog-service
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/blog-service ./cmd/server

FROM arigaio/atlas:latest AS atlas

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/blog-service /blog-service
COPY --from=atlas /atlas /usr/local/bin/atlas
COPY --from=build /src/backend/services/blog-service/migrations /migrations
USER nobody
EXPOSE 50051
ENTRYPOINT ["/blog-service"]
