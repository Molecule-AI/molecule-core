# Platform-only image (no canvas). Used by publish-platform-image workflow
# for GHCR + Fly registry. Tenant image uses Dockerfile.tenant instead.
#
# Build context: repo root.

FROM golang:1.25-alpine AS builder
WORKDIR /app
# Plugin source for replace directive in go.mod
COPY molecule-ai-plugin-github-app-auth/ /plugin/
COPY workspace-server/go.mod workspace-server/go.sum ./
# Add replace directive for Docker builds (plugin is COPYed to /plugin above)
RUN echo 'replace github.com/Molecule-AI/molecule-ai-plugin-github-app-auth => /plugin' >> go.mod
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
