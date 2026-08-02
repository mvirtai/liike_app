package config

import (
	"log"
	"os"
)

// Config sisältää sovelluksen ajoaikaiset asetukset
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
	jwtSecret := getEnv("JWT_SECRET", "dev-secret-key-change-in-production-123456")
	env := getEnv("APP_ENV", "development")

	cfg := &Config{
		Port:        port,
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
		Environment: env,
	}

	log.Printf("[CONFIG] Asetukset ladattu. Ympäristö: %s, Portti: %s",
		cfg.Environment, cfg.Port)
	return cfg
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}
