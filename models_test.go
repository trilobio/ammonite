package main

import (
	"encoding/json"
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

func TestDeck(t *testing.T) {
	deck1 := InputDeck{Name: "deck1", Locations: []Location{Location{Name: "l1", X: 1, Y: 1, Z: 1}}}
	deck2 := InputDeck{Name: "deck2", Locations: []Location{Location{Name: "l2", X: 2, Y: 2, Z: 2}}}
	tx := db.MustBegin()

	var err error
	for _, deck := range []InputDeck{deck1, deck2} {
		err = CreateDeck(tx, deck)
		if err != nil {
			t.Errorf("Failed to create deck. Got error: %s", err)
		}
	}

	// Get both decks
	decks, err := GetDecks(tx)
	if err != nil {
		t.Errorf("Failed to get all decks. Got error: %s", err)
	}
	// One extra from main
	if len(decks) != 3 {
		t.Errorf("Should have gotten 3 decks. Got %d", len(decks))
	}

	// Get one deck
	deck, err := GetDeck(tx, "deck1")
	if err != nil {
		t.Errorf("Failed to get single deck. Got error: %s", err)
	}
	if deck.Calibrated != false {
		t.Errorf("deck calibration should initially be false")
	}

	// Calibrate deck1
	err = SetDeckCalibration(tx, "deck1", 10, 10, 10, 0.8063737663657652, -0.575080903948282, -0.13494466363153904, 0.02886590702694046)
	if err != nil {
		t.Errorf("SetDeckCalibration failed. Got: %s", err)
	}

	// Get deck1 again and then check for calibration
	deck, err = GetDeck(tx, "deck1")
	if err != nil {
		t.Errorf("Failed to get deck second time. Got error: %s", err)
	}
	if deck.Calibrated != true {
		t.Errorf("deck calibration should be true after calibration")
	}

	// Delete deck1
	err = DeleteDeck(tx, "deck1")
	if err != nil {
		t.Errorf("Failed to delete plate1. Got error: %s", err)
	}

	// Test that it is deleted
	_, err = GetDeck(tx, "deck1")
	if err == nil {
		t.Errorf("Getting a deck that doesn't exist should fail.")
	}

	// Rollback
	err = tx.Rollback()
	if err != nil {
		t.Errorf("Rollback should succeed")
	}
}

func TestExecuteProtocol(t *testing.T) {
	var err error
	tx := db.MustBegin()
	// Calibrate deck
	err = SetDeckCalibration(tx, "deck", 132, 158, 121, 0.8063737663657652, -0.575080903948282, -0.13494466363153904, 0.02886590702694046)
	if err != nil {
		t.Errorf("Failed to SetDeckCalibration: %s", err)
	}
	// Commit
	err = tx.Commit()
	if err != nil {
		t.Errorf("Rollback should succeed")
	}

	// Command MoveXYZ
	var moves []interface{}
	moves = append(moves, CommandXyz{"movexyz", 132, 158, 121, 0.8063737663657652, -0.575080903948282, -0.13494466363153904, 0.02886590702694046})
	moves = append(moves, CommandXyz{"movexyz", 132, 158, 141, 0.8063737663657652, -0.575080903948282, -0.13494466363153904, 0.02886590702694046}) // Move up by 20
	moves = append(moves, CommandMove{Command: "move", Deck: "deck", Location: "1", LabwareName: "nest_96_wellplate_100ul_pcr_full_skirt", Address: "A1", DepthFromBottom: 1})
	moves = append(moves, CommandMove{Command: "move", Deck: "deck", Location: "1", LabwareName: "nest_96_wellplate_100ul_pcr_full_skirt", Address: "B1", DepthFromBottom: 1})

	b, err := json.Marshal(&moves)
	if err != nil {
		t.Errorf("Failed to json.Marshal: %s", err)
	}

	// ExecuteProtocol
	err = ExecuteProtocol(db, app.ArmMock, b)
	if err != nil {
		t.Errorf("Failed to ExecuteProtocol: %s", err)
	}
}
