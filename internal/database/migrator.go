package database

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

// RunMigrations ajaa kaikki annetut .up.sql migraatiot järjestyksessä
func RunMigrations(db *sql.DB, fs embed.FS, dir string) error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createTableQuery); err != nil {
		return fmt.Errorf("virhe schema_migrations-taulun luonnissa: %w", err)
	}

	entries, err := fs.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("virhe migraatiohakemiston luvussa: %w", err)
	}

	if err != nil {
	}
	var upFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			upFiles = append(upFiles, entry.Name())
		}
	}
	sort.Strings(upFiles)

	for _, filename := range upFiles {
		var version int
		_, err := fmt.Sscanf(filename, "%d_", &version)
		if err != nil {
			return fmt.Errorf("virheellinen migraatiotiedoston nimi %s: %w", filename, err)
		}

		var count int
		err = db.QueryRow("SELECT COUNT(1) FROM schema_migrations WHERE version = ?", version).Scan(&count)
		if err != nil {
			return fmt.Errorf("virhe migraatioversion tarkistuksessa: %w", err)
		}

		if count > 0 {
			continue // Migraatio jo suoritettu
		}

		content, err := fs.ReadFile(dir + "/" + filename)
		if err != nil {
			return fmt.Errorf("virhe migraatiotiedoston %s luvussa: %w", filename, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("virhe transaktion aloituksessa (%s): %w", filename, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("virhe migraation %s suorituksessa: %w", filename, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("virhe migraationversion tallennuksessa: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("virhe transaktion vahvistuksessa: %w", err)
		}
	}

	return nil
}
