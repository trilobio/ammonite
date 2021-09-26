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
	for i, labware := range labwares {
		var wells []Well
		err = tx.Select(&wells, "SELECT address, depth, diameter, x, y, z FROM well WHERE labware = ?", labware.Name)
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

                               Deck

******************************************************************************/

type InputDeck struct {
	Name      string     `json:"name" db:"name"`
	Locations []Location `json:"locations"`
}

type Deck struct {
	Name       string     `json:"name" db:"name"`
	Calibrated bool       `json:"calibrated" db:"calibrated"`
	X          float64    `json:"x" db:"x"`
	Y          float64    `json:"y" db:"y"`
	Z          float64    `json:"z" db:"z"`
	Qw         float64    `json:"qw" db:"qw"`
	Qx         float64    `json:"qx" db:"qx"`
	Qy         float64    `json:"qy" db:"qy"`
	Qz         float64    `json:"qz" db:"qz"`
	Locations  []Location `json:"locations"`
}

type Location struct {
	Name string  `json:"name" db:"name"`
	X    float64 `json:"x" db:"x"`
	Y    float64 `json:"y" db:"y"`
	Z    float64 `json:"z" db:"z"`
	Qw   float64 `json:"qw" db:"qw"`
	Qx   float64 `json:"qx" db:"qx"`
	Qy   float64 `json:"qy" db:"qy"`
	Qz   float64 `json:"qz" db:"qz"`
}

func GetDecks(tx *sqlx.Tx) ([]Deck, error) {
	var decks []Deck
	err := tx.Select(&decks, "SELECT * FROM deck")
	if err != nil {
		return decks, err
	}
	for i, deck := range decks {
		var locations []Location
		err = tx.Select(&locations, "SELECT name, x, y, z FROM location WHERE deck = ?", deck.Name)
		if err != nil {
			return decks, err
		}
		decks[i].Locations = locations
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

func CreateDeck(tx *sqlx.Tx, deck InputDeck) error {
	_, err := tx.Exec("INSERT INTO deck(name) VALUES (?)", deck.Name)
	if err != nil {
		return err
	}
	for _, location := range deck.Locations {
		_, err := tx.Exec("INSERT INTO location(deck, name, x, y, z, qw, qx, qy, qz) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", deck.Name, location.Name, location.X, location.Y, location.Z, location.Qw, location.Qx, location.Qy, location.Qz)
		if err != nil {
			return err
		}
	}
	return nil
}

func SetDeckCalibration(tx *sqlx.Tx, name string, x float64, y float64, z float64, qw float64, qx float64, qy float64, qz float64) error {
	_, err := tx.Exec("UPDATE deck SET calibrated = ?, x = ?, y = ?, z = ?, qw = ?, qx = ?, qy = ?, qz = ? WHERE name = ?", true, x, y, z, qw, qx, qy, qz, name)
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
	Qw      float64 `json:"qw" db:"qw"`
	Qx      float64 `json:"qx" db:"qx"`
	Qy      float64 `json:"qy" db:"qy"`
	Qz      float64 `json:"qz" db:"qz"`
}

type CommandMove struct {
	Command         string  `json:"command"`
	Deck            string  `json:"name"`
	Location        string  `json:"location"`
	LabwareName     string  `json:"labware_name"`
	Address         string  `json:"address"`
	DepthFromBottom float64 `json:"depth_from_bottom"`
}

type Command struct {
	Command  string
	Pose     kinematics.Pose
	WaitTime int // Milliseconds
}

func ExecuteProtocol(db *sqlx.DB, arm ar3.Arm, protocol []byte) error {
	tx := db.MustBegin()
	err := arm.MoveJointRadians(5, 10, 10, 10, 10, 1, 1, 1, 1, 1, 1, 0)
	if err != nil {
		return err
	}
	var steps []json.RawMessage
	if err := json.Unmarshal(protocol, &steps); err != nil {
		return err
	}
	var commands []Command
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
			commands = append(commands, Command{"move", kinematics.Pose{Position: kinematics.Position{X: movexyz.X, Y: movexyz.Y, Z: movexyz.Z}, Rotation: kinematics.Quaternion{W: movexyz.Qw, X: movexyz.Qx, Y: movexyz.Qy, Z: movexyz.Qz}}, 0})
		case "move":
			var move CommandMove
			err := json.Unmarshal(step, &move)
			if err != nil {
				return err
			}
			// Get deck calibration
			deck, err := GetDeck(tx, move.Deck)
			if err != nil {
				return err
			}
			if !deck.Calibrated {
				return fmt.Errorf("Please calibrate the deck")
			}
			locations := make(map[string]Location)
			for _, location := range deck.Locations {
				locations[location.Name] = location
			}
			if _, ok := locations[move.Location]; !ok {
				return fmt.Errorf("Location not in deck")
			}
			targetLocation := locations[move.Location]

			// Get labware
			labware, err := GetLabware(tx, move.LabwareName)
			if err != nil {
				return err
			}
			wells := make(map[string]Well)
			for _, well := range labware.Wells {
				wells[well.Address] = well
			}
			if _, ok := wells[move.Address]; !ok {
				return fmt.Errorf("Well not in labware")
			}
			targetWell := wells[move.Address]

			// Move above the well, then into it
			locationOffsetX := deck.X + targetLocation.X
			locationOffsetY := deck.Y + targetLocation.Y
			locationOffsetZ := deck.Z + targetLocation.Z
			wellOffsetX := locationOffsetX + targetWell.X
			wellOffsetY := locationOffsetY + targetWell.Y
			wellTop := locationOffsetZ + targetWell.Z + labware.ZDimension + 5
			wellBottom := locationOffsetZ + targetWell.Z + move.DepthFromBottom

			rotation := kinematics.Quaternion{W: deck.Qw, X: deck.Qx, Y: deck.Qy, Z: deck.Qz}

			commands = append(commands, Command{"move", kinematics.Pose{Position: kinematics.Position{X: wellOffsetX, Y: wellOffsetY, Z: wellTop}, Rotation: rotation}, 0})
			commands = append(commands, Command{"move", kinematics.Pose{Position: kinematics.Position{X: wellOffsetX, Y: wellOffsetY, Z: wellBottom}, Rotation: rotation}, 0})
			commands = append(commands, Command{"move", kinematics.Pose{Position: kinematics.Position{X: wellOffsetX, Y: wellOffsetY, Z: wellTop}, Rotation: rotation}, 0})
		}
	}
	// Exit our transaction
	err = tx.Rollback()
	if err != nil {
		return err
	}

	// Now execute the commands
	err = executeProtocolWithCache(arm, commands)
	if err != nil {
		return err
	}
	return nil
}

func executeProtocolWithCache(arm ar3.Arm, commands []Command) error {
	var err error
	for _, command := range commands {
		if command.Command == "move" {
			err = arm.Move(25, 10, 10, 10, 10, command.Pose)
			if err != nil {
				return err
			}
		} else {
			err = arm.Wait(command.WaitTime)
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
        z REAL NOT NULL DEFAULT 0,
	qw REAL NOT NULL DEFAULT 0,
	qx REAL NOT NULL DEFAULT 0,
	qy REAL NOT NULL DEFAULT 0,
	qz REAL NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS location (
	name TEXT,
	deck TEXT NOT NULL REFERENCES deck(name) ON DELETE CASCADE,
	x REAL NOT NULL,
        y REAL NOT NULL,
        z REAL NOT NULL,
	qw REAL NOT NULL,
        qx REAL NOT NULL,
        qy REAL NOT NULL,
        qz REAL NOT NULL
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
