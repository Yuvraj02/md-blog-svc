# Build from this service repo root:
#   docker build -t marketing-digest-blog .
#
# Same image runs the gRPC server (Deployment) and Atlas migrate (Job).
# Job overrides command: atlas migrate apply --dir file:///migrations ...

FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
COPY pkg ./pkg
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/blog-service ./cmd/server

FROM arigaio/atlas:latest AS atlas

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/blog-service /blog-service
COPY --from=atlas /atlas /usr/local/bin/atlas
COPY --from=build /src/migrations /migrations
USER nobody
EXPOSE 50051
ENTRYPOINT ["/blog-service"]
