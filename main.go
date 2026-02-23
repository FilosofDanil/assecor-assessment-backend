package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"assecor-assessment-backend/internal/env"
	"assecor-assessment-backend/internal/handler"
	"assecor-assessment-backend/internal/repository"
	csvrepo "assecor-assessment-backend/internal/repository/csv"
	sqliterepo "assecor-assessment-backend/internal/repository/sqlite"
	"assecor-assessment-backend/internal/routes"
	"assecor-assessment-backend/internal/service"
)

func main() {
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	cfg := env.MustLoad()
	logger.Info("konfiguration geladen",
		zap.String("data_source", cfg.DataSource),
		zap.String("csv_file_path", cfg.CSVFilePath),
		zap.String("server_addr", cfg.ServerAddr),
		zap.Float64("rate_limit", cfg.RateLimit),
		zap.Int("max_persons", cfg.MaxPersons),
	)

	repo, cleanup := mustInitRepo(cfg, logger)
	if cleanup != nil {
		defer cleanup()
	}

	svc := service.NewPersonService(repo, logger)
	h := handler.NewPersonHandler(svc, logger)

	r := chi.NewRouter()
	routes.Setup(r, h, logger, cfg.RateLimit)

	srv := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		logger.Info("server wird gestartet", zap.String("adresse", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("server wird heruntergefahren")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("erzwungenes herunterfahren", zap.Error(err))
	}
	logger.Info("server gestoppt")
}

// mustInitRepo erstellt je nach DATA_SOURCE das passende PersonRepository.
// Bei "sqlite" wird eine In-Memory-Datenbank verwendet; die zurückgegebene
// cleanup-Funktion schließt die DB-Verbindung.
func mustInitRepo(cfg env.Config, logger *zap.Logger) (repository.PersonRepository, func()) {
	switch cfg.DataSource {
	case "sqlite":
		repo, err := sqliterepo.NewPersonRepository(":memory:", cfg.MaxPersons, logger)
		if err != nil {
			logger.Fatal("sqlite-repository konnte nicht initialisiert werden", zap.Error(err))
		}
		return repo, func() { _ = repo.Close() }

	default:
		repo, err := csvrepo.NewPersonRepository(cfg.CSVFilePath, cfg.MaxPersons, logger)
		if err != nil {
			logger.Fatal("csv-repository konnte nicht geladen werden", zap.Error(err))
		}
		return repo, nil
	}
}
