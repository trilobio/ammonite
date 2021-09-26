package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var app App
var db *sqlx.DB

func TestMain(m *testing.M) {
	var err error
	db, err = sqlx.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatalf("Failed to open SQLite in main test function. Got error: %s", err)
	}
	err = CreateDatabase(db)
	if err != nil {
		log.Fatalf("Failed to create database. Got err: %s", err)
	}
	app = initializeApp(db)

	tx := db.MustBegin()
	deck := InputDeck{Name: "deck", Locations: []Location{Location{Name: "1", X: 1, Y: 1, Z: 1}}}

	err = CreateDeck(tx, deck)
	if err != nil {
		log.Fatalf("Failed to CreateDeck: %s", err)
	}
	err = tx.Commit()
	if err != nil {
		log.Fatalf("Failed to commit: %s", err)
	}

	code := m.Run()
	os.Exit(code)
}

func TestPing(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/ping", nil)
	resp := httptest.NewRecorder()
	app.Router.ServeHTTP(resp, req)

	r := `{"message":"Online"}`
	if strings.TrimSpace(resp.Body.String()) != r {
		t.Errorf("Unexpected response. Expected: " + r + "\nGot: " + resp.Body.String())
	}
}

func TestLabwareApi(t *testing.T) {
	// Create a new labware
	m, _ := json.Marshal(Labware{"apiPlate", 10, []Well{Well{Address: "A1", Depth: 1, Diameter: 1, X: 1, Y: 1, Z: 1}}})
	req := httptest.NewRequest("POST", "/api/labwares", bytes.NewReader(m))
	resp := httptest.NewRecorder()
	app.Router.ServeHTTP(resp, req)

	success := `{"message":"successful"}`
	if strings.TrimSpace(resp.Body.String()) != success {
		t.Errorf("Unexpected response. Expected: " + success + "\nGot: " + resp.Body.String())
	}

	// Get labwares
	req = httptest.NewRequest("GET", "/api/labwares", nil)
	resp = httptest.NewRecorder()
	app.Router.ServeHTTP(resp, req)

	var labwares []Labware
	reqBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Reading /api/labwares should succeed. Got error: %s", err)
	}
	err = json.Unmarshal(reqBody, &labwares)
	if err != nil {
		t.Errorf("Unmarshal of labware should succeed. Got error: %s", err)
	}

	for _, labware := range labwares {
		if labware.Name == "apiPlate" {
			if labware.ZDimension != 10 {
				t.Errorf("ZDimension on apiPlate should be 10. Got: %f", labware.ZDimension)
			}
		}
	}

	// Get apiPlate labware
	req = httptest.NewRequest("GET", "/api/labwares/apiPlate", nil)
	resp = httptest.NewRecorder()
	app.Router.ServeHTTP(resp, req)
	var labware Labware
	reqBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Reading /api/labwares/apiPlate should succeed. Got error: %s", err)
	}
	err = json.Unmarshal(reqBody, &labware)
	if err != nil {
		t.Errorf("Unmarshal of single labware should succeed. Got error: %s", err)
	}

	if labware.Name == "apiPlate" {
		if labware.ZDimension != 10 {
			t.Errorf("ZDimension on single apiPlate should be 10. Got: %f", labware.ZDimension)
		}
	}

	// Delete apiPlate labware
	req = httptest.NewRequest("DELETE", "/api/labwares/apiPlate", nil)
	resp = httptest.NewRecorder()
	app.Router.ServeHTTP(resp, req)
	if strings.TrimSpace(resp.Body.String()) != success {
		t.Errorf("Unexpected response. Expected: " + success + "\nGot: " + resp.Body.String())
	}
}

func TestDeckApi(t *testing.T) {
	// Create a new deck
	m, _ := json.Marshal(InputDeck{"defaultDeck", []Location{Location{Name: "1", X: 1, Y: 1, Z: 1}}})
	req := httptest.NewRequest("POST", "/api/decks", bytes.NewReader(m))
	resp := httptest.NewRecorder()
	app.Router.ServeHTTP(resp, req)

	success := `{"message":"successful"}`
	if strings.TrimSpace(resp.Body.String()) != success {
		t.Errorf("Unexpected response. Expected: " + success + "\nGot: " + resp.Body.String())
	}

	// Get decks
	req = httptest.NewRequest("GET", "/api/decks", nil)
	resp = httptest.NewRecorder()
	app.Router.ServeHTTP(resp, req)

	var decks []Deck
	reqBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Reading /api/decks should succeed. Got error: %s", err)
	}
	err = json.Unmarshal(reqBody, &decks)
	if err != nil {
		t.Errorf("Unmarshal of deck should succeed. Got error: %s", err)
	}

	for _, deck := range decks {
		if deck.Name == "defaultDeck" {
			if deck.Calibrated != false {
				t.Errorf("Initial decks should have no calibration")
			}
		}
	}

	// Get defaultDeck deck
	req = httptest.NewRequest("GET", "/api/decks/defaultDeck", nil)
	resp = httptest.NewRecorder()
	app.Router.ServeHTTP(resp, req)
	var deck Deck
	reqBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Reading /api/decks/defaultDeck should succeed. Got error: %s", err)
	}
	err = json.Unmarshal(reqBody, &deck)
	if err != nil {
		t.Errorf("Unmarshal of single deck should succeed. Got error: %s", err)
	}

	if deck.Name == "defaultDeck" {
		if deck.Calibrated != false {
			t.Errorf("Initial decks should have no calibration")
		}
	}

	// Calibrate deck
	qw, qx, qy, qz := 0.8063737663657652, -0.575080903948282, -0.13494466363153904, 0.02886590702694046
	calibrateUrl := fmt.Sprintf("/api/decks/calibrate/defaultDeck/10/10/10/%f/%f/%f/%f", qw, qx, qy, qz)
	req = httptest.NewRequest("POST", calibrateUrl, nil)
	resp = httptest.NewRecorder()
	app.Router.ServeHTTP(resp, req)
	if strings.TrimSpace(resp.Body.String()) != success {
		t.Errorf("Unexpected response. Expected: " + success + "\nGot: " + resp.Body.String())
	}

	// Delete defaultDeck deck
	req = httptest.NewRequest("DELETE", "/api/decks/defaultDeck", nil)
	resp = httptest.NewRecorder()
	app.Router.ServeHTTP(resp, req)
	if strings.TrimSpace(resp.Body.String()) != success {
		t.Errorf("Unexpected response. Expected: " + success + "\nGot: " + resp.Body.String())
	}
}
