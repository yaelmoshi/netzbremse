# Netzbremse

[![build](https://ci.m0sh1.cc/api/badges/9/status.svg)](https://ci.m0sh1.cc/repos/9)
[![release](https://ci.m0sh1.cc/api/badges/9/status.svg?event=tag)](https://ci.m0sh1.cc/repos/9)

Self-hosted speed telemetry for [netzbremse.de](https://netzbremse.de), backed
by PostgreSQL.

## About Netzbremse

[Netzbremse](https://netzbremse.de) ("net brake") is a campaign by
[AKVorrat.at](https://akvorrat.at) investigating throttling and net-neutrality
violations by Deutsche Telekom and other European ISPs. The project collects
crowd-sourced speed-test data via a Cloudflare-backed browser measurement to
build an evidence base for regulatory complaints.

This repository provides two service components that run the Netzbremse
measurement on a schedule and present the results through a lightweight
dashboard:

| Component | Description |
|-----------|-------------|
| **measurement** | Headless browser worker that runs the Netzbremse speed test and writes results to PostgreSQL |
| **dashboard** | Static web UI and JSON API serving measurement data from PostgreSQL |

## Repository layout

```
cmd/measurement/          Measurement worker entrypoint
cmd/dashboard/            Dashboard server entrypoint
cmd/dashboard/static/     Embedded HTML, CSS, and JS served by the dashboard
internal/                 Shared application code (config, model, postgres)
scripts/                  Puppeteer-based browser runner
db/                       SQL schema
.woodpecker/              CI and release pipelines
```

## Configuration

### Database

| Variable | Description |
|----------|-------------|
| `NETZBREMSE_DATABASE_URL` | PostgreSQL connection URI (preferred) |
| `DATABASE_URL` | Fallback if the above is not set |

### Measurement

| Variable | Description |
|----------|-------------|
| `NETZBREMSE_MEASUREMENT_INTERVAL` | Time between runs (e.g. `1h`) |
| `NETZBREMSE_ENDPOINT` | Target URL (default: `https://netzbremse.de/speed`) |
| `NETZBREMSE_SPEEDTEST_COMMAND` | Override the browser runner command |
| `NETZBREMSE_SPEEDTEST_TIMEOUT` | Per-run timeout (e.g. `4m`) |
| `NETZBREMSE_IMPORT_DIR` | Directory to watch for legacy JSON result files |

### Dashboard

| Variable | Description |
|----------|-------------|
| `NETZBREMSE_DASHBOARD_LIMIT` | Max rows returned by the measurements API |

## Local verification

```bash
go test ./...
node --check scripts/netzbremse-browser.mjs
docker build -f Dockerfile.measurement .
docker build -f Dockerfile.dashboard .
```

## CI and release

`https://git.m0sh1.cc/m0sh1/netzbremse` is the source-of-truth repository.
GitHub is a push mirror for GHCR and GitHub-side automation. Pushes to `main`
run tests and `semantic-release` against Forgejo. Tags (`v*`) trigger the full
release pipeline:

1. Build release images for `linux/amd64`
2. Push to `ghcr.io/sm-moshi/netzbremse-{measurement,dashboard}`
3. Trivy scan, Cosign sign, SPDX SBOM + vuln attestations
4. Mirror to `harbor.m0sh1.cc/apps/` with Cosign + Notation signatures
5. GitHub Actions generate CodeQL analysis and build provenance attestations

For the live GitOps deployment, `ghcr.io` is the source of truth. Harbor keeps a
signed replica for storage and downstream reuse, but the cluster should consume
`ghcr.io/sm-moshi/netzbremse-*` images.

## Licence

This project builds on the open measurement infrastructure published by
[AKVorrat](https://akvorrat.at). The PostgreSQL backend, dashboard, and CI
pipeline are original work for [m0sh1.cc](https://m0sh1.cc).
