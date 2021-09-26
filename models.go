package main

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/trilobio/ar3"
	"github.com/trilobio/kinematics"
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
	Wells      []Well  `json:"wells"`
}

func GetLabwares(tx *sqlx.Tx) ([]Labware, error) {
	var labwares []Labware
	err := tx.Select(&labwares, "SELECT * FROM labware")
	if err != nil {
		return labwares, err
	}
	for _, labware := range labwares {
		var wells []Well
		err = tx.Select(&wells, "SELECT address, depth, diameter, x, y, z FROM well WHERE labware = ?", labware.Name)
		if err != nil {
			return labwares, err
		}
		labware.Wells = wells
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

                               Deck

******************************************************************************/

type Deck struct {
	Name       string     `json:"name" db:"name"`
	Calibrated bool       `json:"calibrated" db:"calibrated"`
	X          float64    `json:"x" db:"x"`
	Y          float64    `json:"y" db:"y"`
	Z          float64    `json:"z" db:"z"`
	Locations  []Location `json:"locations"`
}

type Location struct {
	Name string  `json:"name" db:"name"`
	X    float64 `json:"x" db:"x"`
	Y    float64 `json:"y" db:"y"`
	Z    float64 `json:"z" db:"z"`
}

func GetDecks(tx *sqlx.Tx) ([]Deck, error) {
	var decks []Deck
	err := tx.Select(&decks, "SELECT * FROM deck")
	if err != nil {
		return decks, err
	}
	for _, deck := range decks {
		var locations []Location
		err = tx.Select(&locations, "SELECT name, x, y, z FROM location WHERE deck = ?", deck.Name)
		if err != nil {
			return decks, err
		}
		deck.Locations = locations
	}
	return decks, nil

}

func GetDeck(tx *sqlx.Tx, name string) (Deck, error) {
	var deck Deck
	err := tx.Get(&deck, "SELECT * FROM deck WHERE name = ?", name)
	if err != nil {
		return deck, err
	}
	var locations []Location
	err = tx.Select(&locations, "SELECT name, x, y, z FROM location WHERE deck = ?", name)
	if err != nil {
		return deck, err
	}
	deck.Locations = locations
	return deck, nil

}

func CreateDeck(tx *sqlx.Tx, deck Deck) error {
	_, err := tx.Exec("INSERT INTO deck(name) VALUES (?)", deck.Name)
	if err != nil {
		return err
	}
	for _, location := range deck.Locations {
		_, err := tx.Exec("INSERT INTO location(deck, name, x, y, z) VALUES (?, ?, ?, ?, ?)", deck.Name, location.Name, location.X, location.Y, location.Z)
		if err != nil {
			return err
		}
	}
	return nil
}

func SetDeckCalibration(tx *sqlx.Tx, name string, x float64, y float64, z float64) error {
	_, err := tx.Exec("UPDATE deck SET calibrated = ?, x = ?, y = ?, z = ? WHERE name = ?", true, x, y, z, name)
	if err != nil {
		return err
	}
	return nil
}

func DeleteDeck(tx *sqlx.Tx, name string) error {
	_, err := tx.Exec("DELETE FROM deck WHERE name = ?", name)
	if err != nil {
		return err
	}
	return nil
}

/******************************************************************************

                                Protocol

******************************************************************************/

type CommandXyz struct {
	Command string  `json:"command"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Z       float64 `json:"z"`
}

type CommandMove struct {
	Command         string  `json:"command"`
	Deck            string  `json:"name"`
	Location        string  `json:"location"`
	LabwareName     string  `json:"labware_name"`
	Address         string  `json:"address"`
	DepthFromBottom float64 `json:"depth_from_bottom"`
}

var defaultQuaternion kinematics.Quaternion = kinematics.Quaternion{W: 0.8063737663657652, X: -0.575080903948282, Y: -0.13494466363153904, Z: 0.02886590702694046}

func ExecuteProtocol(arm ar3.Arm, protocol []byte) error {
	err := arm.MoveJointRadians(5, 10, 10, 10, 10, 1, 1, 1, 1, 1, 1, 0)
	if err != nil {
		return err
	}
	var steps []json.RawMessage
	if err := json.Unmarshal(protocol, &steps); err != nil {
		return err
	}
	for i, step := range steps {
		stepMap := make(map[string]interface{})
		err := json.Unmarshal(step, &stepMap)
		if err != nil {
			return err
		}
		if _, ok := stepMap["command"]; !ok {
			return fmt.Errorf("command not found in step %d of command", i)
		}

		// Run each different possible command
		command := stepMap["command"]
		switch command {
		case "movexyz":
			var movexyz CommandXyz
			err := json.Unmarshal(step, &movexyz)
			if err != nil {
				return err
			}
			// Move arm to XYZ position
			err = arm.Move(25, 10, 10, 10, 10, kinematics.Pose{Position: kinematics.Position{X: movexyz.X, Y: movexyz.Y, Z: movexyz.Z}, Rotation: defaultQuaternion})
			if err != nil {
				return err
			}
		case "move":
			var move CommandMove
			err := json.Unmarshal(step, &move)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

/******************************************************************************

                               Database

******************************************************************************/

func CreateDatabase(db *sqlx.DB) error {
	_, err := db.Exec(Schema)
	if err != nil {
		return err
	}
	return nil
}

const Schema = `
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

-- Add labware and deck
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

CREATE TABLE IF NOT EXISTS deck (
	name TEXT PRIMARY KEY,
	calibrated BOOLEAN DEFAULT false,
        x REAL NOT NULL DEFAULT 0,
        y REAL NOT NULL DEFAULT 0,
        z REAL NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS location (
	name TEXT PRIMARY KEY,
	deck TEXT NOT NULL REFERENCES deck(name) ON DELETE CASCADE,
	x REAL NOT NULL,
        y REAL NOT NULL,
        z REAL NOT NULL
);

-- Add activity log
CREATE TABLE IF NOT EXISTS activity_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    start INTEGER NOT NULL,
    end INTEGER,
    program TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('RUNNING', 'FAILED', 'COMPLETED')),
    status_message TEXT
);

-- Add device lock
CREATE TABLE IF NOT EXISTS lock (
    id INT PRIMARY KEY,
    active BOOL NOT NULL DEFAULT false,
    locked_by INTEGER REFERENCES activity_log(id)
);
INSERT OR IGNORE INTO lock(id) VALUES (1);
UPDATE lock SET active = 0, locked_by = NULL WHERE id=1;
`
