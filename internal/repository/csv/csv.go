package csv

import (
	"context"
	stdcsv "encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"

	"assecor-assessment-backend/internal/domain"
)

// PersonRepository implementiert repository. PersonRepository und hält alle Personen im Arbeitsspeicher.
type PersonRepository struct {
	mu      sync.RWMutex
	persons []domain.Person
	nextID  int
	logger  *zap.Logger
}

// NewPersonRepository legt ein neues CSV-Repository an und lädt alle Datensätze aus filePath beim Aufruf sofort in den Speicher.
func NewPersonRepository(filePath string, logger *zap.Logger) (*PersonRepository, error) {
	r := &PersonRepository{logger: logger}
	if err := r.load(filePath); err != nil {
		return nil, fmt.Errorf("csv-repository: %w", err)
	}
	return r, nil
}

// load liest die CSV-Datei und befüllt r.persons.
func (r *PersonRepository) load(filePath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("datei öffnen %s: %w", filePath, err)
	}
	defer file.Close()

	reader := stdcsv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	var (
		accumulated []string
		personIndex = 1
	)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("csv lesen: %w", err)
		}

		for _, field := range record {
			if trimmed := strings.TrimSpace(field); trimmed != "" {
				accumulated = append(accumulated, trimmed)
			}
		}

		if len(accumulated) >= 4 {
			person, err := parseRecord(personIndex, accumulated)
			if err != nil {
				r.logger.Warn("ungültiger Datensatz wird übersprungen",
					zap.Int("datensatz", personIndex),
					zap.Error(err),
				)
			} else {
				r.persons = append(r.persons, person)
				personIndex++
			}
			accumulated = nil
		}
	}

	if len(accumulated) > 0 {
		r.logger.Warn("unvollständiger Datensatz am Dateiende",
			zap.Strings("felder", accumulated),
		)
	}

	// Nächste freie ID direkt nach dem letzten geladenen Datensatz setzen.
	r.nextID = len(r.persons) + 1

	r.logger.Info("personen aus CSV geladen",
		zap.Int("anzahl", len(r.persons)),
		zap.String("datei", filePath),
	)
	return nil
}

// parseRecord wandelt bereits getrimmte CSV-Felder in eine Person um.
func parseRecord(id int, fields []string) (domain.Person, error) {
	n := len(fields)
	if n < 4 {
		return domain.Person{}, fmt.Errorf("erwartet >= 4 Felder, erhalten %d", n)
	}

	lastname := fields[0]
	name := fields[1]

	colorStr := fields[n-1]
	colorID, err := strconv.Atoi(colorStr)
	if err != nil {
		return domain.Person{}, fmt.Errorf("ungültige Farb-ID %q: %w", colorStr, err)
	}
	colorName, ok := domain.ColorMap[colorID]
	if !ok {
		return domain.Person{}, fmt.Errorf("unbekannte Farb-ID %d", colorID)
	}

	zipcodeCity := strings.Join(fields[2:n-1], " ")
	zipcode, city := splitZipcodeCity(zipcodeCity)

	return domain.Person{
		ID:       id,
		Name:     name,
		Lastname: lastname,
		Zipcode:  zipcode,
		City:     city,
		Color:    colorName,
	}, nil
}

// splitZipcodeCity trennt eine Zeichenkette der Form "PLZ Stadt" am ersten Leerzeichen in Postleitzahl und Stadtname auf.
func splitZipcodeCity(s string) (string, string) {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return s, ""
}

// GetAll gibt alle Personen zurück.
func (r *PersonRepository) GetAll(_ context.Context) ([]domain.Person, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.Person, len(r.persons))
	copy(out, r.persons)
	return out, nil
}

// GetByID sucht eine Person anhand ihrer positionsbasierten ID.
func (r *PersonRepository) GetByID(_ context.Context, id int) (domain.Person, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.persons {
		if p.ID == id {
			return p, nil
		}
	}
	return domain.Person{}, fmt.Errorf("person mit id %d: %w", id, domain.ErrNotFound)
}

// GetByColor gibt alle Personen zurück, deren Lieblingsfarbe mit color übereinstimmt.
func (r *PersonRepository) GetByColor(_ context.Context, color string) ([]domain.Person, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.Person, 0)
	for _, p := range r.persons {
		if p.Color == color {
			out = append(out, p)
		}
	}
	return out, nil
}

// Add fügt eine neue Person hinzu und vergibt eine eindeutige, monoton steigende ID über den nextID-Zähler.
func (r *PersonRepository) Add(_ context.Context, person domain.Person) (domain.Person, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	person.ID = r.nextID
	r.nextID++
	r.persons = append(r.persons, person)
	return person, nil
}
