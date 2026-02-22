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

	"assecor-assessment-backend/internal/handler"
	csvrepo "assecor-assessment-backend/internal/repository/csv"
	"assecor-assessment-backend/internal/routes"
	"assecor-assessment-backend/internal/service"
)

func main() {
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	repo, err := csvrepo.NewPersonRepository("sample-input.csv", logger)
	if err != nil {
		logger.Fatal("csv-repository konnte nicht geladen werden", zap.Error(err))
	}

	svc := service.NewPersonService(repo, logger)
	h := handler.NewPersonHandler(svc, logger)

	r := chi.NewRouter()
	routes.Setup(r, h, logger)

	srv := &http.Server{
		Addr:         ":8080",
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
