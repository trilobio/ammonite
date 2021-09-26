package main

import (
	"github.com/jmoiron/sqlx"
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
	nestPlate := Labware{"nest_96_wellplate_100ul_pcr_full_skirt", 10, []Well{Well{Address: "A1", Depth: 14.78, Diameter: 5.34, X: 14.38, Y: 74.24, Z: 0.92}, Well{Address: "B1", Depth: 14.78, Diameter: 5.34, X: 14.38, Y: 65.24, Z: 0.92}}}
	deck := Deck{Name: "deck", Locations: []Location{Location{Name: "1", X: 1, Y: 1, Z: 1}}}

	err = CreateLabware(tx, nestPlate)
	if err != nil {
		log.Fatalf("Failed to CreateLabware: %s", err)
	}
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
