#!/bin/bash

set -euo pipefail

export DATABASE_URI=${DATABASE_URI:-"postgresql://127.0.0.1/postgres?user=postgres&password=mypassword&sslmode=disable"}

mkdir -p tmp/

go run main.go -q "select * from users where 1=1" > tmp/users.csv

echo fixtures/users.csv
cat fixtures/users.csv

echo tmp/users.csv
cat tmp/users.csv

diff tmp/users.csv fixtures/users.csv
