package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/Wasid786/schoolManagementSystem/internal/data"
	"github.com/Wasid786/schoolManagementSystem/internal/jsonlog"
	"github.com/Wasid786/schoolManagementSystem/internal/mailer"
	_ "github.com/go-sql-driver/mysql"
)

type config struct {
	port int
	env  string
	cors struct {
		trustedOrigins []string
	}
	jwt struct {
		secret string
	}
	smtp struct {
		host string

		port int

		username string
		password string
		sender   string
	}
}

type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

func main() {
	var cfg config
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	dsn := flag.String("mysql", "new_user:sql@/schooldb?parseTime=true", "MySQL data source name")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "3fc9cc5c63403a", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "8bb2d6764d71ce", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.example>", "SMTP sender")
	flag.StringVar(&cfg.jwt.secret, "jwt-secret", "", "JWT secret")
	displayVersion := flag.Bool("version", false, "Display version and exit")
	fmt.Println(displayVersion)

	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	flag.Parse()
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	db, err := openDB(*dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	err = app.server()
	if err != nil {
		logger.PrintFatal(err, nil)
	}

}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
