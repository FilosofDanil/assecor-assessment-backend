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
	return all[1:] // Kopfzeile überspringen
}

// ─── normalizeCSV ─────────────────────────────────────────────────────────────

func TestNormalizeCSV(t *testing.T) {
	logger := testLogger()

	tests := []struct {
		name      string
		input     string
		wantRows  int
		wantCells [][]string // [Zeile][Spalte] – nur gesetzt wenn wantRows > 0
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
			name:     "Sonderzeichen in Stadtname werden korrekt maskiert",
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
				require.Less(t, i, len(rows), "Zeile %d fehlt in der Ausgabe", i)
				assert.Equal(t, want, rows[i])
			}
		})
	}
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
			name: "Stadt mit Leerzeichen",
			id:   2,
			dto:  &personDTO{Lastname: "Johnson", Name: "Johnny", ZipCity: "88888 made up", ColorID: "3"},
			want: domain.Person{ID: 2, Name: "Johnny", Lastname: "Johnson", Zipcode: "88888", City: "made up", Color: "violett"},
		},
		{
			name: "alle sieben Farben korrekt abgebildet – Farbe 7 (weiß)",
			id:   7,
			dto:  &personDTO{Lastname: "A", Name: "B", ZipCity: "00000 C", ColorID: "7"},
			want: domain.Person{ID: 7, Name: "B", Lastname: "A", Zipcode: "00000", City: "C", Color: "weiß"},
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
		{
			name:    "Farb-ID 0 ist nicht gültig",
			id:      1,
			dto:     &personDTO{Lastname: "X", Name: "Y", ZipCity: "11111 Z", ColorID: "0"},
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
		{"32132 Schweden - ☀", "32132", "Schweden - ☀"},
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
			name:    "normale Einträge werden vollständig geladen",
			input:   "Müller, Hans, 67742 Lauterecken, 1\nPetersen, Peter, 18439 Stralsund, 2\n",
			wantLen: 2,
			wantFirst: domain.Person{
				ID: 1, Name: "Hans", Lastname: "Müller",
				Zipcode: "67742", City: "Lauterecken", Color: "blau",
			},
		},
		{
			name:    "mehrzeiliger Datensatz wird korrekt zusammengeführt",
			input:   "Bart, Bertram, \n12313 Wasweißich, 1\n",
			wantLen: 1,
			wantFirst: domain.Person{
				ID: 1, Name: "Bertram", Lastname: "Bart",
				Zipcode: "12313", City: "Wasweißich", Color: "blau",
			},
		},
		{
			name:    "Stadt mit Leerzeichen bleibt erhalten",
			input:   "Johnson, Johnny, 88888 made up, 3\n",
			wantLen: 1,
			wantFirst: domain.Person{
				ID: 1, Name: "Johnny", Lastname: "Johnson",
				Zipcode: "88888", City: "made up", Color: "violett",
			},
		},
		{
			name:    "leere Datei erzeugt leeres Repository",
			input:   "",
			wantLen: 0,
		},
		{
			name:    "ungültige Farb-ID wird übersprungen, gültige Datensätze bleiben erhalten",
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
			repo, err := NewPersonRepository(tempCSV(t, tt.input), testLogger())
			require.NoError(t, err)

			all, err := repo.GetAll(context.Background())
			require.NoError(t, err)
			assert.Len(t, all, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantFirst, all[0])
			}
		})
	}
}

func TestLoad_DateiNichtGefunden(t *testing.T) {
	_, err := NewPersonRepository("/nicht/vorhanden/pfad.csv", testLogger())
	require.Error(t, err)
}

// ─── PersonRepository – GetByID ───────────────────────────────────────────────

func TestGetByID(t *testing.T) {
	const data = "Müller, Hans, 67742 Lauterecken, 1\nPetersen, Peter, 18439 Stralsund, 2\n"

	repo, err := NewPersonRepository(tempCSV(t, data), testLogger())
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

// ─── PersonRepository – GetByColor ────────────────────────────────────────────

func TestGetByColor(t *testing.T) {
	const data = "A, B, 11111 X, 1\nC, D, 22222 Y, 2\nE, F, 33333 Z, 1\n"

	repo, err := NewPersonRepository(tempCSV(t, data), testLogger())
	require.NoError(t, err)

	tests := []struct {
		name    string
		color   string
		wantLen int
	}{
		{"zwei Treffer für blau", "blau", 2},
		{"ein Treffer für grün", "grün", 1},
		{"kein Treffer für rot liefert leeres Slice (nicht nil)", "rot", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			persons, err := repo.GetByColor(context.Background(), tt.color)
			require.NoError(t, err)
			assert.NotNil(t, persons, "Slice darf nicht nil sein – JSON muss [] statt null ausgeben")
			assert.Len(t, persons, tt.wantLen)
		})
	}
}

// ─── PersonRepository – Add ───────────────────────────────────────────────────

func TestAdd(t *testing.T) {
	const data = "A, B, 11111 X, 1\n"

	repo, err := NewPersonRepository(tempCSV(t, data), testLogger())
	require.NoError(t, err)

	additions := []struct {
		person domain.Person
		wantID int
	}{
		{domain.Person{Name: "Neu", Lastname: "Person", Zipcode: "00000", City: "Stadt", Color: "rot"}, 2},
		{domain.Person{Name: "Noch", Lastname: "Einer", Zipcode: "11111", City: "Ort", Color: "blau"}, 3},
	}

	for _, a := range additions {
		created, err := repo.Add(context.Background(), a.person)
		require.NoError(t, err)
		assert.Equal(t, a.wantID, created.ID)
	}

	all, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

// TestAdd_KeineIDKollisionNachUebersprungeneEintraege stellt sicher, dass nextID
// korrekt gesetzt wird, wenn beim Laden Datensätze übersprungen wurden.
func TestAdd_KeineIDKollisionNachUebersprungeneEintraege(t *testing.T) {
	// Datensatz 1 ist ungültig (Farb-ID 99), Datensatz 2 erhält ID 2.
	// nextID muss nach allen rohen Positionen stehen (also 3), damit Add keine
	// ID vergibt, die bereits einem geladenen Datensatz gehört.
	const data = "A, B, 11111 X, 99\nMüller, Hans, 67742 Lauterecken, 1\n"

	repo, err := NewPersonRepository(tempCSV(t, data), testLogger())
	require.NoError(t, err)

	created, err := repo.Add(context.Background(), domain.Person{
		Name: "Neu", Lastname: "Person", Zipcode: "00000", City: "Stadt", Color: "rot",
	})
	require.NoError(t, err)

	// ID 3 wird erwartet (nicht 2, was bereits von "Müller, Hans" belegt ist).
	assert.Equal(t, 3, created.ID)
}

// ─── Integrationstest gegen echte sample-input.csv ────────────────────────────

func TestLoad_SampleInputCSV(t *testing.T) {
	samplePath := filepath.Join("..", "..", "..", "sample-input.csv")
	if _, err := os.Stat(samplePath); os.IsNotExist(err) {
		t.Skip("sample-input.csv nicht gefunden, Test wird übersprungen")
	}

	repo, err := NewPersonRepository(samplePath, testLogger())
	require.NoError(t, err)

	tests := []struct {
		name  string
		check func(t *testing.T)
	}{
		{
			"gesamt 10 Personen geladen",
			func(t *testing.T) {
				all, err := repo.GetAll(context.Background())
				require.NoError(t, err)
				assert.Len(t, all, 10)
			},
		},
		{
			"2 Personen mit Lieblingsfarbe blau",
			func(t *testing.T) {
				persons, err := repo.GetByColor(context.Background(), "blau")
				require.NoError(t, err)
				assert.Len(t, persons, 2)
			},
		},
		{
			"3 Personen mit Lieblingsfarbe grün",
			func(t *testing.T) {
				persons, err := repo.GetByColor(context.Background(), "grün")
				require.NoError(t, err)
				assert.Len(t, persons, 3)
			},
		},
		{
			"mehrzeiliger Datensatz Bart/Bertram hat ID 8 und korrekte Felder",
			func(t *testing.T) {
				p, err := repo.GetByID(context.Background(), 8)
				require.NoError(t, err)
				assert.Equal(t, "Bart", p.Lastname)
				assert.Equal(t, "Bertram", p.Name)
				assert.Equal(t, "12313", p.Zipcode)
				assert.Equal(t, "Wasweißich", p.City)
				assert.Equal(t, "blau", p.Color)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.check)
	}
}
