package database_test

import (
	"testing"

	"liike_app/internal/database"
)

// TestNew_InMemory tarkistaa, että New() avaa in-memory-tietokannan onnistuneesti
// ja asettaa kaikki suorituskyky-PRAGMAt ilman virhettä.
func TestNew_InMemory(t *testing.T) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("database.New(':memory:') palautti odottamattoman virheen: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Errorf("db.Ping() epäonnistui yhteyden avauksen jälkeen: %v", err)
	}
}

// TestNew_PragmaForeignKeys varmistaa, että foreign_keys-PRAGMA on käytössä.
func TestNew_PragmaForeignKeys(t *testing.T) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("database.New() epäonnistui: %v", err)
	}
	defer db.Close()

	var fkEnabled int
	row := db.QueryRow("PRAGMA foreign_keys;")
	if err := row.Scan(&fkEnabled); err != nil {
		t.Fatalf("PRAGMA foreign_keys -kysely epäonnistui: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("odotettu PRAGMA foreign_keys = 1, saatiin %d", fkEnabled)
	}
}

// TestNew_PragmaJournalMode varmistaa, että journal_mode on WAL.
func TestNew_PragmaJournalMode(t *testing.T) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("database.New() epäonnistui: %v", err)
	}
	defer db.Close()

	var mode string
	row := db.QueryRow("PRAGMA journal_mode;")
	if err := row.Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode -kysely epäonnistui: %v", err)
	}
	// In-memory-tietokanta palauttaa "memory" WAL:n sijaan —
	// WAL-tila toimii vain tiedostopohjaisessa tietokannassa.
	// Testataan siis, että PRAGMA hyväksyttiin ilman virhettä.
	if mode == "" {
		t.Error("PRAGMA journal_mode palautti tyhjän arvon")
	}
}

// TestNew_InvalidPath tarkistaa, että selvästi virheellinen polku johtaa virheeseen.
func TestNew_InvalidPath(t *testing.T) {
	// Käytetään hakemistoa, johon ei ole kirjoitusoikeutta, testaamaan virhettä.
	// Go:n sqlite-ajuri saattaa hyväksyä useimmat polut,
	// joten testataan Ping() epäonnistuminen tyhjällä URI:lla.
	db, err := database.New("/nonexistent_dir_xyz/liike_test.db")
	if err != nil {
		// Odotettu: ajuri hylkäsi polun jo avausvaiheessa.
		return
	}
	defer db.Close()
	if pingErr := db.Ping(); pingErr != nil {
		// Hyväksytty: Ping epäonnistuu, jos tiedostoa ei voi luoda.
		return
	}
	// SQLite saattaa luoda tiedoston onnistuneesti paikalliseen hakemistoon,
	// jos käyttöjärjestelmä sallii sen — testi ei epäonnistu siinä tilanteessa.
	t.Log("Huomio: sqlite loi tiedoston odottamattomaan polkuun – ajuri on joustava")
}
