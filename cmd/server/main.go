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
	// Database -> Repository -> Services -> Handlers -> Middleware
	repo := repository.NewRepository(db)

	authSvc := service.NewAuthService(
		repo,
		cfg.JWTSecret,
	)
	authHandler := handler.NewAuthHandler(authSvc)
	authMiddleware := middleware.AuthMiddleware(authSvc)

	exerciseTypeSvc := service.NewExerciseTypeService(repo)
	exerciseTypeHandler := handler.NewExerciseTypeHandler(exerciseTypeSvc)

	workoutSvc := service.NewWorkoutService(repo)
	workoutHandler := handler.NewWorkoutHandler(workoutSvc)

	// 4. Alusta HTTP ServeMux (Go 1.22+ reititys)
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc(
		"GET /api/v1/health",
		handler.HealthCheckHandler,
	)

	// 5. Julkiset auth-reitit (ei tarvita JWT-tokenia)
	mux.HandleFunc(
		"POST /api/v1/auth/register",
		authHandler.Register,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/login",
		authHandler.Login,
	)

	// 6. Suojatut auth-reitit
	mux.Handle(
		"GET /api/v1/auth/me",
		authMiddleware(
			http.HandlerFunc(authHandler.Me),
		),
	)

	// 7. Harjoitusmuotojen reitit (suojatut)
	mux.Handle(
		"GET /api/v1/exercise-types",
		authMiddleware(
			http.HandlerFunc(exerciseTypeHandler.List),
		),
	)

	mux.Handle(
		"GET /api/v1/exercise-types/{id}",
		authMiddleware(
			http.HandlerFunc(exerciseTypeHandler.GetByID),
		),
	)

	// 8. Suoritusten & Jousiammunnan reitit (suojatut)
	mux.Handle(
		"POST /api/v1/workouts",
		authMiddleware(
			http.HandlerFunc(workoutHandler.Create),
		),
	)

	mux.Handle(
		"GET /api/v1/workouts",
		authMiddleware(
			http.HandlerFunc(workoutHandler.List),
		),
	)

	mux.Handle(
		"GET /api/v1/workouts/{id}",
		authMiddleware(
			http.HandlerFunc(workoutHandler.GetByID),
		),
	)

	mux.Handle(
		"DELETE /api/v1/workouts/{id}",
		authMiddleware(
			http.HandlerFunc(workoutHandler.Delete),
		),
	)

	mux.Handle(
		"POST /api/v1/workouts/{id}/archery-scores",
		authMiddleware(
			http.HandlerFunc(workoutHandler.AddArcheryScores),
		),
	)

	// 9. HTTP-palvelimen asetukset ja aikakatkaisut
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

	// 10. Käynnistä palvelin omassa goroutinessaan
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

	// 11. Hallittu sammutus
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
