/* Package main is the API for the Ammonite One pipetting robotic arm system.

The Ammonite One is planned to be a simple production-ready pipetting robot
built for developers who want a stable and reasonable API for interacting with
a pipetting system. It is intended to be integrated
*/
package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/julienschmidt/httprouter"
	"github.com/trilobio/ar3"
	"io/ioutil"
	"log"
	_ "modernc.org/sqlite"
	"net/http"
	"os"
	"strconv"
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
	Router  *httprouter.Router
	DB      *sqlx.DB
	ArmMock ar3.Arm
	Arm     ar3.Arm
}

// initalizeApp initializes an App for all endpoints to use.
func initializeApp(db *sqlx.DB) App {
	var app App
	app.Router = httprouter.New()
	app.DB = db
	app.Arm = ar3.ConnectMock()
	app.ArmMock = ar3.ConnectMock()

	// Basic routes
	app.Router.GET("/api/ping", app.Ping)
	app.Router.GET("/swagger.json", app.SwaggerJSON)
	app.Router.GET("/docs", app.SwaggerDocs)

	// Labwares
	app.Router.GET("/api/labwares", rootHandler(app.ApiGetLabwares).ServeHTTP)
	app.Router.GET("/api/labwares/:name", rootHandler(app.ApiGetLabware).ServeHTTP)
	app.Router.POST("/api/labwares", rootHandler(app.ApiPostLabware).ServeHTTP)
	app.Router.DELETE("/api/labwares/:name", rootHandler(app.ApiDeleteLabware).ServeHTTP)

	// Decks
	app.Router.GET("/api/decks", rootHandler(app.ApiGetDecks).ServeHTTP)
	app.Router.GET("/api/decks/:name", rootHandler(app.ApiGetDeck).ServeHTTP)
	app.Router.POST("/api/decks", rootHandler(app.ApiPostDeck).ServeHTTP)
	app.Router.POST("/api/decks/calibrate/:name/:x/:y/:z/:qw/:qx/:qy/:qz", rootHandler(app.ApiCalibrateDeck).ServeHTTP)
	app.Router.DELETE("/api/decks/:name", rootHandler(app.ApiDeleteDeck).ServeHTTP)

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

// SwaggerJSON provides the swagger docs for this api in JSON format.
func (app *App) SwaggerJSON(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(doc)
}

// SwaggerDocs provides a human-friendly swagger ui interface.
func (app *App) SwaggerDocs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// https://stackoverflow.com/questions/55733609/display-swagger-ui-on-flask-without-any-hookups
	swaggerDoc := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <script src="//unpkg.com/swagger-ui-dist@3/swagger-ui-standalone-preset.js"></script>
    <!-- <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/3.22.1/swagger-ui-standalone-preset.js"></script> -->
    <script src="//unpkg.com/swagger-ui-dist@3/swagger-ui-bundle.js"></script>
    <!-- <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/3.22.1/swagger-ui-bundle.js"></script> -->
    <link rel="stylesheet" href="//unpkg.com/swagger-ui-dist@3/swagger-ui.css" />
    <!-- <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/3.22.1/swagger-ui.css" /> -->
    <title>Swagger</title>
</head>
<body>
    <div id="swagger-ui"></div>
    <script>
        window.onload = function() {
          SwaggerUIBundle({
	    spec: %s,
            dom_id: '#swagger-ui',
            presets: [
              SwaggerUIBundle.presets.apis,
              SwaggerUIStandalonePreset
            ],
            layout: "StandaloneLayout"
          })
        }
    </script>
</body>
</html>`, string(doc))
	_, _ = w.Write([]byte(swaggerDoc))
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

                                Deck

******************************************************************************/

// ApiGetDecks is a route for getting all decks.
// @Summary Get all decks
// @Tags deck
// @Produce json
// @Success 200 {object} []Deck
// @Failure 400 {string} string
// @Router /decks [get]
func (app *App) ApiGetDecks(w http.ResponseWriter, r *http.Request, _ httprouter.Params) error {
	tx, err := app.DB.Beginx()
	if err != nil {
		return err
	}

	decks, err := GetDecks(tx)
	if err != nil {
		return err
	}

	err = tx.Rollback()
	if err != nil {
		return err
	}

	err = json.NewEncoder(w).Encode(decks)
	if err != nil {
		return err
	}
	return nil

}

// ApiGetDeck is a route for getting a single deck.
// @Summary Get one deck
// @Tags deck
// @Produce json
// @Param name path string true "Deck name"
// @Success 200 {object} Deck
// @Failure 400 {string} string
// @Router /decks/{name} [get]
func (app *App) ApiGetDeck(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {
	tx, err := app.DB.Beginx()
	if err != nil {
		return err
	}

	deck, err := GetDeck(tx, ps.ByName("name"))
	if err != nil {
		return err
	}

	err = tx.Rollback()
	if err != nil {
		return err
	}

	err = json.NewEncoder(w).Encode(deck)
	if err != nil {
		return err
	}
	return nil

}

// ApiPostDeck is a route to create a deck.
// @Summary Create one deck
// @Tags deck
// @Accept json
// @Produce json
// @Param deck body InputDeck true "Deck"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Router /decks/ [post]
func (app *App) ApiPostDeck(w http.ResponseWriter, r *http.Request, _ httprouter.Params) error {
	tx, err := app.DB.Beginx()
	if err != nil {
		return err
	}

	var deck InputDeck
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(reqBody, &deck)
	if err != nil {
		return err
	}

	err = CreateDeck(tx, deck)
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

// ApiCalibrateDeck is a route to calibrate a deck.
// @Summary Calibrate a deck
// @Tags deck
// @Accept json
// @Produce json
// @Param name path string true "Deck name"
// @Param x path number true "X coordinate"
// @Param y path number true "Y coordinate"
// @Param z path number true "Z coordinate"
// @Param qw path number true "Qw coordinate"
// @Param qx path number true "Qx coordinate"
// @Param qy path number true "Qy coordinate"
// @Param qz path number true "Qz coordinate"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Router /decks/calibrate/{name}/{x}/{y}/{z}/{qw}/{qx}/{qy}/{qz} [post]
func (app *App) ApiCalibrateDeck(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {
	tx, err := app.DB.Beginx()
	if err != nil {
		return err
	}

	x, err := strconv.ParseFloat(ps.ByName("x"), 64)
	if err != nil {
		return err
	}
	y, err := strconv.ParseFloat(ps.ByName("y"), 64)
	if err != nil {
		return err
	}
	z, err := strconv.ParseFloat(ps.ByName("z"), 64)
	if err != nil {
		return err
	}

	qw, err := strconv.ParseFloat(ps.ByName("qw"), 64)
	if err != nil {
		return err
	}
	qx, err := strconv.ParseFloat(ps.ByName("qx"), 64)
	if err != nil {
		return err
	}
	qy, err := strconv.ParseFloat(ps.ByName("qy"), 64)
	if err != nil {
		return err
	}
	qz, err := strconv.ParseFloat(ps.ByName("qz"), 64)
	if err != nil {
		return err
	}

	err = SetDeckCalibration(tx, ps.ByName("name"), x, y, z, qw, qx, qy, qz)
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

// ApiDeleteDeck is a route to delete a deck.
// @Summary Delete one deck
// @Tags deck
// @Produce json
// @Param name path string true "Deck name"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Router /decks/{name} [delete]
func (app *App) ApiDeleteDeck(w http.ResponseWriter, r *http.Request, ps httprouter.Params) error {
	tx, err := app.DB.Beginx()
	if err != nil {
		return err
	}

	err = DeleteDeck(tx, ps.ByName("name"))
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
		fmt.Println("ah")
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

	err = ExecuteProtocol(db, app.Arm, commandInputs)
	if err != nil {
		return err
	}

	err = json.NewEncoder(w).Encode(Message{"successful"})
	if err != nil {
		return err
	}
	return nil
}
