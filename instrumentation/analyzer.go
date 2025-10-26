package instrumentation

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/packages"
)

// AnalyzePackage performs static analysis on an instrumentation package.
func AnalyzePackage(pkgPath string) (*PackageAnalysis, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
		Dir: pkgPath,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return nil, nil
	}

	pkg := pkgs[0]
	analysis := &PackageAnalysis{
		Name: pkg.Name,
	}

	// Extract package documentation
	for _, file := range pkg.Syntax {
		if file.Doc != nil && file.Doc.Text() != "" {
			analysis.Description = strings.TrimSpace(file.Doc.Text())
			break
		}
	}

	// Extract semantic conventions from imports
	rawConventions := extractSemanticConventions(pkg)
	analysis.SemanticConventions = mapSemanticConventions(rawConventions, pkg.PkgPath)

	// Extract telemetry (spans, metrics) from tracer/meter usage
	analysis.Telemetry = extractTelemetry(pkg)

	return analysis, nil
}

type PackageAnalysis struct {
	Name                string
	Description         string
	SemanticConventions []string
	Telemetry           []Telemetry
}

func extractSemanticConventions(pkg *packages.Package) []string {
	var conventions []string
	seen := make(map[string]bool)

	for _, imp := range pkg.Imports {
		// Look for semconv imports
		if strings.Contains(imp.PkgPath, "semconv") {
			if !seen[imp.PkgPath] {
				conventions = append(conventions, imp.PkgPath)
				seen[imp.PkgPath] = true
			}
		}
	}

	return conventions
}

func mapSemanticConventions(rawConventions []string, pkgPath string) []string {
	var mapped []string
	seen := make(map[string]bool)

	for _, raw := range rawConventions {
		conventions := inferConventionsFromImport(raw, pkgPath)
		for _, conv := range conventions {
			if !seen[conv] {
				mapped = append(mapped, conv)
				seen[conv] = true
			}
		}
	}

	if len(mapped) == 0 {
		return rawConventions
	}

	return mapped
}

func inferConventionsFromImport(importPath string, pkgPath string) []string {
	var conventions []string

	pkgLower := strings.ToLower(pkgPath)

	if strings.Contains(pkgLower, "http") ||
	   strings.Contains(pkgLower, "gin") ||
	   strings.Contains(pkgLower, "echo") ||
	   strings.Contains(pkgLower, "mux") ||
	   strings.Contains(pkgLower, "restful") {
		if strings.Contains(pkgPath, "otelhttp") || strings.Contains(pkgPath, "httptrace") {
			conventions = append(conventions, "HTTP_CLIENT_SPANS")
		} else {
			conventions = append(conventions, "HTTP_SERVER_SPANS")
		}
		conventions = append(conventions, "HTTP_SERVER_METRICS")
	}

	if strings.Contains(pkgLower, "grpc") {
		conventions = append(conventions, "RPC_SERVER_SPANS")
		conventions = append(conventions, "RPC_CLIENT_SPANS")
	}

	if strings.Contains(pkgLower, "mongo") || strings.Contains(pkgLower, "database") || strings.Contains(pkgLower, "sql") {
		conventions = append(conventions, "DATABASE_CLIENT_SPANS")
	}

	if strings.Contains(pkgLower, "kafka") || strings.Contains(pkgLower, "messaging") {
		conventions = append(conventions, "MESSAGING_CLIENT_SPANS")
	}

	if strings.Contains(pkgLower, "aws") || strings.Contains(pkgLower, "lambda") {
		conventions = append(conventions, "FAAS_SPANS")
	}

	if len(conventions) == 0 {
		return []string{importPath}
	}

	return conventions
}

func extractTelemetry(pkg *packages.Package) []Telemetry {
	spans := extractSpans(pkg)
	metrics := extractMetrics(pkg)

	if len(spans) == 0 && len(metrics) == 0 {
		return nil
	}

	return []Telemetry{{
		When:    "default",
		Spans:   spans,
		Metrics: metrics,
	}}
}

func extractSpans(pkg *packages.Package) []Span {
	spanMap := make(map[string]*Span)

	detectedKinds := detectSpanKindsInPackage(pkg)

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			if selExpr.Sel.Name == "Start" && len(callExpr.Args) >= 2 {
				extractSpanFromStart(callExpr, spanMap, pkg.PkgPath, detectedKinds)
			}

			return true
		})
	}

	var spans []Span
	for _, span := range spanMap {
		spans = append(spans, *span)
	}

	return spans
}

func detectSpanKindsInPackage(pkg *packages.Package) map[string]bool {
	kinds := make(map[string]bool)

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			selExpr, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			kindName := selExpr.Sel.Name
			if strings.Contains(kindName, "SpanKindServer") || strings.Contains(kindName, "Server") {
				kinds["SERVER"] = true
			}
			if strings.Contains(kindName, "SpanKindClient") || strings.Contains(kindName, "Client") {
				kinds["CLIENT"] = true
			}
			if strings.Contains(kindName, "SpanKindProducer") {
				kinds["PRODUCER"] = true
			}
			if strings.Contains(kindName, "SpanKindConsumer") {
				kinds["CONSUMER"] = true
			}

			return true
		})
	}

	return kinds
}

func extractSpanFromStart(callExpr *ast.CallExpr, spanMap map[string]*Span, pkgPath string, detectedKinds map[string]bool) {
	var spanKind string
	var attributes []Attribute

	if len(callExpr.Args) >= 3 {
		for i := 2; i < len(callExpr.Args); i++ {
			kind, attrs := parseSpanStartOption(callExpr.Args[i])
			if kind != "" {
				spanKind = kind
			}
			attributes = append(attributes, attrs...)
		}
	}

	if spanKind == "" {
		if detectedKinds["SERVER"] {
			spanKind = "SERVER"
		} else if detectedKinds["CLIENT"] {
			spanKind = "CLIENT"
		} else if detectedKinds["PRODUCER"] {
			spanKind = "PRODUCER"
		} else if detectedKinds["CONSUMER"] {
			spanKind = "CONSUMER"
		} else {
			spanKind = "INTERNAL"
		}
	}

	if _, exists := spanMap[spanKind]; !exists {
		spanMap[spanKind] = &Span{
			Kind:       spanKind,
			Attributes: getStandardAttributesForSpan(spanKind, pkgPath),
		}
	}

	attrMap := make(map[string]bool)
	for _, attr := range spanMap[spanKind].Attributes {
		attrMap[attr.Name] = true
	}

	for _, attr := range attributes {
		if !attrMap[attr.Name] {
			spanMap[spanKind].Attributes = append(spanMap[spanKind].Attributes, attr)
			attrMap[attr.Name] = true
		}
	}
}

func parseSpanStartOption(expr ast.Expr) (string, []Attribute) {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", nil
	}

	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", nil
	}

	if selExpr.Sel.Name == "WithSpanKind" && len(callExpr.Args) > 0 {
		kind := extractSpanKind(callExpr.Args[0])
		return kind, nil
	}

	if selExpr.Sel.Name == "WithAttributes" {
		attrs := extractAttributes(callExpr.Args)
		return "", attrs
	}

	return "", nil
}

func extractSpanKind(expr ast.Expr) string {
	selExpr, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	kindName := selExpr.Sel.Name
	switch {
	case strings.Contains(kindName, "Server"):
		return "SERVER"
	case strings.Contains(kindName, "Client"):
		return "CLIENT"
	case strings.Contains(kindName, "Producer"):
		return "PRODUCER"
	case strings.Contains(kindName, "Consumer"):
		return "CONSUMER"
	case strings.Contains(kindName, "Internal"):
		return "INTERNAL"
	default:
		return "INTERNAL"
	}
}

func extractAttributes(args []ast.Expr) []Attribute {
	var attributes []Attribute

	for _, arg := range args {
		attr := parseAttributeExpr(arg)
		if attr.Name != "" {
			attributes = append(attributes, attr)
		}
	}

	return attributes
}

func parseAttributeExpr(expr ast.Expr) Attribute {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return Attribute{}
	}

	selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return Attribute{}
	}

	if len(callExpr.Args) < 2 {
		return Attribute{}
	}

	keyLit, ok := callExpr.Args[0].(*ast.BasicLit)
	if !ok {
		return Attribute{}
	}

	attrName := strings.Trim(keyLit.Value, `"`)
	attrType := getAttributeType(selExpr.Sel.Name)

	return Attribute{
		Name: attrName,
		Type: attrType,
	}
}

func getAttributeType(funcName string) string {
	switch {
	case strings.Contains(funcName, "String"):
		return "STRING"
	case strings.Contains(funcName, "Int64"), strings.Contains(funcName, "Int"):
		return "LONG"
	case strings.Contains(funcName, "Bool"):
		return "BOOLEAN"
	case strings.Contains(funcName, "Float64"), strings.Contains(funcName, "Float"):
		return "DOUBLE"
	default:
		return "STRING"
	}
}

func extractMetrics(pkg *packages.Package) []Metric {
	metricMap := make(map[string]*Metric)

	// First, look for explicitly created metrics in the code
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			methodName := selExpr.Sel.Name
			isMetricCreation := strings.Contains(methodName, "Counter") ||
				strings.Contains(methodName, "Histogram") ||
				strings.Contains(methodName, "UpDownCounter") ||
				strings.Contains(methodName, "Gauge")

			if isMetricCreation && len(callExpr.Args) >= 1 {
				if lit, ok := callExpr.Args[0].(*ast.BasicLit); ok {
					metricName := strings.Trim(lit.Value, `"`)
					if _, exists := metricMap[metricName]; !exists {
						metricMap[metricName] = &Metric{
							Name: metricName,
							Type: mapMetricType(methodName),
						}
					}
				}
			}

			return true
		})
	}

	// Add standard metrics based on package type and semantic conventions
	standardMetrics := getStandardMetrics(pkg.PkgPath)
	for _, metric := range standardMetrics {
		if _, exists := metricMap[metric.Name]; !exists {
			metricMap[metric.Name] = &metric
		}
	}

	var metrics []Metric
	for _, metric := range metricMap {
		metrics = append(metrics, *metric)
	}

	return metrics
}

func mapMetricType(methodName string) string {
	switch {
	case strings.Contains(methodName, "Counter") && !strings.Contains(methodName, "UpDown"):
		return "COUNTER"
	case strings.Contains(methodName, "Histogram"):
		return "HISTOGRAM"
	case strings.Contains(methodName, "UpDownCounter"):
		return "UPDOWNCOUNTER"
	case strings.Contains(methodName, "Gauge"):
		return "GAUGE"
	default:
		return "COUNTER"
	}
}

func getStandardAttributesForSpan(spanKind string, pkgPath string) []Attribute {
	pkgLower := strings.ToLower(pkgPath)

	if spanKind == "SERVER" && isHTTPPackage(pkgLower) {
		return []Attribute{
			{Name: "http.request.method", Type: "STRING"},
			{Name: "http.response.status_code", Type: "LONG"},
			{Name: "http.route", Type: "STRING"},
			{Name: "server.address", Type: "STRING"},
			{Name: "server.port", Type: "LONG"},
			{Name: "url.scheme", Type: "STRING"},
			{Name: "url.path", Type: "STRING"},
			{Name: "network.protocol.name", Type: "STRING"},
			{Name: "network.protocol.version", Type: "STRING"},
			{Name: "user_agent.original", Type: "STRING"},
			{Name: "client.address", Type: "STRING"},
			{Name: "network.peer.address", Type: "STRING"},
		}
	}

	if spanKind == "CLIENT" && isHTTPPackage(pkgLower) {
		return []Attribute{
			{Name: "http.request.method", Type: "STRING"},
			{Name: "http.response.status_code", Type: "LONG"},
			{Name: "server.address", Type: "STRING"},
			{Name: "server.port", Type: "LONG"},
			{Name: "url.full", Type: "STRING"},
			{Name: "network.protocol.name", Type: "STRING"},
			{Name: "network.protocol.version", Type: "STRING"},
		}
	}

	if spanKind == "CLIENT" && isDatabasePackage(pkgLower) {
		return []Attribute{
			{Name: "db.system", Type: "STRING"},
			{Name: "db.operation.name", Type: "STRING"},
			{Name: "db.collection.name", Type: "STRING"},
			{Name: "db.query.text", Type: "STRING"},
			{Name: "server.address", Type: "STRING"},
			{Name: "server.port", Type: "LONG"},
		}
	}

	if (spanKind == "SERVER" || spanKind == "CLIENT") && isRPCPackage(pkgLower) {
		return []Attribute{
			{Name: "rpc.system", Type: "STRING"},
			{Name: "rpc.service", Type: "STRING"},
			{Name: "rpc.method", Type: "STRING"},
			{Name: "server.address", Type: "STRING"},
			{Name: "server.port", Type: "LONG"},
		}
	}

	if spanKind == "SERVER" && isLambdaPackage(pkgLower) {
		return []Attribute{
			{Name: "faas.invocation_id", Type: "STRING"},
			{Name: "cloud.resource_id", Type: "STRING"},
		}
	}

	if spanKind == "CLIENT" && isAWSPackage(pkgLower) {
		return []Attribute{
			{Name: "rpc.system", Type: "STRING"},
			{Name: "rpc.service", Type: "STRING"},
			{Name: "rpc.method", Type: "STRING"},
			{Name: "server.address", Type: "STRING"},
			{Name: "server.port", Type: "LONG"},
		}
	}

	return nil
}

func isAWSPackage(pkgPath string) bool {
	return strings.Contains(pkgPath, "aws")
}

func isHTTPPackage(pkgPath string) bool {
	return strings.Contains(pkgPath, "http") ||
		strings.Contains(pkgPath, "gin") ||
		strings.Contains(pkgPath, "echo") ||
		strings.Contains(pkgPath, "mux") ||
		strings.Contains(pkgPath, "restful")
}

func isDatabasePackage(pkgPath string) bool {
	return strings.Contains(pkgPath, "mongo") ||
		strings.Contains(pkgPath, "database") ||
		strings.Contains(pkgPath, "sql")
}

func isRPCPackage(pkgPath string) bool {
	return strings.Contains(pkgPath, "grpc")
}

func isLambdaPackage(pkgPath string) bool {
	return strings.Contains(pkgPath, "lambda")
}

func getStandardMetrics(pkgPath string) []Metric {
	pkgLower := strings.ToLower(pkgPath)

	if isHTTPPackage(pkgLower) {
		return []Metric{
			{
				Name: "http.server.request.duration",
				Type: "HISTOGRAM",
				Unit: "s",
				Attributes: []Attribute{
					{Name: "http.request.method", Type: "STRING"},
					{Name: "http.response.status_code", Type: "LONG"},
					{Name: "http.route", Type: "STRING"},
					{Name: "network.protocol.version", Type: "STRING"},
					{Name: "url.scheme", Type: "STRING"},
				},
			},
			{
				Name: "http.server.request.body.size",
				Type: "HISTOGRAM",
				Unit: "By",
				Attributes: []Attribute{
					{Name: "http.request.method", Type: "STRING"},
					{Name: "http.response.status_code", Type: "LONG"},
					{Name: "http.route", Type: "STRING"},
					{Name: "network.protocol.version", Type: "STRING"},
					{Name: "url.scheme", Type: "STRING"},
				},
			},
			{
				Name: "http.server.response.body.size",
				Type: "HISTOGRAM",
				Unit: "By",
				Attributes: []Attribute{
					{Name: "http.request.method", Type: "STRING"},
					{Name: "http.response.status_code", Type: "LONG"},
					{Name: "http.route", Type: "STRING"},
					{Name: "network.protocol.version", Type: "STRING"},
					{Name: "url.scheme", Type: "STRING"},
				},
			},
		}
	}

	if isRPCPackage(pkgLower) {
		return []Metric{
			{
				Name: "rpc.server.duration",
				Type: "HISTOGRAM",
				Unit: "ms",
				Attributes: []Attribute{
					{Name: "rpc.method", Type: "STRING"},
					{Name: "rpc.service", Type: "STRING"},
					{Name: "rpc.system", Type: "STRING"},
				},
			},
			{
				Name: "rpc.server.request.size",
				Type: "HISTOGRAM",
				Unit: "By",
				Attributes: []Attribute{
					{Name: "rpc.method", Type: "STRING"},
					{Name: "rpc.service", Type: "STRING"},
					{Name: "rpc.system", Type: "STRING"},
				},
			},
			{
				Name: "rpc.server.response.size",
				Type: "HISTOGRAM",
				Unit: "By",
				Attributes: []Attribute{
					{Name: "rpc.method", Type: "STRING"},
					{Name: "rpc.service", Type: "STRING"},
					{Name: "rpc.system", Type: "STRING"},
				},
			},
		}
	}

	return nil
}
