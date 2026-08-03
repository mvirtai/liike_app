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
	"liike_app/internal/database"
	"liike_app/internal/handler"
	"liike_app/internal/migrations"
)

func main() {
	// 0. Lataa konfiguraatio
	cfg := config.Load()

	// 1. Tietokannan alustus ja PRAGMAt
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[DB] Kriittinen virhe tietokantayhteydessä: %s", cfg.DatabaseURL)
	}
	defer db.Close()
	log.Printf("[DB] Tietokantayhteys luotu onnistuneesti: %s", cfg.DatabaseURL)
	// 2. Migraatioiden ajo (UUID schema)
	if err := database.RunMigrations(db, migrations.FS, migrations.Dir); err != nil {
		log.Fatalf("[DB] Virhe migraatioiden suorituksessa: %v", err)
	}
	log.Println("[DB] Tietokantamigraatiot ajettu onnistuneesti.")

	// 3. Alusta HTTP ServeMux (Go 1.22+ reititys)
	mux := http.NewServeMux()

	// 4. Rekisteröi reitit
	mux.HandleFunc("GET /api/v1/health", handler.HealthCheckHandler)

	// 4. Määritä HTTP-palvelimen asetukset ja aikakatkaisut
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 5. Käynnistä palvelin omassa goroutinessaan
	go func() {
		log.Printf("[SERVER] Liike App backend käynnissä osoitteessa http://localhost%s",
			server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[SERVER] Kriittinen virhe palvelinta käynnistäessä: %v", err)
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
