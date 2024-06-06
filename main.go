package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
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
	Vars  string `long:"vars" description:"SQL query template variables" required:"false"`
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

	if opts.Vars != "" {
		var vars interface{}
		if err := json.Unmarshal([]byte(opts.Vars), &vars); err != nil {
			return err
		}
		template, err := template.New("template").Parse(query)
		if err != nil {
			return err
		}

		var buf bytes.Buffer

		if err := template.Execute(&buf, vars); err != nil {
			return err
		}
		query = buf.String()
	}

	databaseUri := os.Getenv("DATABASE_URI")
	if strings.TrimSpace(databaseUri) == "" {
		log.Fatalln("env var DATABASE_URI must be set")
	}

	if err := write2csv(databaseUri, query); err != nil {
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

func write2csv(connectionUri string, query string) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	dataChan := make(chan []string, 100)
	errChan := make(chan error, 1)

	go func() {
		if err := run(connectionUri, query, func(columnNames []string) error {
			dataChan <- columnNames
			return nil
		}, func(row []any) error {
			values := make([]string, len(row))
			for idx := range row {
				values[idx] = fmt.Sprintf("%v", row[idx])
			}
			dataChan <- values
			return nil
		}); err != nil {
			errChan <- err
		}

		close(dataChan)
		close(dataChan)
	}()

Loop:
	for {
		select {
		case data, ok := <-dataChan:
			if !ok {
				break Loop
			}
			if err := w.Write(data); err != nil {
				return err
			}
		case err, ok := <-errChan:
			if !ok {
				break Loop
			}
			return err
		}
	}
	return nil
}

func run(connectionUri string, query string, onHeader func([]string) error, onRecord func([]any) error) error {
	db, err := getDB(connectionUri)
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Queryx(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	if err := onHeader(columns); err != nil {
		return err
	}

	for rows.Next() {
		var record = make([]*string, len(columns))
		var recordPointer = make([]any, len(columns))

		for idx := range record {
			recordPointer[idx] = &record[idx]
		}

		if err := rows.Scan(recordPointer...); err != nil {
			return err
		}

		var csvRecord = make([]any, len(columns))
		for idx, field := range record {
			if field == nil {
				csvRecord[idx] = ""
			} else {
				csvRecord[idx] = *field
			}
		}

		if err := rows.Scan(recordPointer...); err != nil {
			return err
		}
		if err := onRecord(csvRecord); err != nil {
			return err
		}
	}

	return nil
}
