/* Package main is the API for the Ammonite One pipetting robotic arm system.

The Ammonite One is planned to be a simple production-ready pipetting robot
built for developers who want a stable and reasonable API for interacting with
a pipetting system. It is intended to be integrated
*/
package main

import (
	_ "embed"
	"encoding/json"
	"github.com/jmoiron/sqlx"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"log"
	_ "modernc.org/sqlite"
	"net/http"
	"os"
)

/******************************************************************************

                               API

******************************************************************************/

// @title Ammonite API
// @version 0.1
// @description The Ammonite API interface.
// @BasePath /api/
func main() {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = ":memory:"
	}
	db, err := sqlx.Open("sqlite", dbUrl)
	if err != nil {
		log.Fatalf("Failed to connect to database with error: %s", err)
	}
	_ = CreateDatabase(db)
	app := initializeApp(db)

	// Serve application
	s := &http.Server{
		Addr:    ":8080",
		Handler: app.Router,
	}
	log.Fatal(s.ListenAndServe())
}

// App is a struct containing all information about the currently deployed
// application, such as the router and database.
type App struct {
	Router *httprouter.Router
	DB     *sqlx.DB
}

// initalizeApp initializes an App for all endpoints to use.
func initializeApp(db *sqlx.DB) App {
	var app App
	app.Router = httprouter.New()
	app.DB = db

	// Basic routes
	app.Router.GET("/api/ping", app.Ping)
	app.Router.GET("/api/spec", app.OpenApiJSON)
	app.Router.GET("/docs", app.Redocs)
	app.Router.GET("/swagger_docs", app.SwaggerDocs)

	// Labwares
	app.Router.GET("/api/labwares", rootHandler(app.ApiGetLabwares).ServeHTTP)
	app.Router.GET("/api/labwares/:name", rootHandler(app.ApiGetLabware).ServeHTTP)
	app.Router.POST("/api/labwares", rootHandler(app.ApiPostLabware).ServeHTTP)
	app.Router.DELETE("/api/labwares/:name", rootHandler(app.ApiDeleteLabware).ServeHTTP)

	// Protocol
	app.Router.POST("/api/protocols", rootHandler(app.ApiProtocol).ServeHTTP)

	return app
}

type rootHandler func(http.ResponseWriter, *http.Request, httprouter.Params) error

// rootHandler handles errors for endpoints.
func (fn rootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Run function
	err := fn(w, r, p)
	if err != nil {
		log.Printf("ERROR: %s", err) // Log the error
		w.WriteHeader(400)
		_, err = w.Write([]byte(err.Error()))
		if err != nil {
			log.Printf("An error occurred while writing error: %v", err)
		}
	}
}

type Message struct {
	Message string `json:"message"`
}

// Ping is a simple route for verifying that the service is online.
// @Summary A pingable endpoint
// @Tags dev
// @Produce plain
// @Success 200 {object} Message
// @Router /ping [get]
func (app *App) Ping(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	encoder := json.NewEncoder(w)
	_ = encoder.Encode(Message{"Online"})
}

//go:embed docs/swagger.json
var doc []byte

// OpenApiJSON provides the an OpenApi JSON spec.
// @Summary OpenApi spec in JSON format
// @Tags dev
// @Produce json
// @Success 200 {string} openapi
// @Router /spec [get]
func (app *App) OpenApiJSON(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(doc)
}

//go:embed html/redoc.html
var redoc []byte

// Redocs provides a human-friendly swagger ui interface.
func (app *App) Redocs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_, _ = w.Write([]byte(redoc))
}

//go:embed html/swagger.html
var swaggerdoc []byte

// SwaggerDocs provides a human-friendly swagger ui interface.
func (app *App) SwaggerDocs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_, _ = w.Write([]byte(swaggerdoc))
}

/******************************************************************************

                                Labware

******************************************************************************/

// ApiGetLabwares is a route for getting all labwares.
// @Summary Get all labwares
// @Tags labware
// @Produce json
// @Success 200 {object} []Labware
// @Failure 400 {string} string
// @Router /labwares [get]
func (app *App) ApiGetLabwares(w http.ResponseWriter, r *http.Request, _ httprouter.Params) error {
	tx, err := app.DB.Beginx()
	if err != nil {
		return err
	}

	labwares, err := GetLabwares(tx)
	if err != nil {
		return err
	}

	err = tx.Rollback()
	if err != nil {
		return err
	}

	err = json.NewEncoder(w).Encode(labwares)
	if err != nil {
		return err
	}
	return nil

}

// ApiGetLabware is a route for getting a single labware.
// @Summary Get one labware
// @Tags labware
// @Produce json
// @Param name path string true "Labware name"
// @Success 200 {object} Labware
// @Failure 400 {string} string
// @Router /labwares/{name} [get]
func (app *App) ApiGetLabware(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {
	tx, err := app.DB.Beginx()
	if err != nil {
		return err
	}

	labware, err := GetLabware(tx, ps.ByName("name"))
	if err != nil {
		return err
	}

	err = tx.Rollback()
	if err != nil {
		return err
	}

	err = json.NewEncoder(w).Encode(labware)
	if err != nil {
		return err
	}
	return nil

}

// ApiPostLabware is a route to create a labware.
// @Summary Create one labware
// @Tags labware
// @Accept json
// @Produce json
// @Param labware body Labware true "Labware"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Router /labwares/ [post]
func (app *App) ApiPostLabware(w http.ResponseWriter, r *http.Request, _ httprouter.Params) error {
	tx, err := app.DB.Beginx()
	if err != nil {
		return err
	}

	var labware Labware
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(reqBody, &labware)
	if err != nil {
		return err
	}

	err = CreateLabware(tx, labware)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	err = json.NewEncoder(w).Encode(Message{"successful"})
	if err != nil {
		return err
	}
	return nil
}

// ApiDeleteLabware is a route to delete a labware.
// @Summary Delete one labware
// @Tags labware
// @Produce json
// @Param name path string true "Labware name"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Router /labwares/{name} [delete]
func (app *App) ApiDeleteLabware(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {
	tx, err := app.DB.Beginx()
	if err != nil {
		return err
	}

	err = DeleteLabware(tx, ps.ByName("name"))
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	err = json.NewEncoder(w).Encode(Message{"successful"})
	if err != nil {
		return err
	}
	return nil
}

/******************************************************************************

                                Protocol

******************************************************************************/

// ApiProtocol runs a protocol
// @Summary Run a protocol
// @Tags protocol
// @Accept json
// @Produce json
// @Param collection body []CommandInput true "commandInput"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Router /protocol [post]
func (app *App) ApiProtocol(w http.ResponseWriter, r *http.Request, _ httprouter.Params) error {
	var commands []interface{}
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(reqBody, &commands)
	if err != nil {
		return err
	}

	var commandInputs []CommandInput
	for _, command := range commands {
		var newCommand CommandInput
		var ok bool
		newCommand, ok = command.(CommandXyz)
		if ok {
			commandInputs = append(commandInputs, newCommand)
		}

		newCommand, ok = command.(CommandMove)
		if ok {
			commandInputs = append(commandInputs, newCommand)
		}
	}

	err = ExecuteProtocol(app.DB, commandInputs)
	if err != nil {
		return err
	}

	err = json.NewEncoder(w).Encode(Message{"successful"})
	if err != nil {
		return err
	}
	return nil
}
