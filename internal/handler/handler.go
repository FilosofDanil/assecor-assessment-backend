package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"assecor-assessment-backend/internal/domain"
)

// maxRequestBody begrenzt die POST-Body-Größe auf 1 MegaByte
const maxRequestBody = 1 << 20

// PersonService definiert den Vertrag, den der Handler von der Service-Schicht erwartet.
type PersonService interface {
	GetAll(ctx context.Context) ([]domain.Person, error)
	GetByID(ctx context.Context, id int) (domain.Person, error)
	GetByColor(ctx context.Context, color string) ([]domain.Person, error)
	Add(ctx context.Context, person domain.Person) (domain.Person, error)
}

// PersonHandler stellt Personen-Endpunkte über HTTP bereit.
type PersonHandler struct {
	service PersonService
	logger  *zap.Logger
}

// NewPersonHandler erstellt einen neuen PersonHandler.
func NewPersonHandler(svc PersonService, logger *zap.Logger) *PersonHandler {
	return &PersonHandler{service: svc, logger: logger}
}

// GetAll gibt alle Personen zurück.
func (h *PersonHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	persons, err := h.service.GetAll(r.Context())
	if err != nil {
		h.logger.Error("alle personen abrufen", zap.Error(err))
		writeJSON(w, http.StatusInternalServerError, errorBody{"interner serverfehler"})
		return
	}
	writeJSON(w, http.StatusOK, persons)
}

// GetByID gibt eine einzelne Person anhand ihrer ID zurück.
func (h *PersonHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{"id muss eine ganzzahl sein"})
		return
	}

	person, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			writeJSON(w, http.StatusNotFound, errorBody{err.Error()})
		case errors.Is(err, domain.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, errorBody{err.Error()})
		default:
			h.logger.Error("person nach id abrufen", zap.Error(err))
			writeJSON(w, http.StatusInternalServerError, errorBody{"interner serverfehler"})
		}
		return
	}
	writeJSON(w, http.StatusOK, person)
}

// GetByColor gibt alle Personen mit passender Lieblingsfarbe zurück.
func (h *PersonHandler) GetByColor(w http.ResponseWriter, r *http.Request) {
	color := chi.URLParam(r, "color")

	persons, err := h.service.GetByColor(r.Context(), color)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, errorBody{err.Error()})
		default:
			h.logger.Error("personen nach farbe abrufen", zap.Error(err))
			writeJSON(w, http.StatusInternalServerError, errorBody{"interner serverfehler"})
		}
		return
	}
	writeJSON(w, http.StatusOK, persons)
}

// Create fügt einen neuen Personendatensatz hinzu.
// Der Request-Body wird auf maxRequestBody begrenzt (Exploit 1).
func (h *PersonHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	var p domain.Person
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{"ungültiger anfrage-body"})
		return
	}

	created, err := h.service.Add(r.Context(), p)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCapacityReached):
			writeJSON(w, http.StatusServiceUnavailable, errorBody{err.Error()})
		case errors.Is(err, domain.ErrInvalidInput):
			writeJSON(w, http.StatusBadRequest, errorBody{err.Error()})
		default:
			h.logger.Error("person erstellen", zap.Error(err))
			writeJSON(w, http.StatusInternalServerError, errorBody{"interner serverfehler"})
		}
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// errorBody ist die einheitliche Fehlerantwort-Struktur.
type errorBody struct {
	Error string `json:"error"`
}

// writeJSON setzt den Content-Type-Header und schreibt v als JSON in w.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
