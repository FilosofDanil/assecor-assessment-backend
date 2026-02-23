package repository

import (
	"context"

	"assecor-assessment-backend/internal/domain"
)

// PersonRepository abstrahiert den Datenzugriff auf Personen, sodass die
type PersonRepository interface {
	GetAll(ctx context.Context, limit, offset int) ([]domain.Person, error)
	GetByID(ctx context.Context, id int) (domain.Person, error)
	GetByColor(ctx context.Context, color string, limit, offset int) ([]domain.Person, error)
	Add(ctx context.Context, person domain.Person) (domain.Person, error)
}
