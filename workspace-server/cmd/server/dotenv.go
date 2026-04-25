package main

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// loadDotEnvIfPresent walks upward from CWD looking for a .env file and
// merges its KEY=VALUE pairs into the process environment. Already-set
// vars (e.g. from `docker run -e`, CI exports, or ad-hoc `KEY=val
// ./binary`) win over file values so operators can override without
// editing the file.
//
// Why walk upward: the binary may be launched from the monorepo root,
// the workspace-server subdir, or anywhere else the operator finds
// convenient. Walking upward from CWD finds the canonical .env
// (gitignored, lives at the monorepo root) regardless of cwd, so a
// fresh `go build -o /tmp/molecule-server ./cmd/server && /tmp/molecule-server`
// from any subdir picks up the same MOLECULE_ENV / DATABASE_URL / etc.
// the operator already has — without sourcing or `set -a`.
//
// Why no godotenv dep: the format we use is simple — KEY=VALUE with
// optional `#` comments and no interpolation — so a tiny in-tree parser
// is auditable, has no supply-chain surface, and avoids drift across
// repos where some teams configure godotenv differently.
//
// Why it's safe in production: the Dockerfile does not COPY .env into
// the image and `.env` is gitignored, so production containers have no
// .env on disk to load. If an operator goes out of their way to put one
// there, the explicit-env-wins rule above means container env still
// dominates.
func loadDotEnvIfPresent() {
	path, ok := findDotEnv()
	if !ok {
		return
	}
	f, err := os.Open(path)
	if err != nil {
		log.Printf(".env: open %s: %v (skipping)", path, err)
		return
	}
	defer f.Close()

	loaded := 0
	skipped := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		k, v, ok := parseDotEnvLine(scanner.Text())
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(k); exists {
			skipped++
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			log.Printf(".env: set %s: %v", k, err)
			continue
		}
		loaded++
	}
	if err := scanner.Err(); err != nil {
		log.Printf(".env: scan %s: %v", path, err)
	}
	log.Printf(".env: %s — loaded %d, %d already set in env", path, loaded, skipped)
}

// findDotEnv returns the path of the nearest .env file walking upward
// from CWD. Capped at 6 levels so a deeply-nested launch dir doesn't
// scan the entire filesystem.
func findDotEnv() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for i := 0; i < 6; i++ {
		p := filepath.Join(dir, ".env")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

// parseDotEnvLine parses a single .env line. Returns (key, value, true)
// for KEY=VALUE pairs. Returns (_, _, false) for blanks, comments, and
// malformed lines. Supports inline `# comment` after a value when
// preceded by whitespace, matching the convention already in the
// repo's .env file.
func parseDotEnvLine(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	eq := strings.IndexByte(line, '=')
	if eq <= 0 {
		return "", "", false
	}
	k := strings.TrimSpace(line[:eq])
	v := line[eq+1:]
	// Strip inline comment introduced by whitespace + `#`. A bare `#`
	// inside the value (no preceding whitespace) is part of the value
	// — matches the convention in dotenv parsers and lets values like
	// `KEY=token#fragment` round-trip.
	for _, sep := range []string{" #", "\t#"} {
		if i := strings.Index(v, sep); i >= 0 {
			v = v[:i]
			break
		}
	}
	return k, strings.TrimSpace(v), true
}
