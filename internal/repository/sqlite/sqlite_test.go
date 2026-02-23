package sqlite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"assecor-assessment-backend/internal/domain"
)

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func seedRepo(t *testing.T, maxPersons int) *PersonRepository {
	t.Helper()
	repo, err := NewPersonRepository(":memory:", maxPersons, testLogger())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	seed := []domain.Person{
		{Name: "Hans", Lastname: "Müller", Zipcode: "67742", City: "Lauterecken", Color: "blau"},
		{Name: "Peter", Lastname: "Petersen", Zipcode: "18439", City: "Stralsund", Color: "grün"},
		{Name: "Johnny", Lastname: "Johnson", Zipcode: "88888", City: "made up", Color: "blau"},
	}
	for _, p := range seed {
		_, err := repo.Add(context.Background(), p)
		require.NoError(t, err)
	}
	return repo
}

func TestGetAll(t *testing.T) {
	repo := seedRepo(t, 0)
	all, err := repo.GetAll(context.Background(), 0, 0)
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestGetAll_Paginierung(t *testing.T) {
	repo := seedRepo(t, 0)

	tests := []struct {
		name    string
		limit   int
		offset  int
		wantLen int
	}{
		{"alle ohne Limit", 0, 0, 3},
		{"limit 2", 2, 0, 2},
		{"offset 1", 0, 1, 2},
		{"limit 1 offset 1", 1, 1, 1},
		{"offset über Gesamtzahl", 0, 99, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persons, err := repo.GetAll(context.Background(), tt.limit, tt.offset)
			require.NoError(t, err)
			assert.NotNil(t, persons)
			assert.Len(t, persons, tt.wantLen)
		})
	}
}

func TestGetByID(t *testing.T) {
	repo := seedRepo(t, 0)

	p, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "Hans", p.Name)

	_, err = repo.GetByID(context.Background(), 999)
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetByColor(t *testing.T) {
	repo := seedRepo(t, 0)

	blau, err := repo.GetByColor(context.Background(), "blau", 0, 0)
	require.NoError(t, err)
	assert.Len(t, blau, 2)

	gruen, err := repo.GetByColor(context.Background(), "grün", 0, 0)
	require.NoError(t, err)
	assert.Len(t, gruen, 1)

	rot, err := repo.GetByColor(context.Background(), "rot", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, rot)
	assert.Empty(t, rot)
}

func TestAdd_AutoIncrementID(t *testing.T) {
	repo, err := NewPersonRepository(":memory:", 0, testLogger())
	require.NoError(t, err)
	defer func() { _ = repo.Close() }()

	p1, err := repo.Add(context.Background(), domain.Person{Name: "A", Lastname: "B", Color: "rot"})
	require.NoError(t, err)
	assert.Equal(t, 1, p1.ID)

	p2, err := repo.Add(context.Background(), domain.Person{Name: "C", Lastname: "D", Color: "blau"})
	require.NoError(t, err)
	assert.Equal(t, 2, p2.ID)
}

func TestAdd_KapazitaetsgrenzExploit3(t *testing.T) {
	repo := seedRepo(t, 4)

	_, err := repo.Add(context.Background(), domain.Person{Name: "Noch", Lastname: "Einer", Color: "rot"})
	require.NoError(t, err)

	_, err = repo.Add(context.Background(), domain.Person{Name: "Zu", Lastname: "Viel", Color: "blau"})
	require.ErrorIs(t, err, domain.ErrCapacityReached)
}
