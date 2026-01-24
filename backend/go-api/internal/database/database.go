// Package database provides functionality for database connection, management,
// and query execution.
package database

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang-migrate/migrate/v4"
	// Driver for database migrations from file source.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// Driver for file-based migrations.
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	// PostgreSQL driver.
	_ "github.com/lib/pq"
)

// DB is a wrapper around the sqlx.DB struct to allow for extension
// with custom methods and properties, like a performance cache.
type DB struct {
	*sqlx.DB
	// columnCache caches the results of column existence checks to prevent
	// redundant queries to the information_schema, improving performance
	// for features like dynamic pgvector support.
	columnCache      map[string]bool
	columnCacheMutex sync.RWMutex
}

// New establishes a connection to the PostgreSQL database using the provided URL,
// configures the connection pool, pings the database, and initializes the DB struct.
func New(dbURL string) (*DB, error) {
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL environment variable is not set")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}

	// TODO: Move these values to the configuration struct.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping the database: %w", err)
	}

	log.Println("Successfully connected to the PostgreSQL database.")

	return &DB{
		DB:          db,
		columnCache: make(map[string]bool), // Initialize the cache map.
	}, nil
}

// Migrate applies all available database migrations found in the specified path.
// It will not return an error if the database is already up to date.
func (db *DB) Migrate(databaseURL, migrationsPath string) error {
	// The source URL must be prefixed with 'file://'.
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)

	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// m.Up() applies all available 'up' migrations.
	// migrate.ErrNoChange is not a critical error; it means the DB is already up-to-date.
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		log.Printf("Could not get migration version, but migrations were likely applied: %v", err)
	}

	if dirty {
		log.Printf("Database is at migration version %d, but is marked as dirty.", version)
		return fmt.Errorf("database is in a dirty migration state")
	}

	if errors.Is(err, migrate.ErrNilVersion) {
		log.Println("Database migrations applied successfully, but no version tag was found.")
	} else {
		log.Printf("Database migrations are up-to-date at version %d.", version)
	}

	return nil
}
