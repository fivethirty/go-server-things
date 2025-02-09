package sqlite3

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/fivethirty/go-server-things/logs"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	driver "github.com/mattn/go-sqlite3"
)

const (
	InMemory = ":memory:"
	Driver   = "sqlite3"
)

type Config struct {
	Dir        string
	DB         string
	Options    string
	Migrations fs.FS
	Upload     func(context.Context, *os.File) error
}

func (c *Config) Connection() string {
	if c.Dir == InMemory {
		return c.Dir
	}
	return fmt.Sprintf("%s%s?%s", c.Dir, c.DB, c.Options)
}

type SQLite3 struct {
	DB     *sqlx.DB
	config *Config
}

var logger *slog.Logger = logs.Default

func New(ctx context.Context, config Config) (*SQLite3, error) {
	conn := config.Connection()
	logger.Info(
		"Connecting to SQLite",
		"db", conn,
	)

	if conn != InMemory {
		if err := os.MkdirAll(config.Dir, os.ModePerm); err != nil {
			return nil, err
		}
	}

	db, err := sqlx.Open(Driver, conn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return &SQLite3{
		DB:     db,
		config: &config,
	}, nil
}

type migrateLogger struct {
	logger *slog.Logger
}

func (ml *migrateLogger) Verbose() bool {
	return false
}

func (ml *migrateLogger) Printf(format string, v ...any) {
	ml.logger.Info(fmt.Sprintf(format, v...))
}

func (s *SQLite3) Migrate() error {
	driver, err := sqlite3.WithInstance(s.DB.DB, &sqlite3.Config{})
	if err != nil {
		return err
	}

	dir, err := iofs.New(s.config.Migrations, "testmigrations")
	if err != nil {
		log.Fatal(err)
	}

	m, err := migrate.NewWithInstance("iofs", dir, "sqlite3", driver)

	if err != nil {
		return err
	}

	m.Log = &migrateLogger{
		logger: logger,
	}

	err = m.Up()
	if err == migrate.ErrNoChange {
		logger.Info("No new migrations.")
	} else if err != nil {
		return err
	}

	return nil
}

func (s *SQLite3) Close() error {
	return s.DB.Close()
}

func (s *SQLite3) Backup(ctx context.Context, dir string, name string) error {
	file, err := s.copy(ctx, dir, name)
	if err != nil {
		return fmt.Errorf("Backup: %w", err)
	}

	if s.config.Upload == nil {
		logger.Info("No upload function configured, skipping remote database backup.")
		return nil
	}

	err = s.config.Upload(ctx, file)
	if err != nil {
		return fmt.Errorf("Backup: %w", err)
	}
	return nil
}

func (s *SQLite3) copy(ctx context.Context, dir string, name string) (*os.File, error) {
	connStr := fmt.Sprintf(
		"%s%s",
		dir,
		name,
	)
	logger.Info(
		"Copying SQLite",
		"from", strings.Split(s.config.Connection(), "?")[0],
		"to", connStr,
	)
	copy, err := sqlx.Open(Driver, connStr)
	if err != nil {
		return nil, err
	}
	defer copy.Close()

	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	copyConn, err := copy.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer copyConn.Close()
	return s.doCopy(conn, copyConn)
}

func (*SQLite3) doCopy(conn, copyConn *sql.Conn) (*os.File, error) {
	var file *os.File
	return file, conn.Raw(func(rawConn any) error {
		return copyConn.Raw(func(rawCopyConn any) error {
			sqliteConn, ok := rawConn.(*driver.SQLiteConn)
			if !ok {
				return fmt.Errorf("error when casting source raw connection to sqlite connection")
			}

			copySQLiteConn, ok := rawCopyConn.(*driver.SQLiteConn)
			if !ok {
				return fmt.Errorf("error when casting source raw connection to sqlite connection")
			}

			backup, err := copySQLiteConn.Backup("main", sqliteConn, "main")
			if err != nil {
				return err
			}

			if _, err := backup.Step(-1); err != nil {
				return err
			}

			if err := backup.Finish(); err != nil {
				return err
			}

			filename := copySQLiteConn.GetFilename("")

			logger.Info(
				"SQLite dump complete.",
				"to", filename,
			)

			file, err = os.Open(filename)
			return err
		})
	})
}
