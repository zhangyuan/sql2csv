package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/big"
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
	db, err := getDB(connectionUri)
	if err != nil {
		return err
	}
	defer db.Close()

	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	dataChan := make(chan []string, 100)
	errChan := make(chan error, 1)

	go func() {
		if err := run(db, query, func(columnNames []string) error {
			dataChan <- columnNames
			return nil
		}, func(row []any) error {
			values := make([]string, len(row))
			for idx := range row {
				if row[idx] == nil {
					values[idx] = ""
				} else {
					values[idx] = fmt.Sprintf("%v", row[idx])
				}
			}
			dataChan <- values
			return nil
		}); err != nil {
			errChan <- err
			close(errChan)
		}
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

func run(db *sqlx.DB, query string, onHeader func([]string) error, onRecord func([]any) error) error {
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

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	for rows.Next() {
		rawRecord := make([]any, len(columns))
		recordPointer := make([]any, len(columns))

		for idx := range rawRecord {
			recordPointer[idx] = &rawRecord[idx]
		}

		if err := rows.Scan(recordPointer...); err != nil {
			return err
		}

		for idx := range rawRecord {
			if rawRecord[idx] == nil {
				continue
			}
			value, ok := rawRecord[idx].([]uint8)
			if ok {
				// mysql
				stringValue := string(value)

				// todo: add more datatype mapping
				if columnTypes[idx].DatabaseTypeName() == "BIGINT" {
					bigint := new(big.Int)
					bigint.SetString(stringValue, 10)
					rawRecord[idx] = bigint
				} else if columnTypes[idx].DatabaseTypeName() == "VARCHAR" {
					value := rawRecord[idx].([]uint8)
					rawRecord[idx] = string(value)
				}
			}
		}

		if err := onRecord(rawRecord); err != nil {
			return err
		}
	}

	return nil
}
