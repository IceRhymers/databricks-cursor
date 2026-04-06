# databricks-cursor

Standalone OAuth proxy for [Cursor](https://cursor.sh) that injects fresh Databricks OAuth tokens into inference requests. Run it once, point Cursor at it, and use Databricks-hosted models without managing API keys.

## Install

```bash
go install github.com/IceRhymers/databricks-cursor@latest
```

Or build from source:

```bash
git clone https://github.com/IceRhymers/databricks-cursor.git
cd databricks-cursor
make build
```

## Usage

```bash
databricks-cursor
```

On first run, the proxy will:
1. Trigger browser-based Databricks OAuth if not already authenticated
2. Auto-assign a port and save it for future runs
3. Print setup instructions for Cursor

### Cursor setup (one-time)

1. Open Cursor Settings > Models
2. Set "Override OpenAI Base URL" to: `http://127.0.0.1:<port>/v1`
3. Set "OpenAI API Key" to any non-empty value (e.g., `databricks-proxy`)

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `0` (auto) | Port to listen on |
| `--profile` | `DEFAULT` | Databricks CLI profile |
| `--verbose` / `-v` | `false` | Enable verbose logging |
| `--version` | | Print version and exit |
| `--print-env` | | Print environment variables |
| `--upstream` | | Override inference upstream URL |
| `--no-otel` | | Disable OpenTelemetry |
| `--otel-logs-table` | | Unity Catalog table for OTEL logs |
| `--log-file` | | Write logs to file |

### Port persistence

The proxy saves its port to `~/.databricks-cursor/config.json`. On subsequent runs with `--port 0` (default), it reuses the saved port so Cursor's configuration stays valid.

If the saved port is already in use, the proxy exits with a clear error message.

## Development

```bash
make test    # run tests
make build   # compile binary
make dist    # clean + build
```

## License

See [LICENSE](LICENSE) for details.
