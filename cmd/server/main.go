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
	"liike_app/internal/middleware"
	"liike_app/internal/migrations"
	"liike_app/internal/repository"
	"liike_app/internal/service"
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

	log.Printf(
		"[DB] Tietokantayhteys luotu onnistuneesti: %s",
		cfg.DatabaseURL,
	)

	// 2. Migraatioiden ajo
	if err := database.RunMigrations(
		db,
		migrations.FS,
		migrations.Dir,
	); err != nil {
		log.Fatalf(
			"[DB] Virhe migraatioiden suorituksessa: %v",
			err,
		)
	}

	log.Println("[DB] Tietokantamigraatiot ajettu onnistuneesti.")

	// 3. Dependency Injection
	//
	// Rakennetaan riippuvuudet kerroksittain:
	//
	// Database
	//    ↓
	// Repository
	//    ↓
	// AuthService
	//    ↓
	// AuthHandler
	//
	// AuthMiddleware käyttää AuthServiceä JWT-tokenien
	// validoimiseen.
	repo := repository.NewRepository(db)

	authSvc := service.NewAuthService(
		repo,
		cfg.JWTSecret,
	)

	authHandler := handler.NewAuthHandler(authSvc)

	authMiddleware := middleware.AuthMiddleware(authSvc)

	// 4. Alusta HTTP ServeMux (Go 1.22+ reititys)
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc(
		"GET /api/v1/health",
		handler.HealthCheckHandler,
	)

	// 5. Julkiset auth-reitit
	//
	// Näihin ei tarvita JWT-tokenia.

	mux.HandleFunc(
		"POST /api/v1/auth/register",
		authHandler.Register,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/login",
		authHandler.Login,
	)

	// 6. Suojattu auth-reitti
	//
	// AuthMiddleware tarkistaa:
	//
	// Authorization: Bearer <JWT>
	//
	// ja lisää käyttäjän ID:n request contextiin.
	mux.Handle(
		"GET /api/v1/auth/me",
		authMiddleware(
			http.HandlerFunc(authHandler.Me),
		),
	)

	// 7. HTTP-palvelimen asetukset ja aikakatkaisut
	server := &http.Server{
		Addr: fmt.Sprintf(
			":%s",
			cfg.Port,
		),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 8. Käynnistä palvelin omassa goroutinessaan
	go func() {
		log.Printf(
			"[SERVER] Liike App backend käynnissä osoitteessa http://localhost%s",
			server.Addr,
		)

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Fatalf(
				"[SERVER] Kriittinen virhe palvelinta käynnistäessä: %v",
				err,
			)
		}
	}()

	// 9. Hallittu sammutus
	stop := make(chan os.Signal, 1)

	signal.Notify(
		stop,
		os.Interrupt,
		syscall.SIGTERM,
	)

	log.Println("[SERVER] Palvelin käynnistetty.")

	// Odotetaan sammutussignaalia (Ctrl+C tai SIGTERM)
	<-stop

	log.Println(
		"[SERVER] Sammutussignaali vastaanotettu. Suljetaan palvelin...",
	)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf(
			"[SERVER] Virhe palvelimen hallitussa sammutuksessa: %v", err)
	}

	log.Println("[SERVER] Palvelin suljettu onnistuneesti.")
}
