package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

type Migration struct {
	Version int
	Name    string
	SQL     string
}

type Migrator struct {
	db            *sql.DB
	migrationsDir string
}

func NewMigrator(db *sql.DB, migrationsDir string) *Migrator {
	return &Migrator{
		db:            db,
		migrationsDir: migrationsDir,
	}
}

func (m *Migrator) ensureMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			name VARCHAR(255) NOT NULL
		);
	`
	_, err := m.db.Exec(query)
	return err
}

func (m *Migrator) getAppliedMigrations() (map[int]bool, error) {
	applied := make(map[int]bool)
	
	rows, err := m.db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return applied, err
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return applied, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

func (m *Migrator) loadMigrations() ([]Migration, error) {
	var migrations []Migration

	files, err := filepath.Glob(filepath.Join(m.migrationsDir, "*.sql"))
	if err != nil {
		return migrations, err
	}

	for _, file := range files {
		filename := filepath.Base(file)
		
		// Skip down migrations for now
		if strings.Contains(filename, ".down.sql") {
			continue
		}
		
		// Parse version from filename (format: 001_migration_name.sql)
		parts := strings.Split(filename, "_")
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			log.Printf("Skipping migration file %s: invalid version number", filename)
			continue
		}

		name := strings.TrimSuffix(strings.Join(parts[1:], "_"), ".sql")

		content, err := os.ReadFile(file)
		if err != nil {
			return migrations, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func (m *Migrator) Up() error {
	if err := m.ensureMigrationsTable(); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	applied, err := m.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	for _, migration := range migrations {
		if applied[migration.Version] {
			log.Printf("Migration %d (%s) already applied, skipping", migration.Version, migration.Name)
			continue
		}

		log.Printf("Applying migration %d: %s", migration.Version, migration.Name)
		
		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", migration.Version, err)
		}

		if _, err := tx.Exec(migration.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %d: %w", migration.Version, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", 
			migration.Version, migration.Name); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		log.Printf("Successfully applied migration %d: %s", migration.Version, migration.Name)
	}

	return nil
}

func (m *Migrator) Status() error {
	if err := m.ensureMigrationsTable(); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	applied, err := m.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	log.Println("Migration Status:")
	log.Println("================")

	for _, migration := range migrations {
		status := "PENDING"
		if applied[migration.Version] {
			status = "APPLIED"
		}
		log.Printf("[%s] %03d: %s", status, migration.Version, migration.Name)
	}

	return nil
}