# Vaihe 1: Projektirakenne ja Go-backendin perustus – Yksityiskohtainen suunnitelma

**Sovellus:** Liike App  
**Moduuli:** Backend (Go)  
**Dokumentti:** `.plans/phase_1_project_setup.md`  

---

## 📌 1. Tavoite ja Yhteenveto

Vaiheessa 1 luodaan **Liike App** -sovelluksen Go-backendin perusta. Tavoitteena on pystyttää selkeä, kerrosarkkitehtuuria noudattava projektirakenne, ympäristömuuttujien konfiguraationhallinta sekä käynnistyvä HTTP-palvelin, joka tarjoaa JSON-muotoisen terveystarkastusrajapinnan (`GET /api/v1/health`).

Lisäksi palvelimelle toteutetaan siisti hallittu sammutus (*graceful shutdown*), jotta tulevat tietokantayhteydet ja avoimet pyynnöt saadaan suljettua turvallisesti.

---

## 🛠️ 2. Esivaatimukset

- **Go 1.22** tai uudempi asennettuna (`go version`).
- Hakemisto `/home/vivaldev/code/liike_app` toimii projektin juurena.

---

## 📂 3. Kohde-hakemistorakenne

Vaiheen 1 päätteeksi projektin tiedostorakenne on seuraava:

```
liike_app/
├── .plans/
│   ├── action_plan.md
│   └── phase_1_project_setup.md
├── .pr_stories/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── domain/       (valmius tulevia malleja varten)
│   ├── handler/
│   │   └── health.go
│   ├── repository/   (valmius tietokantakerrokselle)
│   └── service/      (valmius liiketoimintalogiikalle)
├── go.mod
└── go.sum
```

---

## 📋 4. Yksityiskohtainen toteutusohje koodaajalle (Step-by-Step)

### Vaihe 1.1: Go-moduulin alustus ja riippuvuudet

Aja seuraavat komennot projektin juurihakemistossa:

```bash
# 1. Alusta Go-moduuli nimellä 'liike_app'
go mod init liike_app

# 2. Hae tulevat perusriippuvuudet (Pure Go SQLite + bcrypt salasanoille)
go get modernc.org/sqlite
go get golang.org/x/crypto/bcrypt
```

---

### Vaihe 1.2: Hakemistojen luonti

Luo tarvittava kansiorakenne:

```bash
mkdir -p cmd/server internal/config internal/domain internal/handler internal/repository internal/service .pr_stories
```

---

### Vaihe 1.3: Konfiguraationhallinta (`internal/config/config.go`)

Luo tiedosto `internal/config/config.go` ja sijoita siihen seuraava koodi:

```go
package config

import (
 "log"
 "os"
)

// Config sisältää sovelluksen ajonaikaiset asetukset
type Config struct {
 Port        string
 DatabaseURL string
 JWTSecret   string
 Environment string
}

// Load lukee ympäristömuuttujat ja asettaa oletusarvot
func Load() *Config {
 port := getEnv("PORT", "8080")
 dbURL := getEnv("DATABASE_URL", "liike.db")
 jwtSecret := getEnv("JWT_SECRET", "dev-secret-key-change-in-production-12345")
 env := getEnv("APP_ENV", "development")

 cfg := &Config{
  Port:        port,
  DatabaseURL: dbURL,
  JWTSecret:   jwtSecret,
  Environment: env,
 }

 log.Printf("[CONFIG] Asetukset ladattu. Ympäristö: %s, Portti: %s", cfg.Environment, cfg.Port)
 return cfg
}

func getEnv(key, fallback string) string {
 if value, exists := os.LookupEnv(key); exists && value != "" {
  return value
 }
 return fallback
}
```

---

### Vaihe 1.4: Terveystarkastus-handler (`internal/handler/health.go`)

Luo tiedosto `internal/handler/health.go` ja sijoita siihen seuraava koodi:

```go
package handler

import (
 "encoding/json"
 "net/http"
 "time"
)

// HealthResponse määrittelee terveystarkastusvastauksen rakenteen
type HealthResponse struct {
 Status    string    `json:"status"`
 Timestamp time.Time `json:"timestamp"`
 Service   string    `json:"service"`
 Version   string    `json:"version"`
}

// HealthCheckHandler käsittelee GET /api/v1/health pyynnöt
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
 resp := HealthResponse{
  Status:    "OK",
  Timestamp: time.Now().UTC(),
  Service:   "Liike App API",
  Version:   "1.0.0",
 }

 w.Header().Set("Content-Type", "application/json")
 w.WriteHeader(http.StatusOK)
 if err := json.NewEncoder(w).Encode(resp); err != nil {
  http.Error(w, "Virhe vastauksen muodostamisessa", http.StatusInternalServerError)
 }
}
```

---

### Vaihe 1.5: Pääohjelma & HTTP-palvelin Graceful Shutdownilla (`cmd/server/main.go`)

Luo tiedosto `cmd/server/main.go` ja sijoita siihen seuraava koodi:

```go
package main

import (
 "context"
 "errors"
 "fmt"
 "log"
 "net/http"
 "os"
 "os/signal"
 "syscall"
 "time"

 "liike_app/internal/config"
 "liike_app/internal/handler"
)

func main() {
 // 1. Lataa konfiguraatio
 cfg := config.Load()

 // 2. Alusta HTTP ServeMux (Go 1.22+ reititys)
 mux := http.NewServeMux()

 // 3. Rekisteröi reitit
 mux.HandleFunc("GET /api/v1/health", handler.HealthCheckHandler)

 // 4. Määritä HTTP-palvelimen asetukset ja aikakatkaisut
 server := &http.Server{
  Addr:         fmt.Sprintf(":%s", cfg.Port),
  Handler:      mux,
  ReadTimeout:  10 * time.Second,
  WriteTimeout: 10 * time.Second,
  IdleTimeout:  60 * time.Second,
 }

 // 5. Käynnistä palvelin omassa goroutiinissaan
 go func() {
  log.Printf("[SERVER] Liike App backend käynnissä osoitteessa http://localhost%s", server.Addr)
  if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
   log.Fatalf("[SERVER] Kriittinen virhe palvelinta käynnistettäessä: %v", err)
  }
 }()

 // 6. Hallittu sammutus (Graceful Shutdown)
 stop := make(chan os.Signal, 1)
 signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

 // Odotetaan sammutussignaalia (Ctrl+C tai SIGTERM)
 <-stop
 log.Println("[SERVER] Sammutussignaali vastaanotettu. Suljetaan palvelin...")

 ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
 defer cancel()

 if err := server.Shutdown(ctx); err != nil {
  log.Fatalf("[SERVER] Virhe palvelimen hallitussa sammutuksessa: %v", err)
 }

 log.Println("[SERVER] Palvelin suljettu onnistuneesti.")
}
```

---

## 🧪 5. Verifiointi ja Testaus

Kun olet luonut tiedostot, suorita seuraavat tarkastukset:

### 1. Käännös ja käynnistys

```bash
go run ./cmd/server/main.go
```

*Varmista, että lokiin tulostuu asetusten lataus ja tieto palvelimen käynnistymisestä osoitteessa `http://localhost:8080`.*

### 2. HTTP API -testaus (toisessa terminaalissa)

```bash
curl -i http://localhost:8080/api/v1/health
```

**Odotettu tulos (HTTP 200 OK):**

```http
HTTP/1.1 200 OK
Content-Type: application/json
Date: Sun, 02 Aug 2026 ...
Content-Length: 95

{"status":"OK","timestamp":"2026-08-02T11:36:00Z","service":"Liike App API","version":"1.0.0"}
```

### 3. Graceful Shutdown -testaus

Paina palvelimen terminaalissa `Ctrl + C`.  
*Varmista, että lokiin tulostuu: `[SERVER] Sammutussignaali vastaanotettu...` ja `[SERVER] Palvelin suljettu onnistuneesti.`*

---

## ✅ 6. Hyväksyntäkriteerit (Acceptance Criteria)

- [ ] `go.mod` ja `go.sum` on luotu projektin juureen.
- [ ] Hakemistorakenne (`cmd/server`, `internal/config`, `internal/handler` jne.) on luotu.
- [ ] Palvelin käynnistyy ilman virheitä komennolla `go run ./cmd/server/main.go`.
- [ ] `GET /api/v1/health` palauttaa statuskoodin 200 ja valitun JSON-rakenteen.
- [ ] Palvelin sulkeutuu hallitusti `Ctrl+C` -näppäinyhdistelmällä.

---

## 📄 7. Vaiheen 1 PR Story (Luodaan suorituksen jälkeen)

Kun olet suorittanut vaiheen 1 ja todennut sen toimivaksi, ilmoita siitä meidän keskustelussa. Luon englanninkielisen PR Story -tiedoston hakemistoon `.pr_stories/PR_STORY_PHASE_1.md`.
