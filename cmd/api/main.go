package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mroobert/json-api/internal/data"
	"github.com/mroobert/json-api/internal/database"
	"github.com/mroobert/json-api/internal/logger"
)

// version contains the application version number.
const version = "1.0.0"

// config holds all the configuration settings for the application.
// We will read in these configuration settings from command-line
// flags when the application starts.
type config struct {
	db      database.Config
	env     string
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	port int
}

// application holds the dependencies for our HTTP handlers, helpers,
// and middleware.
type application struct {
	config       config
	logger       *logger.Logger
	repositories data.Repositories
}

func main() {
	logger := logger.New(os.Stdout, logger.LevelInfo)

	if err := run(logger); err != nil {
		logger.PrintFatal(err, nil)
		os.Exit(1)
	}
}

// run performs the startup and shutdown sequence.
func run(logger *logger.Logger) error {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.DSN, "db-dsn", os.Getenv("DATABASE"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.MaxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.MaxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.MaxConnIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.Parse()

	db, err := database.OpenConnection(cfg.db)
	if err != nil {
		return fmt.Errorf("error opening database: %v", err)
	}
	defer db.Close()
	logger.PrintInfo("database connection pool established", nil)

	app := &application{
		config:       cfg,
		logger:       logger,
		repositories: data.NewRepositories(db),
	}

	// Start http server.
	err = app.serve()
	if err != nil {
		return err
	}

	return err
}
