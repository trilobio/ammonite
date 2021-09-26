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
	"github.com/trilobio/ar3"
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
	Router  *http.ServeMux
	DB      *sqlx.DB
	ArmMock ar3.Arm
	Arm     ar3.Arm
}

// initalizeApp initializes an App for all endpoints to use.
func initializeApp(db *sqlx.DB) App {
	var app App
	app.Router = http.NewServeMux()
	app.DB = db
	app.Arm = ar3.ConnectMock()
	app.ArmMock = ar3.ConnectMock()

	// Basic routes
	app.Router.HandleFunc("/api/ping", app.Ping)
	app.Router.HandleFunc("/swagger.json", app.SwaggerJSON)
	app.Router.HandleFunc("/docs", app.SwaggerDocs)

	return app
}

// Ping is a simple route for verifying that the service is online.
// @Summary A pingable endpoint
// @Tags dev
// @Produce plain
// @Success 200 {string} map[string]string
// @Router /ping [get]
func (app *App) Ping(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)
	_ = encoder.Encode(map[string]string{"message": "Online"})
}

//go:embed docs/swagger.json
var doc []byte

// SwaggerJSON provides the swagger docs for this api in JSON format.
func (app *App) SwaggerJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(doc)
}

// SwaggerDocs provides a human-friendly swagger ui interface.
func (app *App) SwaggerDocs(w http.ResponseWriter, r *http.Request) {
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
