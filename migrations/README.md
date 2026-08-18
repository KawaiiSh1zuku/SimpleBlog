# Database schema

The project keeps its PostgreSQL schema as ordered SQL files. Apply them in filename order to an empty database.

```powershell
psql $env:DATABASE_URL -v ON_ERROR_STOP=1 -f migrations/000001_init.sql
```

The initial schema is transactional. Do not run schema files automatically when the API process starts; deployment is responsible for applying them before starting a new application version.

To validate the schema against an explicitly configured test database without retaining objects:

```powershell
$env:TEST_DATABASE_URL = "postgres://user:password@localhost:5432/postgres"
go test ./migrations -run TestInitialSchema -v
```

The test creates a random schema inside a transaction and always rolls it back.
