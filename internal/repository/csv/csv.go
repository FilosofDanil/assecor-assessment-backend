package csv

import (
	"bytes"
	"context"
	stdcsv "encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/gocarina/gocsv"
	"go.uber.org/zap"

	"assecor-assessment-backend/internal/domain"
)

// personDTO ist das Zwischen-DTO, das gocsv aus der normalisierten CSV befüllt.
type personDTO struct {
	Lastname string `csv:"lastname"`
	Name     string `csv:"name"`
	ZipCity  string `csv:"zipcity"`
	ColorID  string `csv:"colorid"`
}

// PersonRepository hält alle Personen im Arbeitsspeicher und implementiert repository.PersonRepository.
type PersonRepository struct {
	mu         sync.RWMutex
	persons    []domain.Person
	nextID     int
	maxPersons int
	logger     *zap.Logger
}

// NewPersonRepository legt ein neues PersonRepository
func NewPersonRepository(filePath string, maxPersons int, logger *zap.Logger) (*PersonRepository, error) {
	r := &PersonRepository{maxPersons: maxPersons, logger: logger}
	if err := r.load(filePath); err != nil {
		return nil, fmt.Errorf("csv-repository: %w", err)
	}
	return r, nil
}

// load liest die CSV-Datei und befüllt r.persons über gocsv.
func (r *PersonRepository) load(filePath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("datei lesen %s: %w", filePath, err)
	}

	normalized, err := normalizeCSV(data, r.logger)
	if err != nil {
		return fmt.Errorf("csv normalisieren: %w", err)
	}

	var dtos []*personDTO
	if err := gocsv.UnmarshalBytes(normalized, &dtos); err != nil {
		return fmt.Errorf("csv parsen: %w", err)
	}

	r.persons = make([]domain.Person, 0, len(dtos))
	for i, dto := range dtos {
		person, err := toPerson(i+1, dto)
		if err != nil {
			r.logger.Warn("ungültiger datensatz wird übersprungen",
				zap.Int("datensatz", i+1), zap.Error(err))
			continue
		}
		r.persons = append(r.persons, person)
	}

	r.nextID = len(dtos) + 1

	r.logger.Info("personen aus CSV geladen",
		zap.Int("anzahl", len(r.persons)), zap.String("datei", filePath))
	return nil
}

// normalizeCSV verarbeitet das mehrzeilige Datensatzformat der Quell-CSV.
func normalizeCSV(data []byte, logger *zap.Logger) ([]byte, error) {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

	records := make([][]string, 0, len(lines)+1)
	records = append(records, []string{"lastname", "name", "zipcity", "colorid"})

	var accumulated []string
	for _, line := range lines {
		rawParts := strings.Split(line, ",")
		nonEmpty := countNonEmpty(rawParts)
		if len(accumulated) > 0 && nonEmpty >= 4 {
			logger.Warn("fehlerhafter vorgänger-datensatz verworfen",
				zap.Strings("felder", accumulated))
			accumulated = nil
		}

		for _, field := range rawParts {
			if trimmed := strings.TrimSpace(field); trimmed != "" {
				accumulated = append(accumulated, trimmed)
			}
		}

		if len(accumulated) >= 4 {
			n := len(accumulated)
			records = append(records, []string{
				accumulated[0],
				accumulated[1],
				strings.Join(accumulated[2:n-1], " "),
				accumulated[n-1],
			})
			accumulated = nil
		}
	}

	if len(accumulated) > 0 {
		logger.Warn("unvollständiger datensatz am dateiende wird verworfen",
			zap.Strings("felder", accumulated))
	}

	var buf bytes.Buffer
	w := stdcsv.NewWriter(&buf)
	if err := w.WriteAll(records); err != nil {
		return nil, fmt.Errorf("csv schreiben: %w", err)
	}
	return buf.Bytes(), nil
}

// toPerson wandelt ein personDTO in eine domain.Person um.
func toPerson(id int, dto *personDTO) (domain.Person, error) {
	colorID, err := strconv.Atoi(strings.TrimSpace(dto.ColorID))
	if err != nil {
		return domain.Person{}, fmt.Errorf("ungültige farb-id %q: %w", dto.ColorID, err)
	}
	colorName, ok := domain.ColorMap[colorID]
	if !ok {
		return domain.Person{}, fmt.Errorf("unbekannte farb-id %d", colorID)
	}
	zipcode, city := splitZipcodeCity(dto.ZipCity)
	return domain.Person{
		ID: id, Name: dto.Name, Lastname: dto.Lastname,
		Zipcode: zipcode, City: city, Color: colorName,
	}, nil
}

func countNonEmpty(parts []string) int {
	n := 0
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			n++
		}
	}
	return n
}

// splitZipcodeCity trennt "PLZ Stadt" am ersten Leerzeichen.
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

// GetByColor gibt alle Personen mit passender Lieblingsfarbe zurück.
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

// Add fügt eine neue Person hinzu.
func (r *PersonRepository) Add(_ context.Context, person domain.Person) (domain.Person, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.maxPersons > 0 && len(r.persons) >= r.maxPersons {
		return domain.Person{}, fmt.Errorf("max %d personen: %w", r.maxPersons, domain.ErrCapacityReached)
	}

	person.ID = r.nextID
	r.nextID++
	r.persons = append(r.persons, person)
	return person, nil
}
