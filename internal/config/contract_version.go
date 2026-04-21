package config

// ContractVersion is the semantic version of the OpenAPI contract
// the backend was built against.
//
// This is the SINGLE SOURCE OF TRUTH for the contract version.
// When you bump the contract in the contracts repo, update this
// constant and run `make fetch-contract` to download the new spec.
//
// Used by:
//   - /health endpoint (reported as "contract_version")
//   - Makefile fetch-contract (extracted via grep)
//   - CI pipeline (extracted via grep)
const ContractVersion = "0.7.0"
