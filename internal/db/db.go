package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

var (
	DB      *sql.DB
	DataDir string
)

// InitDB initializes SQLite database and runs all pending migrations
func InitDB(customDataDir string) (*sql.DB, error) {
	if customDataDir != "" {
		DataDir = customDataDir
	} else if envDir := os.Getenv("DATA_DIR"); envDir != "" {
		DataDir = envDir
	} else {
		DataDir = "./data"
	}

	// Ensure required directories exist inside DataDir
	subdirs := []string{"", "logos", "epg", "streams"}
	for _, sub := range subdirs {
		dirPath := filepath.Join(DataDir, sub)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
	}

	dbPath := filepath.Join(DataDir, "data.db")
	logrus.Infof("Opening SQLite database at: %s", dbPath)

	// SQLite connection string with WAL and busy timeout
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", filepath.ToSlash(dbPath))
	var err error
	DB, err = sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Verify connectivity
	if err := DB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	// Run embedded migrations
	if err := runMigrations(DB); err != nil {
		return nil, fmt.Errorf("failed to execute migrations: %w", err)
	}

	logrus.Info("Database initialized and migrations applied successfully")
	return DB, nil
}

func runMigrations(db *sql.DB) error {
	// Ensure migration table exists
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("failed to initialize schema_migrations: %w", err)
	}

	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read embedded migrations: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		var version int
		_, err := fmt.Sscanf(entry.Name(), "%d_", &version)
		if err != nil {
			logrus.Warnf("Skipping invalid migration filename %s: %v", entry.Name(), err)
			continue
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", entry.Name(), err)
		}

		if count > 0 {
			continue // Already applied
		}

		content, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", entry.Name(), err)
		}

		logrus.Infof("Applying migration: %s", entry.Name())
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", entry.Name(), err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %s: %w", entry.Name(), err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version, name) VALUES (?, ?)", version, entry.Name()); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", entry.Name(), err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// GetDataDir returns root storage directory
func GetDataDir() string {
	if DataDir == "" {
		return "./data"
	}
	return DataDir
}

// GetLogosDir returns logos folder path
func GetLogosDir() string {
	return filepath.Join(GetDataDir(), "logos")
}

// GetEPGDir returns epg cache folder path
func GetEPGDir() string {
	return filepath.Join(GetDataDir(), "epg")
}

// GetStreamsDir returns streams folder path
func GetStreamsDir() string {
	return filepath.Join(GetDataDir(), "streams")
}
