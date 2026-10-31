package config

// ContractVersion is the semantic version of the OpenAPI contract
// defined in contracts/openapi/.
//
// This is the SINGLE SOURCE OF TRUTH for the contract version.
// Bump it when the spec changes (see contracts/README.md); the build
// stamps it into bundled.yaml (make contract-bundle / Dockerfile).
//
// Used by:
//   - /readyz endpoint (reported as "contract_version")
//   - Makefile and Dockerfile (extracted via grep/sed)
const ContractVersion = "0.13.0"
