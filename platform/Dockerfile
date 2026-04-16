# Build context: repo root (not ./platform) so we can COPY both the Go
# source and the workspace-configs-templates directory that lives beside it.
# CI workflow sets `context: .` and `file: ./platform/Dockerfile`.

FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY platform/go.mod platform/go.sum ./
RUN go mod download
COPY platform/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /platform ./cmd/server

FROM alpine:3.20
# git is required by the `github` SourceResolver for plugin installs from
# GitHub URLs (POST /workspaces/:id/plugins {"source": "github://..."}).
RUN apk add --no-cache ca-certificates git tzdata
COPY --from=builder /platform /platform
COPY platform/migrations /migrations
# Default templates baked into the image so tenants boot with a working
# template picker. Phase B adds a registry + on-demand fetch for
# community templates; these curated defaults always ship in the image.
COPY workspace-configs-templates /workspace-configs-templates
EXPOSE 8080
CMD ["/platform"]
