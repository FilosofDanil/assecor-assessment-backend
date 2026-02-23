package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"
	_ "modernc.org/sqlite"

	"assecor-assessment-backend/internal/domain"
)

// PersonRepository implementiert repository.PersonRepository
type PersonRepository struct {
	db         *sql.DB
	maxPersons int
	logger     *zap.Logger
}

// NewPersonRepository öffnet die SQLite-Datenbank und gibt ein einsatzbereites Repository zurück.
func NewPersonRepository(dsn string, maxPersons int, logger *zap.Logger) (*PersonRepository, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite öffnen: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS persons (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			name     TEXT NOT NULL,
			lastname TEXT NOT NULL,
			zipcode  TEXT NOT NULL DEFAULT '',
			city     TEXT NOT NULL DEFAULT '',
			color    TEXT NOT NULL
		)
	`); err != nil {
		return nil, fmt.Errorf("tabelle erstellen: %w", err)
	}

	logger.Info("sqlite-repository initialisiert", zap.String("dsn", dsn))
	return &PersonRepository{db: db, maxPersons: maxPersons, logger: logger}, nil
}

// Close schließt die zugrunde liegende Datenbankverbindung.
func (r *PersonRepository) Close() error {
	return r.db.Close()
}

// GetAll gibt alle Personen zurück, optional paginiert.
func (r *PersonRepository) GetAll(ctx context.Context, limit, offset int) ([]domain.Person, error) {
	query := "SELECT id, name, lastname, zipcode, city, color FROM persons ORDER BY id"
	args := make([]any, 0, 2)
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, clampOffset(offset))
	} else if offset > 0 {
		// LIMIT -1 in SQLite = kein Limit, aber OFFSET wird angewendet.
		query += " LIMIT -1 OFFSET ?"
		args = append(args, clampOffset(offset))
	}
	return r.queryPersons(ctx, query, args...)
}

// GetByID sucht eine Person anhand ihrer ID.
func (r *PersonRepository) GetByID(ctx context.Context, id int) (domain.Person, error) {
	var p domain.Person
	err := r.db.QueryRowContext(ctx,
		"SELECT id, name, lastname, zipcode, city, color FROM persons WHERE id = ?", id,
	).Scan(&p.ID, &p.Name, &p.Lastname, &p.Zipcode, &p.City, &p.Color)
	if err == sql.ErrNoRows {
		return domain.Person{}, fmt.Errorf("person mit id %d: %w", id, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Person{}, fmt.Errorf("abfrage person id %d: %w", id, err)
	}
	return p, nil
}

// GetByColor gibt alle Personen mit passender Lieblingsfarbe zurück.
func (r *PersonRepository) GetByColor(ctx context.Context, color string, limit, offset int) ([]domain.Person, error) {
	query := "SELECT id, name, lastname, zipcode, city, color FROM persons WHERE color = ? ORDER BY id"
	args := []any{color}
	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, clampOffset(offset))
	} else if offset > 0 {
		query += " LIMIT -1 OFFSET ?"
		args = append(args, clampOffset(offset))
	}
	return r.queryPersons(ctx, query, args...)
}

// Add fügt eine neue Person hinzu und prüft die Kapazitätsgrenze.
func (r *PersonRepository) Add(ctx context.Context, person domain.Person) (domain.Person, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Person{}, fmt.Errorf("transaktion starten: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if r.maxPersons > 0 {
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM persons").Scan(&count); err != nil {
			return domain.Person{}, fmt.Errorf("anzahl abfragen: %w", err)
		}
		if count >= r.maxPersons {
			return domain.Person{}, fmt.Errorf("max %d personen: %w", r.maxPersons, domain.ErrCapacityReached)
		}
	}

	res, err := tx.ExecContext(ctx,
		"INSERT INTO persons (name, lastname, zipcode, city, color) VALUES (?, ?, ?, ?, ?)",
		person.Name, person.Lastname, person.Zipcode, person.City, person.Color,
	)
	if err != nil {
		return domain.Person{}, fmt.Errorf("person einfügen: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return domain.Person{}, fmt.Errorf("letzte id: %w", err)
	}
	person.ID = int(id)

	if err := tx.Commit(); err != nil {
		return domain.Person{}, fmt.Errorf("commit: %w", err)
	}
	return person, nil
}

// queryPersons führt eine Abfrage aus und sammelt die Zeilen als Personen.
func (r *PersonRepository) queryPersons(ctx context.Context, query string, args ...any) ([]domain.Person, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("abfrage: %w", err)
	}
	defer rows.Close()

	out := make([]domain.Person, 0)
	for rows.Next() {
		var p domain.Person
		if err := rows.Scan(&p.ID, &p.Name, &p.Lastname, &p.Zipcode, &p.City, &p.Color); err != nil {
			return nil, fmt.Errorf("zeile lesen: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func clampOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
