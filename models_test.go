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
	labwares, err := GetLabwares(tx)
	if err != nil {
		t.Errorf("Failed to get all labwares. Got error: %s", err)
	}
	if len(labwares) != 2 {
		t.Errorf("Should have gotten 2 labwares. Got %d", len(labwares))
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
	deck1 := Deck{Name: "deck1", Locations: []Location{Location{Name: "1", X: 1, Y: 1, Z: 1}}}
	deck2 := Deck{Name: "deck2", Locations: []Location{Location{Name: "2", X: 2, Y: 2, Z: 2}}}
	tx := db.MustBegin()

	var err error
	for _, deck := range []Deck{deck1, deck2} {
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
	if len(decks) != 2 {
		t.Errorf("Should have gotten 2 decks. Got %d", len(decks))
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
	err = SetDeckCalibration(tx, "deck1", 10, 10, 10)
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
	// Insert A deck and a plate labware
	tx := db.MustBegin()
	nestPlate := Labware{"nest_96_wellplate_100ul_pcr_full_skirt", 10, []Well{Well{Address: "A1", Depth: 14.78, Diameter: 5.34, X: 14.38, Y: 74.24, Z: 0.92}, Well{Address: "A1", Depth: 14.78, Diameter: 5.34, X: 14.38, Y: 65.24, Z: 0.92}}}
	deck := Deck{Name: "deck", Locations: []Location{Location{Name: "1", X: 1, Y: 1, Z: 1}}}

	var err error
	err = CreateLabware(tx, nestPlate)
	if err != nil {
		t.Errorf("Failed to CreateLabware: %s", err)
	}
	err = CreateDeck(tx, deck)
	if err != nil {
		t.Errorf("Failed to CreateDeck: %s", err)
	}

	// Calibrate deck
	err = SetDeckCalibration(tx, "deck", 323.08000000000004, -3.4527691855160434e-14, 474.77)
	if err != nil {
		t.Errorf("Failed to SetDeckCalibration: %s", err)
	}

	// Command MoveXYZ
	var commandXYZ []CommandXyz
	commandXYZ = append(commandXYZ, CommandXyz{"movexyz", 132, 158, 121})
	//commandXYZ = append(commandXYZ, CommandXyz{"movexyz", 132, 158, 121}) // Move up by 20
	b, err := json.Marshal(&commandXYZ)
	if err != nil {
		t.Errorf("Failed to json.Marshal: %s", err)
	}

	// ExecuteProtocol
	err = ExecuteProtocol(app.ArmMock, b)
	if err != nil {
		t.Errorf("Failed to ExecuteProtocol: %s", err)
	}
}
