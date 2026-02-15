# Backend Go Blueprint

A clean architecture Go backend template following hexagonal architecture principles. This blueprint provides a solid foundation for building scalable REST APIs with PostgreSQL integration.

## Architecture

This project follows Clean Architecture (Hexagonal Architecture) with the following layers:

- **Domain**: Core business logic and entities
- **Application**: Use cases and business rules
- **Infrastructure**: External adapters (database, messaging)
- **Interfaces**: API handlers and external interfaces

## Features

- REST API with Gin framework
- PostgreSQL database integration with GORM
- Clean architecture structure
- Docker support with docker-compose
- Environment-based configuration
- User management CRUD operations
- Health check endpoint

## Prerequisites

- Go 1.23+
- Docker and Docker Compose
- PostgreSQL (if running locally)

## Quick Start

### Using Docker Compose (Recommended)

1. Clone the repository
2. Copy environment variables:
   ```bash
   cp .env.example .env
   ```
3. Start the services:
   ```bash
   docker-compose up -d
   ```

The API will be available at `http://localhost:8080`

### Local Development

1. Install dependencies:
   ```bash
   go mod download
   ```

2. Set up PostgreSQL database and update `.env` file

3. Run the application:
   ```bash
   go run cmd/api/main.go
   ```

## API Endpoints

### Health Check
- `GET /health` - Service health status

See `examples.http` for detailed API usage examples.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | Server port | 8080 |
| DB_HOST | Database host | localhost |
| DB_PORT | Database port | 5432 |
| DB_USER | Database user | postgres |
| DB_PASSWORD | Database password | postgres |
| DB_NAME | Database name | microservice_db |

## Project Structure

```
.
├── cmd/
│   └── api/                    # Application entry point
├── internal/
│   ├── domain/                 # Domain layer
│   │   ├── entities/          # Business entities
│   │   ├── repositories/      # Repository interfaces
│   │   └── valueobjects/      # Value objects
│   ├── application/           # Application layer
│   │   ├── ports/            # Port interfaces
│   │   ├── services/         # Application services
│   │   └── usecases/         # Use cases
│   ├── infrastructure/        # Infrastructure layer
│   │   └── adapters/         # External adapters
│   │       └── out/database/ # Database implementations
│   └── interfaces/           # Interface layer
│       └── http/            # HTTP handlers and routing
├── docker-compose.yml        # Docker compose configuration
├── Dockerfile               # Docker image definition
├── examples.http           # API usage examples
└── init.sql               # Database initialization
```

## Development

### Running Tests
```bash
go test ./...
```

### Building
```bash
go build -o bin/api cmd/api/main.go
```

### Docker Build
```bash
docker build -t backend-go-blueprint .
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request
