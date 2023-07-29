#!/bin/bash

set -euo pipefail

DATABASE_URI=${DATABASE_URI:-postgresql://127.0.0.1/postgres?user=postgres&password=mypassword&sslmode=disable}

mkdir -p tmp/

go run main.go -q "select * from users where 1=1" > tmp/users.csv

diff tmp/users.csv fixtures/users.csv
