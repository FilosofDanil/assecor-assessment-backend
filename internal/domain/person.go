package domain

import "errors"

var (
	ErrNotFound     = errors.New("nicht gefunden")
	ErrInvalidInput = errors.New("ungültige eingabe")
)

// ColorMap bildet Farben-IDs aus der CSV-Datei auf ihre Farbnamen ab.
var ColorMap = map[int]string{
	1: "blau",
	2: "grün",
	3: "violett",
	4: "rot",
	5: "gelb",
	6: "türkis",
	7: "weiß",
}

// ColorNameID bildet Farbnamen auf ihre jeweiligen IDs ab.
var ColorNameID = map[string]int{
	"blau":    1,
	"grün":    2,
	"violett": 3,
	"rot":     4,
	"gelb":    5,
	"türkis":  6,
	"weiß":    7,
}

// Person repräsentiert eine Person und ihrer Lieblingsfarbe.
type Person struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Lastname string `json:"lastname"`
	Zipcode  string `json:"zipcode"`
	City     string `json:"city"`
	Color    string `json:"color"`
}
