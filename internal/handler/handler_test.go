package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"assecor-assessment-backend/internal/domain"
)

// mockService implementiert PersonService für Handler-Tests.
type mockService struct {
	persons []domain.Person
	nextID  int
}

func newMockService(persons []domain.Person) *mockService {
	return &mockService{persons: persons, nextID: len(persons) + 1}
}

func (m *mockService) GetAll(_ context.Context, limit, offset int) ([]domain.Person, error) {
	out := make([]domain.Person, len(m.persons))
	copy(out, m.persons)
	if offset > 0 && offset < len(out) {
		out = out[offset:]
	} else if offset >= len(out) {
		return make([]domain.Person, 0), nil
	}
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

func (m *mockService) GetByID(_ context.Context, id int) (domain.Person, error) {
	if id <= 0 {
		return domain.Person{}, fmt.Errorf("id muss positiv sein: %w", domain.ErrInvalidInput)
	}
	for _, p := range m.persons {
		if p.ID == id {
			return p, nil
		}
	}
	return domain.Person{}, fmt.Errorf("person mit id %d: %w", id, domain.ErrNotFound)
}

func (m *mockService) GetByColor(_ context.Context, color string, _, _ int) ([]domain.Person, error) {
	if _, ok := domain.ColorNameID[color]; !ok {
		return nil, fmt.Errorf("ungültige farbe: %w", domain.ErrInvalidInput)
	}
	out := make([]domain.Person, 0)
	for _, p := range m.persons {
		if p.Color == color {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *mockService) Add(_ context.Context, person domain.Person) (domain.Person, error) {
	if person.Name == "" || person.Lastname == "" {
		return domain.Person{}, fmt.Errorf("name und nachname sind erforderlich: %w", domain.ErrInvalidInput)
	}
	if _, ok := domain.ColorNameID[person.Color]; !ok {
		return domain.Person{}, fmt.Errorf("ungültige farbe: %w", domain.ErrInvalidInput)
	}
	person.ID = m.nextID
	m.nextID++
	m.persons = append(m.persons, person)
	return person, nil
}

func setupRouter(h *PersonHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/persons", h.GetAll)
	r.Post("/persons", h.Create)
	r.Get("/persons/{id}", h.GetByID)
	r.Get("/persons/color/{color}", h.GetByColor)
	return r
}

func neuerTestHandler() (*PersonHandler, *chi.Mux) {
	logger, _ := zap.NewDevelopment()
	svc := newMockService([]domain.Person{
		{ID: 1, Name: "Hans", Lastname: "Müller", Zipcode: "67742", City: "Lauterecken", Color: "blau"},
		{ID: 2, Name: "Peter", Lastname: "Petersen", Zipcode: "18439", City: "Stralsund", Color: "grün"},
		{ID: 3, Name: "Johnny", Lastname: "Johnson", Zipcode: "88888", City: "made up", Color: "violett"},
	})
	h := NewPersonHandler(svc, logger)
	return h, setupRouter(h)
}

func TestGetAll_GibtPersonenZurueck(t *testing.T) {
	_, router := neuerTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/persons", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var persons []domain.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&persons))
	assert.Len(t, persons, 3)
}

func TestGetAll_MitPaginierung(t *testing.T) {
	_, router := neuerTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/persons?limit=2&offset=1", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var persons []domain.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&persons))
	assert.Len(t, persons, 2)
}

func TestGetByID_Gefunden(t *testing.T) {
	_, router := neuerTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/persons/1", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var p domain.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&p))
	assert.Equal(t, "Hans", p.Name)
}

func TestGetByID_NichtGefunden(t *testing.T) {
	_, router := neuerTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/persons/999", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetByID_UngueltigeID(t *testing.T) {
	_, router := neuerTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/persons/abc", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetByID_NegativeID(t *testing.T) {
	_, router := neuerTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/persons/-1", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetByColor_Gefunden(t *testing.T) {
	_, router := neuerTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/persons/color/blau", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var persons []domain.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&persons))
	assert.Len(t, persons, 1)
}

func TestGetByColor_Leer(t *testing.T) {
	_, router := neuerTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/persons/color/gelb", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var persons []domain.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&persons))
	assert.Empty(t, persons)
}

func TestGetByColor_UnbekannteFarbe(t *testing.T) {
	_, router := neuerTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/persons/color/pink", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreate_Gueltig(t *testing.T) {
	_, router := neuerTestHandler()
	body := `{"name":"Neu","lastname":"Person","zipcode":"00000","city":"Stadt","color":"rot"}`
	req := httptest.NewRequest(http.MethodPost, "/persons", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var p domain.Person
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&p))
	assert.Equal(t, 4, p.ID)
	assert.Equal(t, "rot", p.Color)
}

func TestCreate_FehlenderName(t *testing.T) {
	_, router := neuerTestHandler()
	body := `{"lastname":"Person","color":"rot"}`
	req := httptest.NewRequest(http.MethodPost, "/persons", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreate_UngueltigesJSON(t *testing.T) {
	_, router := neuerTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/persons", bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreate_UnbekannteFarbe(t *testing.T) {
	_, router := neuerTestHandler()
	body := `{"name":"A","lastname":"B","color":"neon"}`
	req := httptest.NewRequest(http.MethodPost, "/persons", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
