FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /platform ./cmd/server

FROM alpine:3.20
# git is required by the `github` SourceResolver for plugin installs from
# GitHub URLs (POST /workspaces/:id/plugins {"source": "github://..."}).
RUN apk add --no-cache ca-certificates git tzdata
COPY --from=builder /platform /platform
COPY migrations /migrations
EXPOSE 8080
CMD ["/platform"]
