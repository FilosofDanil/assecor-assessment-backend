package csv

import (
	"context"
	"os"
	"path/filepath"
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

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestLoad_NormaleEintraege(t *testing.T) {
	data := "Müller, Hans, 67742 Lauterecken, 1\nPetersen, Peter, 18439 Stralsund, 2\n"
	repo, err := NewPersonRepository(writeTempCSV(t, data), testLogger())
	require.NoError(t, err)

	all, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, all, 2)

	assert.Equal(t, domain.Person{ID: 1, Name: "Hans", Lastname: "Müller", Zipcode: "67742", City: "Lauterecken", Color: "blau"}, all[0])
	assert.Equal(t, domain.Person{ID: 2, Name: "Peter", Lastname: "Petersen", Zipcode: "18439", City: "Stralsund", Color: "grün"}, all[1])
}

func TestLoad_MehrzeiligeDatensatz(t *testing.T) {
	data := "Bart, Bertram, \n12313 Wasweißich, 1\n"
	repo, err := NewPersonRepository(writeTempCSV(t, data), testLogger())
	require.NoError(t, err)

	all, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	require.Len(t, all, 1)

	assert.Equal(t, "Bart", all[0].Lastname)
	assert.Equal(t, "Bertram", all[0].Name)
	assert.Equal(t, "12313", all[0].Zipcode)
	assert.Equal(t, "Wasweißich", all[0].City)
	assert.Equal(t, "blau", all[0].Color)
}

func TestLoad_StadtMitLeerzeichen(t *testing.T) {
	data := "Johnson, Johnny, 88888 made up, 3\n"
	repo, err := NewPersonRepository(writeTempCSV(t, data), testLogger())
	require.NoError(t, err)

	all, _ := repo.GetAll(context.Background())
	require.Len(t, all, 1)
	assert.Equal(t, "88888", all[0].Zipcode)
	assert.Equal(t, "made up", all[0].City)
	assert.Equal(t, "violett", all[0].Color)
}

func TestLoad_DateiNichtGefunden(t *testing.T) {
	_, err := NewPersonRepository("/nicht/vorhanden/pfad.csv", testLogger())
	require.Error(t, err)
}

func TestGetByID_Gefunden(t *testing.T) {
	data := "Müller, Hans, 67742 Lauterecken, 1\n"
	repo, _ := NewPersonRepository(writeTempCSV(t, data), testLogger())

	p, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "Hans", p.Name)
}

func TestGetByID_NichtGefunden(t *testing.T) {
	data := "Müller, Hans, 67742 Lauterecken, 1\n"
	repo, _ := NewPersonRepository(writeTempCSV(t, data), testLogger())

	_, err := repo.GetByID(context.Background(), 999)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetByColor_Treffer(t *testing.T) {
	data := "A, B, 11111 X, 1\nC, D, 22222 Y, 2\nE, F, 33333 Z, 1\n"
	repo, _ := NewPersonRepository(writeTempCSV(t, data), testLogger())

	persons, err := repo.GetByColor(context.Background(), "blau")
	require.NoError(t, err)
	assert.Len(t, persons, 2)
}

func TestGetByColor_Leer(t *testing.T) {
	data := "A, B, 11111 X, 1\n"
	repo, _ := NewPersonRepository(writeTempCSV(t, data), testLogger())

	persons, err := repo.GetByColor(context.Background(), "rot")
	require.NoError(t, err)
	// Nicht-nil Slice gewährleistet JSON-Ausgabe [] statt null.
	assert.NotNil(t, persons)
	assert.Empty(t, persons)
}

func TestAdd_ErhoehungNextID(t *testing.T) {
	data := "A, B, 11111 X, 1\n"
	repo, _ := NewPersonRepository(writeTempCSV(t, data), testLogger())

	first, err := repo.Add(context.Background(), domain.Person{
		Name: "Neu", Lastname: "Person", Zipcode: "00000", City: "Stadt", Color: "rot",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, first.ID)

	second, err := repo.Add(context.Background(), domain.Person{
		Name: "Noch", Lastname: "Einer", Zipcode: "11111", City: "Ort", Color: "blau",
	})
	require.NoError(t, err)
	assert.Equal(t, 3, second.ID)

	all, _ := repo.GetAll(context.Background())
	assert.Len(t, all, 3)
}

// TestLoad_SampleInputCSV ist ein Integrationstest gegen die echte sample-input.csv.
func TestLoad_SampleInputCSV(t *testing.T) {
	samplePath := filepath.Join("..", "..", "..", "sample-input.csv")
	if _, err := os.Stat(samplePath); os.IsNotExist(err) {
		t.Skip("sample-input.csv nicht gefunden, Test wird übersprungen")
	}

	repo, err := NewPersonRepository(samplePath, testLogger())
	require.NoError(t, err)

	all, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, all, 10)

	blau, _ := repo.GetByColor(context.Background(), "blau")
	assert.Len(t, blau, 2)

	gruen, _ := repo.GetByColor(context.Background(), "grün")
	assert.Len(t, gruen, 3)

	// Mehrzeiliger Datensatz (Bart, Bertram) muss korrekt zusammengeführt werden.
	bart, err := repo.GetByID(context.Background(), 8)
	require.NoError(t, err)
	assert.Equal(t, "Bart", bart.Lastname)
	assert.Equal(t, "Bertram", bart.Name)
	assert.Equal(t, "12313", bart.Zipcode)
	assert.Equal(t, "Wasweißich", bart.City)
	assert.Equal(t, "blau", bart.Color)
}
