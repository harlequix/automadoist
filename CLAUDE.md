# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Automadoist is a Go CLI tool that automates Todoist task management. It syncs with the Todoist API via the `godoist` client library and applies label-based rules to organize tasks. Two commands:

- **`next_items`** — Traverses projects breadth-first, identifies actionable leaf tasks, and manages `next` labels
- **`reviews`** — Finds tasks matching review prefixes and manages `review` labels

## Build & Run

```bash
# Build
go build -o automadoist .

# Run (requires Todoist API token)
./automadoist --token <TOKEN> next_items
./automadoist --token <TOKEN> reviews

# Run with config file
./automadoist --config config.yaml next_items

# Debug mode
./automadoist --debug --token <TOKEN> next_items

# Docker
docker compose up godoist        # one-shot next_items
docker compose up cron           # cron every 15 min

# Tests
go test -v ./...
```

## Linting

Uses [Trunk](https://docs.trunk.io/cli) for linting:

```bash
trunk check        # run all linters
trunk fmt          # auto-format
```

Enabled linters: golangci-lint, gofmt, yamllint, prettier, trufflehog (secret scanning), osv-scanner (vulnerability scanning).

## Architecture

All source is in the root package (`package main`), three files:

- **`godoist.go`** — Entry point. CLI setup (urfave/cli), config loading (Koanf: CLI flags > config file > `GODOIST_*` env vars), logger initialization.
- **`next_items.go`** — `NextItemsConfig` struct and `process_next_items()`. Recursively collects subprojects from an entry point, evaluates tasks breadth-first. Sequential tasks (parent name ends with `!`) only expose the last child; parallel tasks expose all children. Leaf tasks are filtered by skip prefixes, deadlines, and ignore labels.
- **`reviews.go`** — `ReviewsConfig` struct and `reviews()`. Reuses the next-items traversal with adjusted config (removes review prefixes from skip list, removes review label from ignore list) to find review-eligible tasks.

## Key Dependency

The `godoist` library (`github.com/harlequix/godoist`) is the Todoist API client. The `godoist.Todoist` client provides `Projects`, `Tasks`, and `API.Commit()` for batching updates. For local development against the library, add a replace directive:

```
go mod edit -replace github.com/harlequix/godoist=../godoist
```

## Configuration Priority

1. CLI flags (`--token`, `--config`, `--debug`, `--log-level`)
2. YAML config file (path from `--config`)
3. Environment variables (`GODOIST_` prefix, `_` maps to `.` for nesting)
