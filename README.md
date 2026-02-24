# Automadoist

Automatic task management for Todoist. Traverses your project tree and manages labels on actionable tasks so your filters stay up to date without manual effort.

## What it does

Automadoist provides three commands:

- **`next_items`** — Walks your project hierarchy breadth-first, finds actionable leaf tasks, and adds/removes a `@next` label. Tasks that are no longer actionable get pruned automatically.
- **`reviews`** — Finds tasks matching configurable prefixes (e.g., `*review project X`) and manages a `@review` label so review tasks surface in your filters.
- **`default_tags`** — Interactive TUI for assigning default labels to projects. When a task first becomes actionable, it inherits its project's default tags.

Run it on a cron (every 15 minutes works well) and your Todoist filters stay current without you thinking about it.

## How it works

### Traversal

Starting from a configurable root project (`entry_point`), Automadoist collects all subprojects recursively. For each project, it evaluates tasks breadth-first:

- **Leaf tasks** (no subtasks) are candidates for the `@next` label
- **Parent tasks** expose their children for further evaluation
- **Sequential parents** (name ends with `!`) only expose the *last* child — the next thing to do
- **Parallel parents** (default) expose all children

### Filtering

Tasks are filtered out if they:
- Start with a skip prefix (default: `*`)
- Have a future deadline (configurable)
- Carry an ignore label (default: `@waiting`, `@review`)

### Context preservation

When a task loses its `@next` status, Automadoist can save its labels and priority as a context comment. When the task becomes actionable again, saved context is restored — preserving any manual customizations you made.

## Installation

### Go install

```bash
go install github.com/harlequix/automadoist@latest
```

### Build from source

```bash
git clone https://github.com/harlequix/automadoist.git
cd automadoist
go build -o automadoist .
```

### Docker

```bash
git clone https://github.com/harlequix/automadoist.git
cd automadoist
docker compose up godoist        # one-shot run
docker compose up cron           # cron every 15 minutes
```

## Configuration

Automadoist loads configuration from three sources (in priority order):

1. **CLI flags** (`--token`, `--config`, `--debug`, `--log-level`)
2. **YAML config file** (path from `--config`)
3. **Environment variables** (`GODOIST_` prefix, `_` maps to `.` for nesting)

Copy the example config to get started:

```bash
cp config.example.yaml config.yaml
```

See [`config.example.yaml`](config.example.yaml) for all options with descriptions, and [`config.schema.json`](config.schema.json) for the full JSON schema.

### Minimal config

```yaml
token: "your-todoist-api-token"
```

Everything else has sensible defaults. The token can also be set via the `GODOIST_TOKEN` environment variable.

## Usage

```bash
# Run next_items with a config file
automadoist --config config.yaml next_items

# Run reviews
automadoist --config config.yaml reviews

# Use env var for token
export GODOIST_TOKEN="your-token"
automadoist next_items

# Pass token directly
automadoist --token "your-token" next_items

# Debug mode
automadoist --debug --config config.yaml next_items

# Interactive default tags configurator
automadoist --config config.yaml default_tags
```

## Docker

The included `compose.yml` supports two modes:

**One-shot** — run `next_items` once:
```bash
docker compose up godoist
```

**Cron** — run every 15 minutes:
```bash
docker compose up cron
```

Both modes read `config.yaml` from the project root. Set `GODOIST_TOKEN` in a `.env` file or export it in your shell.

## License

[MIT](LICENSE)
