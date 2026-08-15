# Build from this service repo root:
#   docker build -t marketing-digest-blog .
#
# Same image runs the gRPC server (Deployment) and Atlas migrations (Job).

FROM golang:1.25-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
COPY pkg ./pkg
COPY . .

RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" \
    -o /out/blog-service ./cmd/server


# Final image contains both blog-service and Atlas.
FROM arigaio/atlas:latest-alpine

COPY --from=build /out/blog-service /blog-service
COPY --from=build /src/migrations /migrations

USER nobody

EXPOSE 50051

ENTRYPOINT ["/blog-service"]