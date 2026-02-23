package csv

import (
	"bytes"
	"context"
	stdcsv "encoding/csv"
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

func tempCSV(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.csv")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func parseCSVRows(t *testing.T, data []byte) [][]string {
	t.Helper()
	r := stdcsv.NewReader(bytes.NewReader(data))
	all, err := r.ReadAll()
	require.NoError(t, err)
	require.NotEmpty(t, all, "mindestens eine Kopfzeile erwartet")
	return all[1:]
}

// ─── normalizeCSV ─────────────────────────────────────────────────────────────

func TestNormalizeCSV(t *testing.T) {
	logger := testLogger()

	tests := []struct {
		name      string
		input     string
		wantRows  int
		wantCells [][]string
	}{
		{
			name:     "normale Einträge werden korrekt aufgeteilt",
			input:    "Müller, Hans, 67742 Lauterecken, 1\nPetersen, Peter, 18439 Stralsund, 2\n",
			wantRows: 2,
			wantCells: [][]string{
				{"Müller", "Hans", "67742 Lauterecken", "1"},
				{"Petersen", "Peter", "18439 Stralsund", "2"},
			},
		},
		{
			name:     "mehrzeiliger Datensatz wird zusammengeführt",
			input:    "Bart, Bertram, \n12313 Wasweißich, 1\n",
			wantRows: 1,
			wantCells: [][]string{
				{"Bart", "Bertram", "12313 Wasweißich", "1"},
			},
		},
		{
			name:     "Stadt mit Leerzeichen bleibt als einzelnes Feld",
			input:    "Johnson, Johnny, 88888 made up, 3\n",
			wantRows: 1,
			wantCells: [][]string{
				{"Johnson", "Johnny", "88888 made up", "3"},
			},
		},
		{
			name:     "Stadt mit mehreren Leerzeichen",
			input:    "Millenium, Milly, 77777 made up too, 4\n",
			wantRows: 1,
			wantCells: [][]string{
				{"Millenium", "Milly", "77777 made up too", "4"},
			},
		},
		{
			name:     "Sonderzeichen in Stadtname",
			input:    "Andersson, Anders, 32132 Schweden - ☀, 2\n",
			wantRows: 1,
			wantCells: [][]string{
				{"Andersson", "Anders", "32132 Schweden - ☀", "2"},
			},
		},
		{
			name:     "Windows-Zeilenumbrüche werden normalisiert",
			input:    "Müller, Hans, 67742 Lauterecken, 1\r\n",
			wantRows: 1,
			wantCells: [][]string{
				{"Müller", "Hans", "67742 Lauterecken", "1"},
			},
		},
		{
			name:     "leere Eingabe erzeugt keine Datenzeilen",
			input:    "",
			wantRows: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := normalizeCSV([]byte(tt.input), logger)
			require.NoError(t, err)
			rows := parseCSVRows(t, out)
			assert.Len(t, rows, tt.wantRows)
			for i, want := range tt.wantCells {
				require.Less(t, i, len(rows))
				assert.Equal(t, want, rows[i])
			}
		})
	}
}

// ─── Bug 2: Akkumulationsschutz ─────────────────────────────────────────────

func TestNormalizeCSV_AkkumulationsschutzBug2(t *testing.T) {
	// Ein fehlerhafter 3-Feld-Datensatz gefolgt von einem korrekten.
	// Ohne den Schutz würden die 3 Felder mit den nächsten verschmelzen.
	// Mit dem Schutz wird bei > maxFieldsAccumulated verworfen.
	input := "A, B, C\nD, E, F\nG, H, I\nMüller, Hans, 67742 Lauterecken, 1\n"
	out, err := normalizeCSV([]byte(input), testLogger())
	require.NoError(t, err)

	rows := parseCSVRows(t, out)
	// Der letzte gültige 4-Feld-Datensatz muss erhalten bleiben.
	require.GreaterOrEqual(t, len(rows), 1)
	last := rows[len(rows)-1]
	assert.Equal(t, "Müller", last[0])
}

// ─── toPerson ─────────────────────────────────────────────────────────────────

func TestToPerson(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		dto     *personDTO
		want    domain.Person
		wantErr bool
	}{
		{
			name: "vollständige gültige Eingabe",
			id:   1,
			dto:  &personDTO{Lastname: "Müller", Name: "Hans", ZipCity: "67742 Lauterecken", ColorID: "1"},
			want: domain.Person{ID: 1, Name: "Hans", Lastname: "Müller", Zipcode: "67742", City: "Lauterecken", Color: "blau"},
		},
		{
			name:    "Farb-ID kein Integer",
			id:      1,
			dto:     &personDTO{Lastname: "X", Name: "Y", ZipCity: "11111 Z", ColorID: "abc"},
			wantErr: true,
		},
		{
			name:    "Farb-ID außerhalb des gültigen Bereichs",
			id:      1,
			dto:     &personDTO{Lastname: "X", Name: "Y", ZipCity: "11111 Z", ColorID: "99"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toPerson(tt.id, tt.dto)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ─── splitZipcodeCity ─────────────────────────────────────────────────────────

func TestSplitZipcodeCity(t *testing.T) {
	tests := []struct {
		input    string
		wantZip  string
		wantCity string
	}{
		{"67742 Lauterecken", "67742", "Lauterecken"},
		{"88888 made up", "88888", "made up"},
		{"77777 made up too", "77777", "made up too"},
		{"12345", "12345", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			zip, city := splitZipcodeCity(tt.input)
			assert.Equal(t, tt.wantZip, zip)
			assert.Equal(t, tt.wantCity, city)
		})
	}
}

// ─── PersonRepository – Laden ─────────────────────────────────────────────────

func TestLoad(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLen   int
		wantFirst domain.Person
	}{
		{
			name:    "normale Einträge",
			input:   "Müller, Hans, 67742 Lauterecken, 1\nPetersen, Peter, 18439 Stralsund, 2\n",
			wantLen: 2,
			wantFirst: domain.Person{
				ID: 1, Name: "Hans", Lastname: "Müller",
				Zipcode: "67742", City: "Lauterecken", Color: "blau",
			},
		},
		{
			name:    "mehrzeiliger Datensatz",
			input:   "Bart, Bertram, \n12313 Wasweißich, 1\n",
			wantLen: 1,
			wantFirst: domain.Person{
				ID: 1, Name: "Bertram", Lastname: "Bart",
				Zipcode: "12313", City: "Wasweißich", Color: "blau",
			},
		},
		{
			name:    "leere Datei",
			input:   "",
			wantLen: 0,
		},
		{
			name:    "ungültige Farb-ID übersprungen, gültige bleiben",
			input:   "A, B, 11111 X, 99\nMüller, Hans, 67742 Lauterecken, 1\n",
			wantLen: 1,
			wantFirst: domain.Person{
				ID: 2, Name: "Hans", Lastname: "Müller",
				Zipcode: "67742", City: "Lauterecken", Color: "blau",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := NewPersonRepository(tempCSV(t, tt.input), 0, testLogger())
			require.NoError(t, err)

			all, err := repo.GetAll(context.Background(), 0, 0)
			require.NoError(t, err)
			assert.Len(t, all, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantFirst, all[0])
			}
		})
	}
}

func TestLoad_DateiNichtGefunden(t *testing.T) {
	_, err := NewPersonRepository("/nicht/vorhanden/pfad.csv", 0, testLogger())
	require.Error(t, err)
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

func TestGetByID(t *testing.T) {
	const data = "Müller, Hans, 67742 Lauterecken, 1\nPetersen, Peter, 18439 Stralsund, 2\n"
	repo, err := NewPersonRepository(tempCSV(t, data), 0, testLogger())
	require.NoError(t, err)

	tests := []struct {
		name     string
		id       int
		wantName string
		wantErr  error
	}{
		{"erste Person", 1, "Hans", nil},
		{"zweite Person", 2, "Peter", nil},
		{"nicht vorhanden", 999, "", domain.ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := repo.GetByID(context.Background(), tt.id)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, p.Name)
		})
	}
}

// ─── GetByColor ───────────────────────────────────────────────────────────────

func TestGetByColor(t *testing.T) {
	const data = "A, B, 11111 X, 1\nC, D, 22222 Y, 2\nE, F, 33333 Z, 1\n"
	repo, err := NewPersonRepository(tempCSV(t, data), 0, testLogger())
	require.NoError(t, err)

	tests := []struct {
		name    string
		color   string
		wantLen int
	}{
		{"zwei Treffer für blau", "blau", 2},
		{"ein Treffer für grün", "grün", 1},
		{"kein Treffer liefert leeres Slice (nicht nil)", "rot", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persons, err := repo.GetByColor(context.Background(), tt.color, 0, 0)
			require.NoError(t, err)
			assert.NotNil(t, persons)
			assert.Len(t, persons, tt.wantLen)
		})
	}
}

// ─── Paginierung ──────────────────────────────────────────────────────────────

func TestGetAll_Paginierung(t *testing.T) {
	const data = "A, B, 11111 X, 1\nC, D, 22222 Y, 2\nE, F, 33333 Z, 3\n"
	repo, err := NewPersonRepository(tempCSV(t, data), 0, testLogger())
	require.NoError(t, err)

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
		{"offset über Gesamtzahl hinaus", 0, 99, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persons, err := repo.GetAll(context.Background(), tt.limit, tt.offset)
			require.NoError(t, err)
			assert.Len(t, persons, tt.wantLen)
		})
	}
}

// ─── Add + Kapazitätsgrenze ───────────────────────────────────────────────────

func TestAdd(t *testing.T) {
	const data = "A, B, 11111 X, 1\n"
	repo, err := NewPersonRepository(tempCSV(t, data), 0, testLogger())
	require.NoError(t, err)

	created, err := repo.Add(context.Background(), domain.Person{
		Name: "Neu", Lastname: "Person", Color: "rot",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, created.ID)

	all, _ := repo.GetAll(context.Background(), 0, 0)
	assert.Len(t, all, 2)
}

func TestAdd_KapazitaetsgrenzExploit3(t *testing.T) {
	const data = "A, B, 11111 X, 1\n"
	repo, err := NewPersonRepository(tempCSV(t, data), 2, testLogger())
	require.NoError(t, err)

	_, err = repo.Add(context.Background(), domain.Person{Name: "N", Lastname: "P", Color: "rot"})
	require.NoError(t, err)

	_, err = repo.Add(context.Background(), domain.Person{Name: "Z", Lastname: "Q", Color: "blau"})
	require.ErrorIs(t, err, domain.ErrCapacityReached)
}

func TestAdd_KeineIDKollisionNachUebersprungeneEintraege(t *testing.T) {
	const data = "A, B, 11111 X, 99\nMüller, Hans, 67742 Lauterecken, 1\n"
	repo, err := NewPersonRepository(tempCSV(t, data), 0, testLogger())
	require.NoError(t, err)

	created, err := repo.Add(context.Background(), domain.Person{
		Name: "Neu", Lastname: "Person", Color: "rot",
	})
	require.NoError(t, err)
	assert.Equal(t, 3, created.ID)
}

// ─── Integrationstest gegen echte sample-input.csv ────────────────────────────

func TestLoad_SampleInputCSV(t *testing.T) {
	samplePath := filepath.Join("..", "..", "..", "sample-input.csv")
	if _, err := os.Stat(samplePath); os.IsNotExist(err) {
		t.Skip("sample-input.csv nicht gefunden")
	}

	repo, err := NewPersonRepository(samplePath, 0, testLogger())
	require.NoError(t, err)

	all, err := repo.GetAll(context.Background(), 0, 0)
	require.NoError(t, err)
	assert.Len(t, all, 10)

	blau, _ := repo.GetByColor(context.Background(), "blau", 0, 0)
	assert.Len(t, blau, 2)

	gruen, _ := repo.GetByColor(context.Background(), "grün", 0, 0)
	assert.Len(t, gruen, 3)

	bart, err := repo.GetByID(context.Background(), 8)
	require.NoError(t, err)
	assert.Equal(t, "Bart", bart.Lastname)
	assert.Equal(t, "Bertram", bart.Name)
	assert.Equal(t, "12313", bart.Zipcode)
	assert.Equal(t, "Wasweißich", bart.City)
	assert.Equal(t, "blau", bart.Color)
}
