package env

import (
	"os"
	"strconv"
)

// Config enthält alle konfigurierbaren Werte der Anwendung, die über Umgebungsvariablen gesetzt werden können.
type Config struct {
	ServerAddr  string  // SERVER_ADDR – Adresse des HTTP-Servers (Standard: ":8081")
	CSVFilePath string  // CSV_FILE_PATH – Path zur CSV-Datei (Standard: "sample-input.csv")
	DataSource  string  // DATA_SOURCE – "csv" oder "sqlite" (Standard: "csv")
	RateLimit   float64 // RATE_LIMIT – Erlaubte Anfragen pro Sekunde (Standard: 100)
	MaxPersons  int     // MAX_PERSONS – Max. Anzahl Personen im Speicher (Standard: 10000)
}

// MustLoad liest die Konfiguration aus Umgebungsvariablen.
func MustLoad() Config {
	return Config{
		ServerAddr:  getOr("SERVER_ADDR", ":8081"),
		CSVFilePath: getOr("CSV_FILE_PATH", "sample-input.csv"),
		DataSource:  getOr("DATA_SOURCE", "csv"),
		RateLimit:   getFloatOr("RATE_LIMIT", 100),
		MaxPersons:  getIntOr("MAX_PERSONS", 10_000),
	}
}

func getOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getFloatOr(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
