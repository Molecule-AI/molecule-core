package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitPostgres(databaseURL string) error {
	var err error
	DB, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	log.Println("Connected to Postgres")
	return nil
}

func RunMigrations(migrationsDir string) error {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)

	for _, f := range files {
		log.Printf("Applying migration: %s", filepath.Base(f))
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if _, err := DB.Exec(string(content)); err != nil {
			return fmt.Errorf("exec %s: %w", filepath.Base(f), err)
		}
	}
	log.Printf("Applied %d migrations", len(files))
	return nil
}
