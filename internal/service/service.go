package service

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"assecor-assessment-backend/internal/domain"
	"assecor-assessment-backend/internal/repository"
)

// PersonService kapselt die Geschäftslogik für Personenoperationen.
type PersonService struct {
	repo   repository.PersonRepository
	logger *zap.Logger
}

// NewPersonService gibt einen einsatzbereiten PersonService zurück.
func NewPersonService(repo repository.PersonRepository, logger *zap.Logger) *PersonService {
	return &PersonService{repo: repo, logger: logger}
}

// GetAll gibt alle Personen zurück, optional paginiert.
func (s *PersonService) GetAll(ctx context.Context, limit, offset int) ([]domain.Person, error) {
	return s.repo.GetAll(ctx, limit, offset)
}

// GetByID sucht eine einzelne Person anhand ihrer ID.
func (s *PersonService) GetByID(ctx context.Context, id int) (domain.Person, error) {
	if id <= 0 {
		return domain.Person{}, fmt.Errorf("id muss positiv sein: %w", domain.ErrInvalidInput)
	}
	return s.repo.GetByID(ctx, id)
}

// GetByColor gibt alle Personen mit passender Lieblingsfarbe zurück.
func (s *PersonService) GetByColor(ctx context.Context, color string, limit, offset int) ([]domain.Person, error) {
	normalized := strings.ToLower(strings.TrimSpace(color))
	if _, ok := domain.ColorNameID[normalized]; !ok {
		s.logger.Warn("unbekannte farbe angefragt", zap.String("farbe", color))
		return nil, fmt.Errorf("ungültige farbe: %w", domain.ErrInvalidInput)
	}
	return s.repo.GetByColor(ctx, normalized, limit, offset)
}

// Add validiert und fügt eine neue Person hinzu. Der Farbname wird normalisiert.
func (s *PersonService) Add(ctx context.Context, person domain.Person) (domain.Person, error) {
	if person.Name == "" || person.Lastname == "" {
		return domain.Person{}, fmt.Errorf("name und nachname sind erforderlich: %w", domain.ErrInvalidInput)
	}
	person.Color = strings.ToLower(strings.TrimSpace(person.Color))
	if _, ok := domain.ColorNameID[person.Color]; !ok {
		s.logger.Warn("ungültige farbe beim erstellen", zap.String("farbe", person.Color))
		return domain.Person{}, fmt.Errorf("ungültige farbe: %w", domain.ErrInvalidInput)
	}
	return s.repo.Add(ctx, person)
}
