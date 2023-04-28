## Usage

### Example

Run a local posrtgres instance:

```bash
docker-compose up
```

Open a new termial and run:

```bash
export DATABASE_URI="postgresql://localhost/postgres?user=postgres&password=mypassword&sslmode=disable"

go run main.go -q "select * from users"
```
