package main

import (
	"encoding/csv"
	"errors"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jessevdk/go-flags"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var opts struct {
	Query string `short:"q" long:"query" description:"SQL query" required:"false"`
	File  string `short:"f" long:"file" description:"SQL query file" required:"false"`
}

func main() {
	if err := invoke(); err != nil {
		log.Fatalln(err)
	}
}

func invoke() error {
	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		return err
	}

	if opts.File == "" && opts.Query == "" {
		return errors.New("either file or query must be provided")
	}

	var query string
	if opts.File != "" {
		if bytes, err := os.ReadFile(opts.File); err != nil {
			return err
		} else {
			query = string(bytes)
		}

	} else {
		query = opts.Query
	}

	databaseUri := os.Getenv("DATABASE_URI")
	if strings.TrimSpace(databaseUri) == "" {
		log.Fatalln("env var DATABASE_URI must be set")
	}

	if err := run(databaseUri, query); err != nil {
		return err
	}

	return nil
}

func getDB(connectionUri string) (*sqlx.DB, error) {
	if strings.HasPrefix(connectionUri, "postgres") {
		return sqlx.Connect("postgres", strings.TrimPrefix(connectionUri, "postgres://"))
	} else if strings.HasPrefix(connectionUri, "mysql") {
		return sqlx.Connect("mysql", strings.TrimPrefix(connectionUri, "mysql://"))
	}
	return nil, errors.New("invalid connection URI")
}

func run(connectionUri string, query string) error {
	db, err := getDB(connectionUri)
	if err != nil {
		return err
	}

	rows, err := db.Queryx(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	data := make(chan []string, 20)
	errs := make(chan error, 1)

	go func() {
		isFirstRow := true
		for rows.Next() {
			columns, err := rows.Columns()
			if err != nil {
				errs <- err
				close(errs)
				close(data)
			}
			if isFirstRow {
				data <- columns
				isFirstRow = false
			}

			var record = make([]string, len(columns))
			var recordPointer = make([]any, len(columns))

			for idx := range record {
				recordPointer[idx] = &record[idx]
			}
			rows.Scan(recordPointer...)

			data <- record
		}

		close(data)
		close(errs)
	}()

	for {
		if row, ok := <-data; ok {
			w.Write(row)
		} else {
			break
		}
	}

	if err := <-errs; err != nil {
		return err
	}

	return nil
}
