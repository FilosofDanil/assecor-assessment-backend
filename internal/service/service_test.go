package service

import (
	"context"
	"fmt"
	"strings"
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

func (m *mockRepo) GetAll(_ context.Context) ([]domain.Person, error) {
	out := make([]domain.Person, len(m.persons))
	copy(out, m.persons)
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

func (m *mockRepo) GetByColor(_ context.Context, color string) ([]domain.Person, error) {
	out := make([]domain.Person, 0)
	for _, p := range m.persons {
		if p.Color == color {
			out = append(out, p)
		}
	}
	return out, nil
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

// validePerson gibt eine vollständig gültige Person zurück, die alle
// Validierungsregeln erfüllt. Einzelne Felder können in Tests überschrieben werden.
func validePerson() domain.Person {
	return domain.Person{
		Name:     "Hans",
		Lastname: "Müller",
		Zipcode:  "67742",
		City:     "Musterstadt",
		Color:    "rot",
	}
}

// ─── GetAll ───────────────────────────────────────────────────────────────────

func TestGetAll(t *testing.T) {
	svc := neuerTestService(seedRepo())
	persons, err := svc.GetAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, persons, 2)
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

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

// ─── GetByColor ───────────────────────────────────────────────────────────────

func TestGetByColor_Gueltig(t *testing.T) {
	svc := neuerTestService(seedRepo())
	persons, err := svc.GetByColor(context.Background(), "blau")
	require.NoError(t, err)
	assert.Len(t, persons, 1)
}

func TestGetByColor_Grossschreibung(t *testing.T) {
	svc := neuerTestService(seedRepo())
	persons, err := svc.GetByColor(context.Background(), "Blau")
	require.NoError(t, err)
	assert.Len(t, persons, 1)

	persons2, err := svc.GetByColor(context.Background(), "BLAU")
	require.NoError(t, err)
	assert.Len(t, persons2, 1)
}

func TestGetByColor_UnbekannteFarbe(t *testing.T) {
	svc := neuerTestService(seedRepo())
	_, err := svc.GetByColor(context.Background(), "pink")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestGetByColor_GenericErrorOhneUserInput(t *testing.T) {
	svc := neuerTestService(seedRepo())
	_, err := svc.GetByColor(context.Background(), "xss<script>")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "xss<script>")
}

// ─── Add ──────────────────────────────────────────────────────────────────────

func TestAdd_Gueltig(t *testing.T) {
	repo := seedRepo()
	svc := neuerTestService(repo)
	created, err := svc.Add(context.Background(), validePerson())
	require.NoError(t, err)
	assert.Equal(t, 3, created.ID)
}

func TestAdd_FarbeGrossschreibung(t *testing.T) {
	svc := neuerTestService(seedRepo())
	p := validePerson()
	p.Color = "ROT"
	created, err := svc.Add(context.Background(), p)
	require.NoError(t, err)
	assert.Equal(t, "rot", created.Color)
}

func TestAdd_FuehrendeLeerzechenWerdenGetrimmt(t *testing.T) {
	svc := neuerTestService(seedRepo())
	p := validePerson()
	p.Name = "  Hans  "
	p.Lastname = "  Müller  "
	p.Zipcode = "  12345  "
	p.City = "  Berlin  "
	created, err := svc.Add(context.Background(), p)
	require.NoError(t, err)
	assert.Equal(t, "Hans", created.Name)
	assert.Equal(t, "Müller", created.Lastname)
	assert.Equal(t, "12345", created.Zipcode)
	assert.Equal(t, "Berlin", created.City)
}

// ─── Add – Name / Nachname ────────────────────────────────────────────────────

func TestAdd_NameValidierung(t *testing.T) {
	tests := []struct {
		name     string
		person   domain.Person
		wantErr  bool
		errField string
	}{
		{
			name:    "name leer → Fehler",
			person:  func() domain.Person { p := validePerson(); p.Name = ""; return p }(),
			wantErr: true,
		},
		{
			name:    "name ein zeichen → Fehler",
			person:  func() domain.Person { p := validePerson(); p.Name = "A"; return p }(),
			wantErr: true,
		},
		{
			name:    "name genau zwei zeichen → gültig",
			person:  func() domain.Person { p := validePerson(); p.Name = "Li"; return p }(),
			wantErr: false,
		},
		{
			name:    "name 255 zeichen → gültig",
			person:  func() domain.Person { p := validePerson(); p.Name = strings.Repeat("a", 255); return p }(),
			wantErr: false,
		},
		{
			name:    "name 256 zeichen → Fehler",
			person:  func() domain.Person { p := validePerson(); p.Name = strings.Repeat("a", 256); return p }(),
			wantErr: true,
		},
		{
			name:    "name mit unicode-zeichen (umlaut) korrekt gezählt",
			person:  func() domain.Person { p := validePerson(); p.Name = "Ös"; return p }(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := neuerTestService(seedRepo())
			_, err := svc.Add(context.Background(), tt.person)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, domain.ErrInvalidInput)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAdd_NachnameValidierung(t *testing.T) {
	tests := []struct {
		name    string
		person  domain.Person
		wantErr bool
	}{
		{
			name:    "nachname leer → Fehler",
			person:  func() domain.Person { p := validePerson(); p.Lastname = ""; return p }(),
			wantErr: true,
		},
		{
			name:    "nachname ein zeichen → Fehler",
			person:  func() domain.Person { p := validePerson(); p.Lastname = "X"; return p }(),
			wantErr: true,
		},
		{
			name:    "nachname genau zwei zeichen → gültig",
			person:  func() domain.Person { p := validePerson(); p.Lastname = "Li"; return p }(),
			wantErr: false,
		},
		{
			name:    "nachname 256 zeichen → Fehler",
			person:  func() domain.Person { p := validePerson(); p.Lastname = strings.Repeat("b", 256); return p }(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := neuerTestService(seedRepo())
			_, err := svc.Add(context.Background(), tt.person)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, domain.ErrInvalidInput)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── Add – Postleitzahl ───────────────────────────────────────────────────────

func TestAdd_PostleitzahlValidierung(t *testing.T) {
	tests := []struct {
		name    string
		person  domain.Person
		wantErr bool
	}{
		{
			name:    "postleitzahl leer → Fehler",
			person:  func() domain.Person { p := validePerson(); p.Zipcode = ""; return p }(),
			wantErr: true,
		},
		{
			name:    "postleitzahl 1 zeichen → gültig",
			person:  func() domain.Person { p := validePerson(); p.Zipcode = "1"; return p }(),
			wantErr: false,
		},
		{
			name:    "postleitzahl 20 zeichen → gültig",
			person:  func() domain.Person { p := validePerson(); p.Zipcode = strings.Repeat("1", 20); return p }(),
			wantErr: false,
		},
		{
			name:    "postleitzahl 21 zeichen → Fehler",
			person:  func() domain.Person { p := validePerson(); p.Zipcode = strings.Repeat("1", 21); return p }(),
			wantErr: true,
		},
		{
			name:    "deutsche PLZ fünfstellig → gültig",
			person:  func() domain.Person { p := validePerson(); p.Zipcode = "67742"; return p }(),
			wantErr: false,
		},
		{
			name:    "britische PLZ mit leerzeichen → gültig",
			person:  func() domain.Person { p := validePerson(); p.Zipcode = "SW1A 1AA"; return p }(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := neuerTestService(seedRepo())
			_, err := svc.Add(context.Background(), tt.person)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, domain.ErrInvalidInput)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── Add – Stadt ──────────────────────────────────────────────────────────────

func TestAdd_StadtValidierung(t *testing.T) {
	tests := []struct {
		name    string
		person  domain.Person
		wantErr bool
	}{
		{
			name:    "stadt leer → Fehler",
			person:  func() domain.Person { p := validePerson(); p.City = ""; return p }(),
			wantErr: true,
		},
		{
			name:    "stadt ein zeichen → Fehler",
			person:  func() domain.Person { p := validePerson(); p.City = "X"; return p }(),
			wantErr: true,
		},
		{
			name:    "stadt genau zwei zeichen → gültig",
			person:  func() domain.Person { p := validePerson(); p.City = "Aa"; return p }(),
			wantErr: false,
		},
		{
			name:    "stadt 255 zeichen → gültig",
			person:  func() domain.Person { p := validePerson(); p.City = strings.Repeat("s", 255); return p }(),
			wantErr: false,
		},
		{
			name:    "stadt 256 zeichen → Fehler",
			person:  func() domain.Person { p := validePerson(); p.City = strings.Repeat("s", 256); return p }(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := neuerTestService(seedRepo())
			_, err := svc.Add(context.Background(), tt.person)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, domain.ErrInvalidInput)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ─── Add – Farbe ──────────────────────────────────────────────────────────────

func TestAdd_FehlenderName(t *testing.T) {
	svc := neuerTestService(seedRepo())
	p := validePerson()
	p.Name = ""
	_, err := svc.Add(context.Background(), p)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestAdd_UnbekannteFarbe(t *testing.T) {
	svc := neuerTestService(seedRepo())
	p := validePerson()
	p.Color = "neon"
	_, err := svc.Add(context.Background(), p)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}
