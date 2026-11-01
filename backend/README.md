# Backend

Go API with clean architecture:

- `internal/domain`: listing rules and normalization, without I/O
- `internal/application`: search and import use cases, depending on a repository interface
- `internal/adapters/postgres`: PostgreSQL implementation
- `internal/adapters/httpapi`: HTTP request and response handling
- `internal/adapters/csvfeed`: CSV parsing
- `cmd/api`, `cmd/importer`: composition roots

The API is read-only. Imports happen through a local/admin command, so there is no unauthenticated write endpoint. See the root README for setup.
