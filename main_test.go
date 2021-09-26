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
