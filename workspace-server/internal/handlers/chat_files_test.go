package handlers

// Unit tests for chat_files.go. The Docker-touching paths (Upload
// actually copying into a container, Download actually streaming tar)
// are exercised via integration tests — docker-in-docker is out of
// scope for the unit suite. These tests cover the validation + error
// surfaces that a caller can reach without a running container.

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"report.pdf", "report.pdf"},
		{"my file.pdf", "my_file.pdf"},
		{"../../etc/passwd", "passwd"},
		{"weird;$name`.txt", "weird__name_.txt"},
		{"", "file"},
		{".", "file"},
		{"..", "file"},
	}
	for _, tc := range cases {
		got := sanitizeFilename(tc.in)
		if got != tc.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSanitizeFilename_LongNamePreservesExtension(t *testing.T) {
	// 120-char base + .pdf — the helper should truncate the base but
	// keep the extension intact so content-type inference still works.
	longBase := strings.Repeat("a", 120)
	got := sanitizeFilename(longBase + ".pdf")
	if len(got) > 100 {
		t.Errorf("filename not truncated: len=%d", len(got))
	}
	if !strings.HasSuffix(got, ".pdf") {
		t.Errorf("extension stripped: %q", got)
	}
}

func TestChatUpload_InvalidWorkspaceID(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	tmplh := NewTemplatesHandler(t.TempDir(), nil)
	h := NewChatFilesHandler(tmplh)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "not-a-uuid"}}
	c.Request = httptest.NewRequest("POST", "/workspaces/not-a-uuid/chat/uploads", nil)

	h.Upload(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on invalid workspace id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChatUpload_MissingFiles(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	tmplh := NewTemplatesHandler(t.TempDir(), nil)
	h := NewChatFilesHandler(tmplh)

	// Multipart body with no `files` field — only a text field.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("other", "value")
	mw.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "00000000-0000-0000-0000-000000000001"}}
	req := httptest.NewRequest("POST", "/workspaces/00000000-0000-0000-0000-000000000001/chat/uploads", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	c.Request = req

	h.Upload(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when files field missing, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "files") {
		t.Errorf("expected error to mention files field: %s", w.Body.String())
	}
}

func TestChatDownload_InvalidPath(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	tmplh := NewTemplatesHandler(t.TempDir(), nil)
	h := NewChatFilesHandler(tmplh)

	cases := []struct {
		name, path, wantSubstr string
	}{
		{"empty", "", "path query required"},
		{"relative", "workspace/foo.txt", "must be absolute"},
		{"wrong root", "/etc/passwd", "must be under"},
		{"traversal", "/workspace/../etc/passwd", "invalid path"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: "00000000-0000-0000-0000-000000000001"}}
			req := httptest.NewRequest("GET", "/workspaces/xxx/chat/download?path="+tc.path, nil)
			c.Request = req

			h.Download(c)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for %s, got %d: %s", tc.name, w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tc.wantSubstr) {
				t.Errorf("expected error to contain %q, got: %s", tc.wantSubstr, w.Body.String())
			}
		})
	}
}

func TestContentDispositionAttachment_Escapes(t *testing.T) {
	cases := []struct {
		name, input, wantSubstr string
	}{
		{
			name:       "plain ASCII passes through",
			input:      "report.pdf",
			wantSubstr: `filename="report.pdf"`,
		},
		{
			name:       "double-quote is backslash-escaped",
			input:      `weird".pdf`,
			wantSubstr: `filename="weird\".pdf"`,
		},
		{
			name:       "CR and LF dropped to prevent header injection",
			input:      "bad\r\nX-Leak: 1\r\n.txt",
			wantSubstr: `filename="badX-Leak: 1.txt"`,
		},
		{
			name:       "non-ASCII emits filename* percent-encoded",
			input:      "résumé.pdf",
			wantSubstr: "filename*=UTF-8''r%C3%A9sum%C3%A9.pdf",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := contentDispositionAttachment(tc.input)
			if !strings.Contains(got, tc.wantSubstr) {
				t.Errorf("contentDispositionAttachment(%q) = %q, missing substring %q", tc.input, got, tc.wantSubstr)
			}
			// Must never contain a bare CR or LF — either would end the header.
			if strings.ContainsAny(got, "\r\n") {
				t.Errorf("header contains CR/LF: %q", got)
			}
		})
	}
}

func TestChatDownload_DockerUnavailable(t *testing.T) {
	setupTestDB(t)
	setupTestRedis(t)

	tmplh := NewTemplatesHandler(t.TempDir(), nil) // docker=nil
	h := NewChatFilesHandler(tmplh)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "00000000-0000-0000-0000-000000000001"}}
	req := httptest.NewRequest("GET", "/workspaces/xxx/chat/download?path=/workspace/report.pdf", nil)
	c.Request = req

	h.Download(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when docker is nil, got %d: %s", w.Code, w.Body.String())
	}
}
