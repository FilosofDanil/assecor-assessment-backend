package routes

import (
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"assecor-assessment-backend/internal/handler"
	"assecor-assessment-backend/internal/middleware"
)

// Setup registriert globale Middleware und alle Personen-Endpunkte am Router.
func Setup(r chi.Router, h *handler.PersonHandler, logger *zap.Logger, rps float64) {
	r.Use(chimw.RequestID)
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logging(logger))
	r.Use(middleware.RateLimit(rps, logger))

	r.Route("/persons", func(r chi.Router) {
		r.Get("/", h.GetAll)
		r.Post("/", h.Create)
		r.Get("/{id}", h.GetByID)
		r.Get("/color/{color}", h.GetByColor)
	})
}
