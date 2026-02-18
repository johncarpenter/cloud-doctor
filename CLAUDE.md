# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cloud Doctor is a Go CLI tool that performs multi-cloud health checks across AWS, GCP, and Azure. It diagnoses cost anomalies, detects idle resources, and provides infrastructure analysis from the terminal. Forked from [elC0mpa/aws-doctor](https://github.com/elC0mpa/aws-doctor) and extended with multi-cloud support.

## Build & Run Commands

```bash
# Build
go build -o cloud-doctor

# Run (AWS default)
./cloud-doctor

# Run with provider flags
./cloud-doctor --provider gcp --project <id> --billing-account billingAccounts/<id>
./cloud-doctor --provider azure --subscription <uuid>
./cloud-doctor --provider all --project <id> --billing-account billingAccounts/<id> --subscription <uuid>

# Analysis modes
./cloud-doctor --trend          # 6-month spending trend
./cloud-doctor --waste          # Unused resource detection

# Dependency management
go mod tidy

# Run tests
go test ./...
```

There is no Makefile, linter config, or CI pipeline. The project uses standard `go build`.

## Architecture

### Layered Design

```
app.go (entry point, routing by --provider flag)
  → service/orchestrator/    (workflow coordination: default | trend | waste)
    → service/{aws,gcp,azure}/  (provider implementations)
      → model/                  (shared data structures)
  → utils/                     (terminal output: tables, charts, banners)
```

### Provider Interface Pattern

All three cloud providers implement the same three interfaces defined in `service/interfaces.go`:

- **IdentityService** — `GetAccountInfo()`: returns account/project/subscription ID
- **CostService** — billing queries: current month, last month, 6-month history
- **ResourceService** — waste detection: unused volumes, IPs, stopped instances, expiring reservations

Each provider has four sub-packages under `service/<provider>/`: `config`, `identity` (or `sts` for AWS), cost service (`costexplorer`/`billing`/`costmanagement`), and `compute` (or `ec2`/`elb` for AWS).

### Multi-Cloud Execution

`app.go` handles `--provider all` by running provider-specific functions in parallel via goroutines + `sync.WaitGroup`. Each provider's errors are collected independently — a failing provider doesn't block others.

### Orchestrator

`service/orchestrator/service.go` coordinates the three workflows (default cost comparison, trend, waste). It accepts the three service interfaces via constructor injection and routes based on `model.Flags`.

### Display Layer

`utils/` contains all terminal rendering: `cost_table.go` (month-over-month comparison), `barchart.go` (6-month trend via ntcharts), `waste_table.go` (resource waste summary), `multicloud.go` (aggregated cross-provider views). Uses `go-pretty` for tables and `lipgloss` for styling.

## Key Dependencies

| Library | Purpose |
|---------|---------|
| `aws-sdk-go-v2` | AWS API (STS, Cost Explorer, EC2, ELB) |
| `cloud.google.com/go/bigquery` | GCP billing data via BigQuery |
| `azure-sdk-for-go` | Azure API (Cost Management, Compute, Network, Reservations) |
| `go-pretty/v6` | Table formatting |
| `lipgloss` | Terminal styling |
| `ntcharts` | ASCII bar charts |
| `spinner` | Loading indicator |
| `go-figure` | ASCII art banner |

## Adding a New Cloud Provider

1. Create `model/<provider>.go` with provider-specific types implementing `AccountInfo`, `CostInfo`, `UnusedVolume`, etc.
2. Create `service/<provider>/config/`, `identity/`, cost service, and `compute/` packages implementing the three interfaces from `service/interfaces.go`
3. Add a `run<Provider>()` function in `app.go` following the AWS/GCP/Azure pattern
4. Add the provider to the `runAll*` parallel execution functions in `app.go`
5. Add display support in `utils/multicloud.go`

## CLI Flags

Defined in `model/flag.go`, parsed in `service/flag/service.go`:

- `--provider` (aws|gcp|azure|all, default: aws)
- `--region` (default: us-east-1, AWS only)
- `--profile` (AWS credential profile)
- `--project` (GCP project ID, required for GCP)
- `--billing-account` (GCP billing account, required for GCP costs)
- `--subscription` (Azure subscription ID, required for Azure)
- `--trend` / `--waste` (analysis mode toggles)

## MCP Server

The project includes an MCP (Model Context Protocol) server at `cmd/mcp/` that exposes cloud operations as tools for AI assistants.

### Build & Run

```bash
# Build MCP server
go build -o cloud-doctor-mcp ./cmd/mcp

# Run (stdio mode for MCP clients)
./cloud-doctor-mcp
```

### MCP Architecture

```
cmd/mcp/
├── main.go           # Server entry point, tool registration
├── config.go         # Environment variable configuration
├── response/
│   ├── types.go      # JSON response structures
│   └── convert.go    # Model to response converters
└── tools/
    ├── aws.go        # AWS tool handlers (9 tools)
    ├── gcp.go        # GCP tool handlers (9 tools)
    ├── azure.go      # Azure tool handlers (10 tools)
    └── multicloud.go # Multi-cloud aggregate tools (2 tools)
```

### Environment Variables

| Variable | Provider | Required | Default |
|----------|----------|----------|---------|
| `AWS_REGION` | AWS | No | `us-east-1` |
| `AWS_PROFILE` | AWS | No | default |
| `GCP_PROJECT_ID` | GCP | Yes* | - |
| `GCP_BILLING_ACCOUNT` | GCP | Yes* | - |
| `AZURE_SUBSCRIPTION_ID` | Azure | Yes* | - |

*Required only when using that provider's tools

### Key Dependencies

| Library | Purpose |
|---------|---------|
| `github.com/mark3labs/mcp-go` | MCP server framework |

## Authentication

Each provider uses its standard credential chain. No custom auth is implemented:
- **AWS**: env vars → `~/.aws/credentials` → IAM roles → SSO
- **GCP**: Application Default Credentials (`gcloud auth application-default login`)
- **Azure**: DefaultAzureCredential (`az login`)

