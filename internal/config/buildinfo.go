package config

// Build-time variables injected via -ldflags during Docker image builds.
//
// In local development (make dev, make app-build) these keep their default
// values because the exact identity doesn't matter.
//
// In CI the Dockerfile injects real values:
//
//	docker build \
//	  --build-arg APP_COMMIT=$(git rev-parse HEAD) \
//	  --build-arg APP_BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) .
//
// Which the Dockerfile translates into:
//
//	go build -ldflags="-X ductifact/internal/config.Commit=${APP_COMMIT} \
//	                    -X ductifact/internal/config.BuildTime=${APP_BUILD_TIME}"
//
// Together they provide immutable build identity:
//   - Commit   → exact source code state (what)
//   - BuildTime → when the image was produced (when / ordering)
//
// Used by:
//   - GET /readyz (reported as "commit" and "build_time")
var (
	Commit    = "unknown"
	BuildTime = "unknown"
)
