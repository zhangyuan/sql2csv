## Usage

You can download the executables from the [Releases](https://github.com/zhangyuan/sql2csv/releases) page.

### Example

Run a local posrtgres instance:

```bash
docker-compose up
```

Open a new termial and run:

```bash
export DATABASE_URI="postgresql://localhost/postgres?user=postgres&password=mypassword&sslmode=disable"

go run main.go -q "select * from users"

# or use the executable on MacOS for example

./sql2csv-amd64-darwin -q "select * from users"
```
