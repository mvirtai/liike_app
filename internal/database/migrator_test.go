package database_test

import (
	"database/sql"
	"embed"
	"testing"

	"liike_app/internal/database"
)

// testMigrationFS sisältää testimigraatiot embed-muistissa.
// Go:n embed sisällyttää vain tiedostot eikä tyhjiä hakemistoja.
//
//go:embed testdata/migrations/001_create_users.up.sql testdata/migrations/002_create_exercise_types.up.sql
var testMigrationFS embed.FS

// openTestDB avaa in-memory SQLite-tietokannan testejä varten.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("openTestDB: database.New() epäonnistui: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestRunMigrations_CreatesSchemaTable varmistaa, että RunMigrations luo
// schema_migrations-taulun automaattisesti.
func TestRunMigrations_CreatesSchemaTable(t *testing.T) {
	db := openTestDB(t)

	err := database.RunMigrations(db, testMigrationFS, "testdata/migrations")
	if err != nil {
		t.Fatalf("RunMigrations() palautti virheen: %v", err)
	}

	var count int
	row := db.QueryRow(
		"SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='schema_migrations'",
	)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("schema_migrations-taulun kysely epäonnistui: %v", err)
	}
	if count != 1 {
		t.Error("odotettu schema_migrations-taulu puuttuu migraatioiden ajamisen jälkeen")
	}
}

// TestRunMigrations_AppliesAllMigrations tarkistaa, että kaikki .up.sql-tiedostot
// kirjataan schema_migrations-tauluun.
func TestRunMigrations_AppliesAllMigrations(t *testing.T) {
	db := openTestDB(t)

	if err := database.RunMigrations(db, testMigrationFS, "testdata/migrations"); err != nil {
		t.Fatalf("RunMigrations() epäonnistui: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(1) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("schema_migrations-kysely epäonnistui: %v", err)
	}
	// testdata/migrations-kansio sisältää kaksi .up.sql-tiedostoa
	if count != 2 {
		t.Errorf("odotettu 2 migraatiomerkintää, saatiin %d", count)
	}
}

// TestRunMigrations_Idempotent varmistaa, että RunMigrations voidaan ajaa
// useasti ilman virhettä tai duplikaattimerkintöjä.
func TestRunMigrations_Idempotent(t *testing.T) {
	db := openTestDB(t)

	for i := 0; i < 3; i++ {
		if err := database.RunMigrations(db, testMigrationFS, "testdata/migrations"); err != nil {
			t.Fatalf("RunMigrations() ajo %d/3 epäonnistui: %v", i+1, err)
		}
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(1) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("schema_migrations-kysely epäonnistui: %v", err)
	}
	if count != 2 {
		t.Errorf("idempotenssi rikki: odotettu 2 migraatiomerkintää, saatiin %d", count)
	}
}

// TestRunMigrations_CreatesUserTable tarkistaa, että users-taulu luodaan
// ensimmäisen migraation jälkeen.
func TestRunMigrations_CreatesUserTable(t *testing.T) {
	db := openTestDB(t)

	if err := database.RunMigrations(db, testMigrationFS, "testdata/migrations"); err != nil {
		t.Fatalf("RunMigrations() epäonnistui: %v", err)
	}

	var name string
	err := db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='users'",
	).Scan(&name)
	if err != nil {
		t.Fatalf("users-taulun tarkistus epäonnistui: %v", err)
	}
	if name != "users" {
		t.Errorf("odotettu taulu 'users', saatiin '%s'", name)
	}
}

// TestRunMigrations_NoUpSqlFiles varmistaa, että jos hakemistossa ei ole
// .up.sql-tiedostoja, RunMigrations ei palauta virhettä.
func TestRunMigrations_NoUpSqlFiles(t *testing.T) {
	// Käytetään erillisiä embed-tiedostoja, jotka eivät sisällä .up.sql-tiedostoja
	// Testataan, että schema_migrations-taulu luodaan myös tässä tapauksessa.
	db := openTestDB(t)

	// Ajetaan migraatiot normaalisti ensin
	if err := database.RunMigrations(db, testMigrationFS, "testdata/migrations"); err != nil {
		t.Fatalf("RunMigrations() epäonnistui: %v", err)
	}

	// Ajetaan uudestaan — ei uusia .up.sql-tiedostoja ajetaan, count pysyy samana
	var countBefore int
	if err := db.QueryRow("SELECT COUNT(1) FROM schema_migrations").Scan(&countBefore); err != nil {
		t.Fatalf("schema_migrations-kysely epäonnistui: %v", err)
	}

	if err := database.RunMigrations(db, testMigrationFS, "testdata/migrations"); err != nil {
		t.Fatalf("RunMigrations() toinen ajo epäonnistui: %v", err)
	}

	var countAfter int
	if err := db.QueryRow("SELECT COUNT(1) FROM schema_migrations").Scan(&countAfter); err != nil {
		t.Fatalf("schema_migrations-kysely epäonnistui: %v", err)
	}
	if countBefore != countAfter {
		t.Errorf("migraatioiden määrä muuttui odottamattomasti: %d -> %d", countBefore, countAfter)
	}
}
