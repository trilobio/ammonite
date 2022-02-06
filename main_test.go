package main

import (
	"bytes"
	"encoding/json"
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

func TestProtocolApi(t *testing.T) {
	// Create a new deck
	var moves []CommandInput
	moves = append(moves, CommandXyz{132, 158, 121})
	moves = append(moves, CommandXyz{132, 158, 141})
	moves = append(moves, CommandMove{Deck: "deck", Location: "1", LabwareName: "nest_96_wellplate_100ul_pcr_full_skirt", Address: "A1", DepthFromBottom: 1})
	moves = append(moves, CommandMove{Deck: "deck", Location: "1", LabwareName: "nest_96_wellplate_100ul_pcr_full_skirt", Address: "B1", DepthFromBottom: 1})

	m, _ := json.Marshal(moves)
	req := httptest.NewRequest("POST", "/api/protocols", bytes.NewReader(m))
	resp := httptest.NewRecorder()
	app.Router.ServeHTTP(resp, req)

	success := `{"message":"successful"}`
	if strings.TrimSpace(resp.Body.String()) != success {
		t.Errorf("Unexpected response. Expected: " + success + "\nGot: " + resp.Body.String())
	}
}
