# OpenTelemetry Ecosystem Explorer: Go Instrumentation 🔭

Go implementation of the [OpenTelemetry Ecosystem Explorer](https://github.com/open-telemetry/opentelemetry-java-instrumentation/blob/main/docs/contributing/documenting-instrumentation.md) documentation generator.

Generates structured YAML documentation for OpenTelemetry Golang instrumentation libraries using static analysis (Go AST). Currently covers **opentelemetry-go-contrib** and **opentelemetry-go** libraries.

**Output**: `instrumentation-list.yaml` - Complete catalog of instrumentation libraries with their telemetry (spans/metrics), attributes, and semantic conventions.

## Example Output

```yaml
- repository: opentelemetry-go-contrib
  name: otelgin
  display_name: Gin
  description: Package otelgin instruments the github.com/gin-gonic/gin package.
  semantic_conventions:
    - HTTP_SERVER_SPANS
    - HTTP_SERVER_METRICS
  library_link: https://pkg.go.dev/github.com/gin-gonic/gin
  source_path: instrumentation/github.com/gin-gonic/gin/otelgin
  minimum_go_version: 1.24.0
  scope:
    name: go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin
  target_versions:
    library: v1.11.0
  telemetry:
    - when: default
      spans:
        - kind: SERVER
          attributes:
            - name: http.request.method
              type: STRING
            - name: http.response.status_code
              type: LONG
            - name: http.route
              type: STRING
      metrics:
        - name: http.server.response.body.size
          type: HISTOGRAM
          unit: By
          attributes:
            - name: http.request.method
              type: STRING
            - name: http.response.status_code
              type: LONG
            - name: http.route
              type: STRING
```

## Usage

```bash
make dev  # Clone repos and generate instrumentation-list.yaml
```

## How It Works

1. Clones upstream `opentelemetry-go*` repos to `.repo/`
2. Discovers all instrumentation packages via go.mod files
3. Extracts telemetry using Go AST static analysis
4. Maps semantic conventions (`HTTP_SERVER_SPANS`, `DATABASE_CLIENT_SPANS`, etc.)
5. Generates machine-readable insturmention.yaml schema

## Development

```bash
❯ make
  help         ❓ Makefile commands
  clean        🧹 Cleanup build artifacts
  dev          🚀 Start development server
  lint         🧹 Run linter checks
  fmt          ✨ Format code
  tidy         📚 Tidy modules
  docs         📖 Godocs
  test         🧪 Run all tests
  test-perf    ⚡  Run benchmark tests
  vuln         🛡️ Scan for vulnerabilities
  pre-commit   ✅ Run all checks
```

## Roadmap

- [x] opentelemetry-go-contrib instrumentation
- [ ] opentelemetry-go core libraries
- [ ] Configuration documentation (With* options)
- [ ] Integration with OTel Ecosystem Explorer

## Related Projects

- [OpenTelemetry Java Instrumentation](https://github.com/open-telemetry/opentelemetry-java-instrumentation) - Java implementation
- [OpenTelemetry Go Contrib](https://github.com/open-telemetry/opentelemetry-go-contrib) - Instrumentation source

## License

Apache 2.0
