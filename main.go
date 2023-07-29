package main

import (
	"encoding/csv"
	"log"
	"os"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var opts struct {
	Query string `short:"q" long:"query" description:"SQL query" required:"true"`
}

func main() {
	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		os.Exit(1)
	}

	databaseUri := os.Getenv("DATABASE_URI")
	if strings.TrimSpace(databaseUri) == "" {
		log.Fatalln("env var DATABASE_URI must be set")
	}

	if err := invoke(databaseUri, opts.Query); err != nil {
		log.Fatalln(err)
	}
}

func invoke(connectionUri string, query string) error {
	db, err := sqlx.Connect("postgres", connectionUri)
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
