package main

import (
	"testing"
)

func TestLabware(t *testing.T) {
	labware1 := Labware{"plate1", 10, []Well{Well{Address: "A1", Depth: 1, Diameter: 1, X: 1, Y: 1, Z: 1}}}
	labware2 := Labware{"plate2", 20, []Well{Well{Address: "A1", Depth: 2, Diameter: 2, X: 2, Y: 2, Z: 2}}}
	tx := db.MustBegin()

	var err error
	for _, labware := range []Labware{labware1, labware2} {
		err = CreateLabware(tx, labware)
		if err != nil {
			t.Errorf("Failed to create labware. Got error: %s", err)
		}
	}

	// Get both labwares
	_, err = GetLabwares(tx)
	if err != nil {
		t.Errorf("Failed to get all labwares. Got error: %s", err)
	}

	// Get one labware
	labware, err := GetLabware(tx, "plate1")
	if err != nil {
		t.Errorf("Failed to get single labware. Got error: %s", err)
	}
	if labware.Wells[0].Address != "A1" {
		t.Errorf("First address should be A1. Got: %s", labware.Wells[0].Address)
	}

	// Delete plate1
	err = DeleteLabware(tx, "plate1")
	if err != nil {
		t.Errorf("Failed to delete plate1. Got error: %s", err)
	}

	// Test that it is deleted
	_, err = GetLabware(tx, "plate1")
	if err == nil {
		t.Errorf("Getting a plate that doesn't exist should fail.")
	}

	// Rollback
	err = tx.Rollback()
	if err != nil {
		t.Errorf("Rollback should succeed")
	}
}

func TestExecuteProtocol(t *testing.T) {
	var err error
	// Command MoveXYZ
	var moves []CommandInput
	moves = append(moves, CommandXyz{257, 0, 287})
	moves = append(moves, CommandXyz{257, 0, 307}) // go up by 20
	//moves = append(moves, CommandMove{Deck: "deck", Location: "1", LabwareName: "nest_96_wellplate_100ul_pcr_full_skirt", Address: "A1", DepthFromBottom: 1})
	//moves = append(moves, CommandMove{Deck: "deck", Location: "1", LabwareName: "nest_96_wellplate_100ul_pcr_full_skirt", Address: "B1", DepthFromBottom: 1})

	// ExecuteProtocol
	err = ExecuteProtocol(db, moves)
	if err != nil {
		t.Errorf("Failed to ExecuteProtocol: %s", err)
	}
}
