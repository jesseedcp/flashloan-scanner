# flashloan-scanner


<!--
parent:
  order: false
-->

<div align="center">
  <h1> flashloan-scanner repo </h1>
</div>

<div align="center">
  <a href="https://github.com/cpchain-network/flashloan-scanner/releases/latest">
    <img alt="Version" src="https://img.shields.io/github/tag/cpchain-network/flashloan-scanner.svg" />
  </a>
  <a href="https://github.com/cpchain-network/flashloan-scanner/blob/main/LICENSE">
    <img alt="License: Apache-2.0" src="https://img.shields.io/github/license/cpchain-network/flashloan-scanner.svg" />
  </a>
  <a href="https://pkg.go.dev/github.com/cpchain-network/flashloan-scanner">
    <img alt="GoDoc" src="https://godoc.org/github.com/cpchain-network/flashloan-scanner?status.svg" />
  </a>
</div>

flashloan-scanner is a flash-loan transaction scanning project for protocol-pattern detection and result analysis.

**Tips**: 
- need [Go 1.23+](https://golang.org/dl/)
- need [Postgresql](https://www.postgresql.org/)


## Install

### Install dependencies
```bash
go mod tidy
```
### build
```bash
make
```

### Config env

- For evm chain contracts config, you can [config](https://github.com/cpchain-network/flashloan-scanner/tree/main/config) director and refer to exist config do it.
- For yaml config, you can use `flashloan-scanner.example.yaml` as the project template and fill in your real environment values.

### run scanner
```bash
./flashloan-scanner flashloan-scan --config ./flashloan-scanner.yaml
```

### run report
```bash
./flashloan-scanner flashloan-report --config ./flashloan-scanner.local.yaml --chain-id 11155111 --limit 5
```

## Flashloan scanner smoke path

The repository now includes a first-pass flashloan scanner for:

- `aave_v3`
- `balancer_v2`
- `uniswap_v2`

The simplest local smoke path is:

```powershell
pwsh -NoProfile -File .\scripts\flashloan-smoke.ps1 -Config .\flashloan-scanner.local.yaml
```

If you want a fully local scanner smoke without real chain tx fetching, load the built-in Aave fixture first:

```powershell
pwsh -NoProfile -File .\scripts\flashloan-smoke.ps1 -Config .\flashloan-scanner.local.yaml -LoadFixture
```

For a Balancer V2 local smoke, use the dedicated config:

```powershell
pwsh -NoProfile -File .\scripts\flashloan-smoke.ps1 -Config .\flashloan-scanner.balancer.local.yaml -LoadFixture
```

For a Uniswap V2 local smoke, use the dedicated config:

```powershell
pwsh -NoProfile -File .\scripts\flashloan-smoke.ps1 -Config .\flashloan-scanner.uniswap.local.yaml -LoadFixture
```

For a real Ethereum mainnet sample window that contains all three protocols, a ready-to-edit config template is available at:

```text
flashloan-scanner.mainnet.real-window.yaml
```

That template is prefilled with:

- chain id `1`
- block range `22485844` to `22486844`
- official Aave V3 Pool
- official Balancer V2 Vault
- one confirmed real Uniswap V2 flash-swap pair sample

The script does two things:

1. runs `migrate`
2. runs `flashloan-scan`

Notes:

- `scanner.enabled` must be `true` in the selected config
- `scanner.run_mode` should be `once` for smoke runs
- the scanner depends on existing `block_headers_<chain_id>` and `contract_events_<chain_id>` data
- `aave.pools`, `balancer.vaults`, or `uniswap_v2.factories/pairs` must be configured for the selected protocol
- when using `-LoadFixture`, set `scanner.skip_tx_fetch: true`
- the built-in fixture loader currently supports `aave_v3`, `balancer_v2`, and `uniswap_v2`
- `scanner.trace_enabled: true` is currently meaningful for `aave_v3`, `balancer_v2`, and `uniswap_v2`
- enabling trace requires the selected RPC endpoint to support `debug_traceTransaction` with `callTracer`
- the built-in fixture path does not inject trace data, so keep `scanner.trace_enabled: false` for local fixture smoke runs

## Flashloan result report

To inspect scanner results already written into the database:

```powershell
go run .\cmd\flashloan-scanner flashloan-report --config .\flashloan-scanner.local.yaml --chain-id 11155111 --limit 5
```

To inspect a single transaction:

```powershell
go run .\cmd\flashloan-scanner flashloan-report --config .\flashloan-scanner.local.yaml --chain-id 11155111 --tx-hash 0x...
```

To export JSON:

```powershell
go run .\cmd\flashloan-scanner flashloan-report --config .\flashloan-scanner.local.yaml --chain-id 11155111 --format json --output .\flashloan-report.json
```

To export CSV:

```powershell
go run .\cmd\flashloan-scanner flashloan-report --config .\flashloan-scanner.local.yaml --chain-id 11155111 --format csv --output .\flashloan-report.csv
```

If you prefer direct PostgreSQL queries, reusable query templates are available at:

- `sql/flashloan_report_queries.sql`

The file includes ready-to-run examples for:

- latest transactions
- strict-only transactions
- single-tx drilldown
- interaction + leg drilldown
- protocol-level counts
- candidate / verified / strict summaries
- exclusion-reason analysis
- asset-level borrowed volume
- trace-related strict interaction lookups

## Contribute

TBD

