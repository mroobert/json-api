package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mroobert/json-api/internal/database"
)

// version contains the application version number.
const version = "1.0.0"

// config holds all the configuration settings for the application.
// We will read in these configuration settings from command-line
// flags when the application starts.
type config struct {
	db   database.Config
	env  string
	port int
}

// application holds the dependencies for our HTTP handlers, helpers,
// and middleware.
type application struct {
	config config
	logger *log.Logger
}

func main() {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	if err := run(logger); err != nil {
		logger.Fatal(err)
		os.Exit(1)
	}
}

// run performs the startup and shutdown sequence.
func run(logger *log.Logger) error {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.DSN, "db-dsn", os.Getenv("DATABASE"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.MaxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.MaxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.MaxConnIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.Parse()

	db, err := database.OpenConnection(cfg.db)
	if err != nil {
		return fmt.Errorf("error opening database: %v", err)
	}
	defer db.Close()
	logger.Print("database connection pool established")

	app := &application{
		config: cfg,
		logger: logger,
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start the HTTP server.
	logger.Printf("starting %s server on %s", cfg.env, srv.Addr)
	err = srv.ListenAndServe()
	if err != nil {
		return fmt.Errorf("error starting %s server on %s", cfg.env, srv.Addr)
	}

	return err
}
