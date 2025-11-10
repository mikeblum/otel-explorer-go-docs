# OpenTelemetry Ecosystem Explorer: Go Instrumentation 🔭

Generates [OpenTelemetry Weaver](https://github.com/open-telemetry/weaver) registry documentation for Go instrumentation libraries using static analysis (Go AST). Currently covers **opentelemetry-go-contrib** with future support planned for **opentelemetry-go** core libraries.

**Output**: Weaver-format registry in `registry/` directory with signals (spans/metrics) and attributes.

## Quick Start

```bash
# Install weaver CLI
make install

# Generate and validate registry
make dev
```

## Output Format

Generates a Weaver-compatible registry:

```
registry/
├── registry_manifest.yaml  # Registry metadata
├── signals.yaml           # Spans and metrics (542 lines)
└── attributes.yaml        # Deduplicated attributes (50 lines)
```

### Example Signal

```yaml
- id: gin.server.span
  type: span
  stability: development
  brief: Span for gin
  span_kind: SERVER
  attributes:
    - ref: http.request.method
      requirement_level: recommended
    - ref: http.response.status_code
      requirement_level: recommended
```

### Example Attributes

```yaml
- id: registry.otel.go
  type: attribute_group
  display_name: OpenTelemetry Go Instrumentation Attributes
  attributes:
    - id: http.request.method
      type: string
      brief: Http Request Method
      stability: development
```

## How It Works

1. Clones opentelemetry-go-contrib to `.repo/`
2. Discovers instrumentation packages via go.mod files
3. Extracts telemetry using Go AST static analysis
4. Converts to Weaver format (signals.yaml + attributes.yaml)
5. Validates registry with `weaver registry check`

## Commands

```bash
make install         # Install weaver CLI
make dev            # Generate and validate registry
make weaver-check   # Validate registry format
make weaver-resolve # Resolve dependencies
make weaver-stats   # Show registry statistics
make test           # Run tests
make lint           # Run linter
make pre-commit     # Run all checks
```

Tests validate extraction against AWS SDK, Gin, gRPC, MongoDB, and Lambda instrumentation.

## Roadmap

- [x] opentelemetry-go-contrib instrumentation
- [x] Weaver format output (signals.yaml + attributes.yaml)
- [ ] opentelemetry-go core libraries
- [ ] Configuration documentation (With* options)
- [ ] Integration with OTel Ecosystem Explorer

## Related Projects

- [OpenTelemetry Weaver](https://github.com/open-telemetry/weaver) - Schema tooling
- [OpenTelemetry Java Instrumentation](https://github.com/open-telemetry/opentelemetry-java-instrumentation) - Java implementation
- [OpenTelemetry Go Contrib](https://github.com/open-telemetry/opentelemetry-go-contrib) - Instrumentation source

## License

Apache 2.0
