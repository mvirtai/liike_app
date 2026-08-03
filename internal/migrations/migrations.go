// Package migrations sisältää SQL-migraatiotiedostot embed.FS:nä.
// Tämä paketti on ainoa paikka, josta migraatiot embedataan,
// koska Go:n //go:embed vaatii tiedostot samassa tai alihakemistossa.
package migrations

import "embed"

// FS on kaikki SQL-migraatiotiedostot embed.FS-muodossa.
//
//go:embed sql/*.sql
var FS embed.FS

// Dir on hakemiston nimi, josta migraatiot löytyvät FS:stä.
const Dir = "sql"
