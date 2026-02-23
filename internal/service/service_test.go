package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"assecor-assessment-backend/internal/domain"
)

// mockRepo ist ein Test-Double, das repository.PersonRepository implementiert.
type mockRepo struct {
	persons []domain.Person
	nextID  int
}

func newMockRepo(persons []domain.Person) *mockRepo {
	return &mockRepo{persons: persons, nextID: len(persons) + 1}
}

func (m *mockRepo) GetAll(_ context.Context, limit, offset int) ([]domain.Person, error) {
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

func (m *mockRepo) GetByID(_ context.Context, id int) (domain.Person, error) {
	for _, p := range m.persons {
		if p.ID == id {
			return p, nil
		}
	}
	return domain.Person{}, fmt.Errorf("person mit id %d: %w", id, domain.ErrNotFound)
}

func (m *mockRepo) GetByColor(_ context.Context, color string, limit, offset int) ([]domain.Person, error) {
	var matched []domain.Person
	for _, p := range m.persons {
		if p.Color == color {
			matched = append(matched, p)
		}
	}
	if matched == nil {
		matched = make([]domain.Person, 0)
	}
	return matched, nil
}

func (m *mockRepo) Add(_ context.Context, person domain.Person) (domain.Person, error) {
	person.ID = m.nextID
	m.nextID++
	m.persons = append(m.persons, person)
	return person, nil
}

func seedRepo() *mockRepo {
	return newMockRepo([]domain.Person{
		{ID: 1, Name: "Hans", Lastname: "Müller", Zipcode: "67742", City: "Lauterecken", Color: "blau"},
		{ID: 2, Name: "Peter", Lastname: "Petersen", Zipcode: "18439", City: "Stralsund", Color: "grün"},
	})
}

func neuerTestService(repo *mockRepo) *PersonService {
	logger, _ := zap.NewDevelopment()
	return NewPersonService(repo, logger)
}

func TestGetAll(t *testing.T) {
	svc := neuerTestService(seedRepo())
	persons, err := svc.GetAll(context.Background(), 0, 0)
	require.NoError(t, err)
	assert.Len(t, persons, 2)
}

func TestGetByID_Gueltig(t *testing.T) {
	svc := neuerTestService(seedRepo())
	p, err := svc.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "Hans", p.Name)
}

func TestGetByID_NichtGefunden(t *testing.T) {
	svc := neuerTestService(seedRepo())
	_, err := svc.GetByID(context.Background(), 99)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetByID_UngueltigeID(t *testing.T) {
	svc := neuerTestService(seedRepo())
	_, err := svc.GetByID(context.Background(), 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestGetByColor_Gueltig(t *testing.T) {
	svc := neuerTestService(seedRepo())
	persons, err := svc.GetByColor(context.Background(), "blau", 0, 0)
	require.NoError(t, err)
	assert.Len(t, persons, 1)
}

func TestGetByColor_Grossschreibung(t *testing.T) {
	svc := neuerTestService(seedRepo())
	persons, err := svc.GetByColor(context.Background(), "Blau", 0, 0)
	require.NoError(t, err)
	assert.Len(t, persons, 1)

	persons2, err := svc.GetByColor(context.Background(), "BLAU", 0, 0)
	require.NoError(t, err)
	assert.Len(t, persons2, 1)
}

func TestGetByColor_UnbekannteFarbe(t *testing.T) {
	svc := neuerTestService(seedRepo())
	_, err := svc.GetByColor(context.Background(), "pink", 0, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestGetByColor_GenericErrorOhneUserInput(t *testing.T) {
	svc := neuerTestService(seedRepo())
	_, err := svc.GetByColor(context.Background(), "xss<script>", 0, 0)
	require.Error(t, err)
	// Exploit 5: Die Fehlermeldung darf die Benutzereingabe nicht enthalten.
	assert.NotContains(t, err.Error(), "xss<script>")
}

func TestAdd_Gueltig(t *testing.T) {
	repo := seedRepo()
	svc := neuerTestService(repo)
	created, err := svc.Add(context.Background(), domain.Person{
		Name: "Neu", Lastname: "Benutzer", Color: "rot",
	})
	require.NoError(t, err)
	assert.Equal(t, 3, created.ID)
}

func TestAdd_FarbeGrossschreibung(t *testing.T) {
	svc := neuerTestService(seedRepo())
	created, err := svc.Add(context.Background(), domain.Person{
		Name: "A", Lastname: "B", Color: "ROT",
	})
	require.NoError(t, err)
	assert.Equal(t, "rot", created.Color)
}

func TestAdd_FehlenderName(t *testing.T) {
	svc := neuerTestService(seedRepo())
	_, err := svc.Add(context.Background(), domain.Person{Lastname: "Benutzer", Color: "rot"})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestAdd_UnbekannteFarbe(t *testing.T) {
	svc := neuerTestService(seedRepo())
	_, err := svc.Add(context.Background(), domain.Person{Name: "A", Lastname: "B", Color: "neon"})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}
