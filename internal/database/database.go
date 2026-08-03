package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// New avaa SQLite-tietokantayhteyden ja asettaa suorituskyky-PRAGMAt
func New(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("virhe tietokannan avaamisessa: %w", err)
	}

	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	}

	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("virhe PRAGMA-asetuksessa (%s): %w", p, err)
		}
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("virhe tietokantayhteydessä (ping): %w", err)
	}

	return db, nil
}
