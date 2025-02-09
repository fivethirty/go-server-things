package sqlite3_test

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/fivethirty/go-server-things/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

//go:embed testmigrations/*.sql
var migrations embed.FS

func TestConnecton(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dir      string
		db       string
		options  string
		expected string
	}{
		{
			name:     "in memory",
			dir:      sqlite3.InMemory,
			db:       "ignored.db",
			options:  "ignored=true",
			expected: sqlite3.InMemory,
		},
		{
			name:     "filesystem no options",
			dir:      "/foo/bar/",
			db:       "test.db",
			expected: "/foo/bar/test.db?",
		},
		{
			name:     "filesystem no options",
			dir:      "/foo/bar/",
			db:       "test.db",
			options:  "cache=shared",
			expected: "/foo/bar/test.db?cache=shared",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := sqlite3.Config{
				Dir:     tt.dir,
				DB:      tt.db,
				Options: tt.options,
			}

			if config.Connection() != tt.expected {
				t.Fatalf("Expected %s but got %s", tt.expected, config.Connection())
			}
		})
	}
}

func TestSQLite3(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dir      string
		db       string
		options  string
		doUpload bool
	}{
		{
			name:     "should create and backup in memory database",
			dir:      sqlite3.InMemory,
			db:       "test.db",
			doUpload: false,
		},
		{
			name:     "should create and backup filesystem database",
			dir:      fmt.Sprintf("/tmp/%s/", uuid.New()),
			db:       "test.db",
			doUpload: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			t.Cleanup(func() {
				os.RemoveAll(tt.dir)
			})
			ctx := context.Background()

			dir, err := iofs.New(migrations, "testmigrations")
			if err != nil {
				t.Fatal(err)
			}

			config := sqlite3.Config{
				Dir:        tt.dir,
				DB:         tt.db,
				Options:    tt.options,
				Migrations: dir,
			}

			db, err := sqlite3.New(ctx, config)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				db.Close()
			})

			testMigrations(t, ctx, db)
			testBackup(t, ctx, db)
		})
	}
}

func testMigrations(t *testing.T, ctx context.Context, s *sqlite3.SQLite3) {
	t.Helper()

	if err := s.Migrate(); err != nil {
		t.Fatal(err)
	}

	testRow(t, ctx, s.DB)
}

func testBackup(t *testing.T, ctx context.Context, s *sqlite3.SQLite3) {
	t.Helper()
	dir := fmt.Sprintf("/tmp/%s/", uuid.New())
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	copied, err := s.Copy(ctx, dir, "backup.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		copied.Close()
	})

	path, err := filepath.Abs(copied.Name())
	if err != nil {
		t.Fatal(err)
	}

	copiedDB, err := sqlx.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		copiedDB.Close()
	})

	testRow(t, ctx, copiedDB)
}

func testRow(t *testing.T, ctx context.Context, db *sqlx.DB) {
	t.Helper()

	var text string
	err := db.GetContext(ctx, &text, "SELECT text FROM test WHERE id = 1;")
	if err != nil {
		t.Fatal(err)
	}

	if text != "hello world" {
		t.Fatalf("Expected hello world but got %s", text)
	}
}
