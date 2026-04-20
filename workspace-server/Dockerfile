# Platform-only image (no canvas). Used by publish-platform-image workflow
# for GHCR + Fly registry. Tenant image uses Dockerfile.tenant instead.
#
# Build context: repo root.

FROM golang:1.25-alpine AS builder
WORKDIR /app
# Plugin source for replace directive in go.mod
COPY molecule-ai-plugin-github-app-auth/ /plugin/
COPY workspace-server/go.mod workspace-server/go.sum ./
# Add replace directives for Docker builds:
# 1. Platform → plugin (plugin source at /plugin/)
# 2. Plugin → platform (plugin's go.mod has a relative replace that doesn't
#    work in Docker; fix it to point at /app where the platform source lives)
RUN echo 'replace github.com/Molecule-AI/molecule-ai-plugin-github-app-auth => /plugin' >> go.mod
RUN sed -i 's|replace github.com/Molecule-AI/molecule-monorepo/platform => .*|replace github.com/Molecule-AI/molecule-monorepo/platform => /app|' /plugin/go.mod
RUN go mod download
COPY workspace-server/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /platform ./cmd/server

# Clone templates + plugins at build time from manifest.json
FROM alpine:3.20 AS templates
RUN apk add --no-cache git jq
COPY manifest.json /manifest.json
COPY scripts/clone-manifest.sh /scripts/clone-manifest.sh
RUN chmod +x /scripts/clone-manifest.sh && /scripts/clone-manifest.sh /manifest.json /workspace-configs-templates /org-templates /plugins

FROM alpine:3.20
RUN apk add --no-cache ca-certificates git tzdata
COPY --from=builder /platform /platform
COPY workspace-server/migrations /migrations
COPY --from=templates /workspace-configs-templates /workspace-configs-templates
COPY --from=templates /org-templates /org-templates
COPY --from=templates /plugins /plugins
EXPOSE 8080
CMD ["/platform"]
