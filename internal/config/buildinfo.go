package config

// Build-time variables injected via -ldflags during Docker image builds.
//
// In local development (make dev, make app-build) these keep their default
// values ("dev" / "unknown") because the exact version doesn't matter.
//
// In production the Dockerfile (and CI) injects real values:
//
//	docker build \
//	  --build-arg APP_VERSION=$(git describe --tags) \
//	  --build-arg APP_COMMIT=$(git rev-parse --short HEAD) .
//
// Which the Dockerfile translates into:
//
//	go build -ldflags="-X ductifact/internal/config.Version=${APP_VERSION}
//	                     -X ductifact/internal/config.Commit=${APP_COMMIT}"
//
// Used by:
//   - GET /readyz (reported as "version" and "commit")
var (
	Version = "dev"
	Commit  = "unknown"
)
