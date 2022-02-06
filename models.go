package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"io/fs"
	"io/ioutil"
)

/******************************************************************************

                                Labware

******************************************************************************/

type Well struct {
	Address  string  `json:"address" db:"address"`
	Depth    float64 `json:"depth" db:"depth"`
	Diameter float64 `json:"diameter" db:"diameter"`
	X        float64 `json:"x" db:"x"`
	Y        float64 `json:"y" db:"y"`
	Z        float64 `json:"z" db:"z"`
}

type Labware struct {
	Name string `json:"name" db:"name"`
	//XDimension float64 `json:"xDimension"` // Microplate: 127.76
	//YDimension float64 `json:"yDimension"` // Microplate: 85.48
	ZDimension float64 `json:"zDimension" db:"zdimension"`
	Wells      []Well  `json:"wells,omitempty"`
}

func GetLabwares(tx *sqlx.Tx) ([]Labware, error) {
	var labwares []Labware
	err := tx.Select(&labwares, "SELECT * FROM labware")
	if err != nil {
		return labwares, err
	}
	for i := range labwares {
		var wells []Well
		err = tx.Select(&wells, "SELECT address, depth, diameter, x, y, z FROM well WHERE labware = ?", labwares[i].Name)
		if err != nil {
			return labwares, err
		}
		labwares[i].Wells = wells
	}
	return labwares, nil

}

func GetLabware(tx *sqlx.Tx, name string) (Labware, error) {
	var labware Labware
	err := tx.Get(&labware, "SELECT * FROM labware WHERE name = ?", name)
	if err != nil {
		return labware, err
	}
	var wells []Well
	err = tx.Select(&wells, "SELECT address, depth, diameter, x, y, z FROM well WHERE labware = ?", name)
	if err != nil {
		return labware, err
	}
	labware.Wells = wells
	return labware, nil

}

func CreateLabware(tx *sqlx.Tx, labware Labware) error {
	_, err := tx.Exec("INSERT INTO labware(name, zdimension) VALUES (?, ?)", labware.Name, labware.ZDimension)
	if err != nil {
		return err
	}
	for _, well := range labware.Wells {
		_, err := tx.Exec("INSERT INTO well(labware, address, depth, diameter, x, y, z) VALUES (?, ?, ?, ?, ?, ?, ?)", labware.Name, well.Address, well.Depth, well.Diameter, well.X, well.Y, well.Z)
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteLabware(tx *sqlx.Tx, name string) error {
	_, err := tx.Exec("DELETE FROM labware WHERE name = ?", name)
	if err != nil {
		return err
	}
	return nil
}

/******************************************************************************

                                Protocol

******************************************************************************/

type CommandInput interface {
	Command() string
}

type CommandXyz struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func (c CommandXyz) Command() string { return "movexyz" }

type CommandMove struct {
	Deck            string  `json:"name"`
	Location        string  `json:"location"`
	LabwareName     string  `json:"labware_name"`
	Address         string  `json:"address"`
	DepthFromBottom float64 `json:"depth_from_bottom"`
}

func (c CommandMove) Command() string { return "move" }

type Command struct {
	Command  string
	WaitTime int // Milliseconds
}

func ExecuteProtocol(db *sqlx.DB, protocol []CommandInput) error {
	for _, step := range protocol {
		// Run each different possible command
		command := step.Command()
		switch command {
		case "movexyz":
			var movexyz CommandXyz
			movexyz = step.(CommandXyz)
			_ = movexyz

			// Move arm to XYZ position
		default:
			return fmt.Errorf("Command not found. Only valid commands are `move, wait, movexyz`, got: %s", command)
		}
	}
	return nil
}

/******************************************************************************

                                Defaults

******************************************************************************/

type OpentronsParameters struct {
	LoadName string `json:"loadName"`
}

type OpentronsDimensions struct {
	ZDimension float64 `json:"zDimension"`
}

type OpentronsLabware struct {
	Dimensions OpentronsDimensions `json:"dimensions"`
	Parameters OpentronsParameters `json:"parameters"`
	Wells      map[string]Well     `json:"wells"`
}

func opentronsLabwareToLabware(ol OpentronsLabware) Labware {
	var wells []Well
	for address, well := range ol.Wells {
		newWell := well
		newWell.Address = address
		wells = append(wells, newWell)
	}
	return Labware{Name: ol.Parameters.LoadName, ZDimension: ol.Dimensions.ZDimension, Wells: wells}
}

//go:embed data/**/*
var content embed.FS

func defaultLabware() ([]Labware, error) {
	var labwares []Labware
	matches, err := fs.Glob(content, "data/**/*")
	if err != nil {
		return labwares, err
	}
	for _, match := range matches {
		file, err := content.Open(match)
		if err != nil {
			return labwares, err
		}
		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			return labwares, err
		}

		var opentronsLabware OpentronsLabware
		err = json.Unmarshal(fileBytes, &opentronsLabware)
		if err != nil {
			return labwares, err
		}

		labwares = append(labwares, opentronsLabwareToLabware(opentronsLabware))
	}
	return labwares, nil
}

/******************************************************************************

                               Database

******************************************************************************/

func CreateDatabase(db *sqlx.DB) error {
	_, err := db.Exec(Schema)
	if err != nil {
		return err
	}
	// Add in default labwares
	defaultLabwares, err := defaultLabware()
	if err != nil {
		return err
	}
	tx := db.MustBegin()
	for _, labware := range defaultLabwares {
		_, err = GetLabware(tx, labware.Name)
		if err != nil {
			err = CreateLabware(tx, labware)
			if err != nil {
				return err
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

const Schema = `
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

-- Add labware
CREATE TABLE IF NOT EXISTS labware (
	name TEXT PRIMARY KEY,
	zdimension REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS well (
	labware TEXT NOT NULL REFERENCES labware(name) ON DELETE CASCADE,
	address TEXT NOT NULL,
	depth REAL NOT NULL,
	diameter REAL NOT NULL,
	x REAL NOT NULL,
	y REAL NOT NULL,
	z REAL NOT NULL
);
CREATE INDEX idx_labware_well ON well(labware);

CREATE TABLE IF NOT EXISTS location (
	name TEXT,
	deck TEXT NOT NULL REFERENCES deck(name) ON DELETE CASCADE,
	x REAL NOT NULL,
        y REAL NOT NULL,
        z REAL NOT NULL
);

-- Add device lock
CREATE TABLE IF NOT EXISTS lock (
    id INT PRIMARY KEY,
    active BOOL NOT NULL DEFAULT false
);

INSERT OR IGNORE INTO lock(id) VALUES (1);
UPDATE lock SET active = 0 WHERE id=1;
`
