# Liike App – Action Plan & Roadmap

Sovellus: **Liike App** – Liikunnan- ja treenintenseuraussovellus  
Teknologiat: **Go (Backend)**, **SQLite (Tietokanta)**, **React + TypeScript (Frontend - myöhemmin)**

---

## 🎯 Vahvistetut vaatimukset & Tekniset valinnat

1. **Käyttäjähallinta & Autentikaatio**: Rakennetaan alusta alkaen (rekisteröinti, kirjautuminen, salasanojen bcrypt-hajautus, JWT-autentikaatio).
2. **Go Backend -teknologiat**:
   - Reititys: Go standardikirjasto `net/http` (Go 1.22+ moderneilla reititysrajoitteilla).
   - Tietokanta-ajuri: Pure Go SQLite (`modernc.org/sqlite`) – ei vaadi CGO-kääntäjää.
3. **Lajit ja harjoitusmuodot**:
   - **Kävely & Juoksu**: Matka (km), kesto (min/s), keskisyke, kalorit.
   - **Jousiammunta**: Oma mukautettu tuloskirjausjärjestelmä (kierrokset, sarjat/amput, nuolet, osumapisteet / X-10).
   - **Kyykky, Vatsalihaslankku & Painot**: Sarjat, toistot, painot (kg) tai kestot (sekunnit).
   - **Jooga**: Kesto, fiilis/muistiinpanot.
4. **Dokumentaatiokäytännöt**:
   - Suunnitelmat tallennetaan hakemistoon `.plans/`.
   - PR Storyt tallennetaan hakemistoon `.pr_stories/` ja kirjoitetaan englanniksi.

---

## 🗺️ Vaihejako (Roadmap)

- [ ] **Vaihe 1: Projektirakenne & Go-backendin perustus** *(Meneillään)*
  - Go-moduulin alustus (`go mod init liike_app`).
  - Arkkitehtuurirakenne (`cmd/server`, `internal/config`, `internal/domain`, `internal/repository`, `internal/service`, `internal/handler`).
  - HTTP-palvelimen käynnistys ja terveystarkastus-endpoint (`GET /api/v1/health`).
- [ ] **Vaihe 2: Tietokanta-arkkitehtturi, Migraatiot & Käyttäjä/Laji-skeemat**
  - SQLite-yhteyspooli (`modernc.org/sqlite`).
  - Migraatiotyökalu ja tietokantataulut: `users`, `exercise_types`, `workouts`, `workout_sets`, `archery_scores`.
- [ ] **Vaihe 3: Käyttäjähallinta & JWT-Autentikaatio (API)**
  - `POST /api/v1/auth/register`, `POST /api/v1/auth/login`.
  - JWT Auth Middleware suojaamaan treenirajapintoja.
- [ ] **Vaihe 4: Suoritusten & Jousiammunnan REST API**
  - CRUD-rajapinnat suorituksille ja lajeille.
  - Jousiammunnan erikoistuloskirjauksen suorituslogiikka ja API.
- [ ] **Vaihe 5: Frontend Alustus (React + TypeScript + Vite)**
- [ ] **Vaihe 6: Frontend UI -Näkymät & Integraatio**
- [ ] **Vaihe 7: Viimeistely, E2E-testaus & PR Story**

---

## 📝 Vaihe 1: Ohjeistus käyttäjälle (Step-by-Step Instructions)

### 1.1 Projektihakemiston alustus
Aja terminaalissa projektin juurihakemistossa (`/home/vivaldev/code/liike_app`):

```bash
go mod init liike_app
```

Asenna tarvittavat riippuvuudet valmiiksi:
```bash
go get modernc.org/sqlite
go get golang.org/x/crypto/bcrypt
```

### 1.2 Hakemistorakenteen luominen
Luo seuraava Go-projektirakenne:

```
liike_app/
├── .plans/
│   └── action_plan.md
├── .pr_stories/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── domain/
│   ├── handler/
│   │   └── health.go
│   ├── repository/
│   └── service/
├── go.mod
└── go.sum
```

### 1.3 Tiedostojen toteutus

#### A. `internal/config/config.go`
```go
package config

import (
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "liike.db"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "super-secret-key-change-in-production"
	}

	return &Config{
		Port:        port,
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
	}
}
```

#### B. `internal/handler/health.go`
```go
package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    "OK",
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
```

#### C. `cmd/server/main.go`
```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"liike_app/internal/config"
	"liike_app/internal/handler"
)

func main() {
	cfg := config.Load()

	mux := http.NewServeMux()

	// Health check endpoint (Go 1.22+ routing syntax)
	mux.HandleFunc("GET /api/v1/health", handler.HealthCheckHandler)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Liike App backend starting at http://localhost%s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}
```

---

## 🧪 Vaiheen 1 Verifiointi

Käynnistä palvelin:
```bash
go run ./cmd/server/main.go
```

Testaa reitti:
```bash
curl http://localhost:8080/api/v1/health
```

Odotettu vastaus:
```json
{"status":"OK","timestamp":"2026-08-02T..."}
```
