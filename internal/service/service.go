package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"go.uber.org/zap"

	"assecor-assessment-backend/internal/domain"
	"assecor-assessment-backend/internal/repository"
)

const (
	nameMinLen    = 2
	nameMaxLen    = 255
	zipcodeMaxLen = 5
	cityMinLen    = 2
	cityMaxLen    = 255
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

// GetAll gibt alle Personen zurück.
func (s *PersonService) GetAll(ctx context.Context) ([]domain.Person, error) {
	return s.repo.GetAll(ctx)
}

// GetByID sucht eine einzelne Person anhand ihrer ID.
func (s *PersonService) GetByID(ctx context.Context, id int) (domain.Person, error) {
	if id <= 0 {
		return domain.Person{}, fmt.Errorf("id muss positiv sein: %w", domain.ErrInvalidInput)
	}
	return s.repo.GetByID(ctx, id)
}

// GetByColor gibt alle Personen mit passender Lieblingsfarbe zurück.
func (s *PersonService) GetByColor(ctx context.Context, color string) ([]domain.Person, error) {
	normalized := strings.ToLower(strings.TrimSpace(color))
	if _, ok := domain.ColorNameID[normalized]; !ok {
		s.logger.Warn("unbekannte farbe angefragt", zap.String("farbe", color))
		return nil, fmt.Errorf("ungültige farbe: %w", domain.ErrInvalidInput)
	}
	return s.repo.GetByColor(ctx, normalized)
}

// Add validiert und fügt eine neue Person hinzu. Der Farbname wird normalisiert.
func (s *PersonService) Add(ctx context.Context, person domain.Person) (domain.Person, error) {
	person.Name = strings.TrimSpace(person.Name)
	person.Lastname = strings.TrimSpace(person.Lastname)
	person.Zipcode = strings.TrimSpace(person.Zipcode)
	person.City = strings.TrimSpace(person.City)
	person.Color = strings.ToLower(strings.TrimSpace(person.Color))

	if err := validatePerson(person); err != nil {
		return domain.Person{}, err
	}

	if _, ok := domain.ColorNameID[person.Color]; !ok {
		s.logger.Warn("ungültige farbe beim erstellen", zap.String("farbe", person.Color))
		return domain.Person{}, fmt.Errorf("ungültige farbe: %w", domain.ErrInvalidInput)
	}
	return s.repo.Add(ctx, person)
}

// validatePerson prüft alle Pflichtfelder und Längengrenzen einer Person.
func validatePerson(p domain.Person) error {
	if err := checkLength("vorname", p.Name, nameMinLen, nameMaxLen); err != nil {
		return err
	}
	if err := checkLength("nachname", p.Lastname, nameMinLen, nameMaxLen); err != nil {
		return err
	}
	if p.Zipcode == "" {
		return fmt.Errorf("postleitzahl ist erforderlich: %w", domain.ErrInvalidInput)
	}
	if n := utf8.RuneCountInString(p.Zipcode); n > zipcodeMaxLen {
		return fmt.Errorf("postleitzahl darf maximal %d zeichen lang sein: %w", zipcodeMaxLen, domain.ErrInvalidInput)
	}
	if err := checkLength("stadt", p.City, cityMinLen, cityMaxLen); err != nil {
		return err
	}
	return nil
}

// checkLength gibt ErrInvalidInput zurück, wenn die Zeichenanzahl von s
func checkLength(field, s string, min, max int) error {
	n := utf8.RuneCountInString(s)
	if n < min {
		return fmt.Errorf("%s muss mindestens %d zeichen lang sein: %w", field, min, domain.ErrInvalidInput)
	}
	if n > max {
		return fmt.Errorf("%s darf maximal %d zeichen lang sein: %w", field, max, domain.ErrInvalidInput)
	}
	return nil
}
