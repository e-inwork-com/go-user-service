package main

import (
	"expvar"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/e-inwork-com/go-user-service/api"
	"github.com/e-inwork-com/go-user-service/internal/data"
	"github.com/e-inwork-com/go-user-service/internal/jsonlog"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env if available
	err := godotenv.Load()
	if err != nil {
		log.Println("Enviroment file .env is not found!")
	}

	// Set Configuration
	var cfg api.Config

	// Read environment  from a command line and OS
	flag.IntVar(&cfg.Port, "port", 4001, "API server port")
	flag.StringVar(&cfg.Env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.Db.Dsn, "db-dsn", os.Getenv("DBDSN"), "Database DSN")
	flag.StringVar(&cfg.Auth.Secret, "auth-secret", os.Getenv("AUTHSECRET"), "Authentication Secret")
	flag.IntVar(&cfg.Db.MaxOpenConn, "db-max-open-conn", 25, "Database max open connections")
	flag.IntVar(&cfg.Db.MaxIdleConn, "db-max-idle-conn", 25, "Database max idle connections")
	flag.StringVar(&cfg.Db.MaxIdleTime, "db-max-idle-time", "15m", "Database max connection idle time")
	flag.BoolVar(&cfg.Limiter.Enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.Float64Var(&cfg.Limiter.Rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.Limiter.Burst, "limiter-burst", 4, "Rate limiter maximum burst")
	displayVersion := flag.Bool("version", false, "Display version and exit")
	flag.Parse()

	// Set CORS Trusted Origins
	cfg.Cors.TrustedOrigins = strings.Fields(os.Getenv("CORS-TRUSTED-ORIGINS"))

	// Show version on the terminal
	if *displayVersion {
		fmt.Printf("Version:\t%s\n", api.Version)
		fmt.Printf("Build time:\t%s\n", api.BuildTime)
		os.Exit(0)
	}

	// Set logger
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// Set Database
	db, err := api.OpenDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()

	// Log a status of the database
	logger.PrintInfo("database connection pool established", nil)

	// Publish variables
	expvar.NewString("version").Set(api.Version)
	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))
	expvar.Publish("database", expvar.Func(func() interface{} {
		return db.Stats()
	}))
	expvar.Publish("timestamp", expvar.Func(func() interface{} {
		return time.Now().Unix()
	}))

	// Set the application
	app := &api.Application{
		Config: cfg,
		Logger: logger,
		Models: data.InitModels(db),
	}

	// Run the application
	err = app.Serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
}
