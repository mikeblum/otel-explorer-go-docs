package instrumentation

import (
	"log/slog"

	"gopkg.in/yaml.v3"
)

type SpanKind string

const (
	SpanKindServer   SpanKind = "SERVER"
	SpanKindClient   SpanKind = "CLIENT"
	SpanKindProducer SpanKind = "PRODUCER"
	SpanKindConsumer SpanKind = "CONSUMER"
	SpanKindInternal SpanKind = "INTERNAL"
)

type MetricType string

const (
	MetricTypeCounter       MetricType = "COUNTER"
	MetricTypeHistogram     MetricType = "HISTOGRAM"
	MetricTypeUpDownCounter MetricType = "UPDOWNCOUNTER"
	MetricTypeGauge         MetricType = "GAUGE"
)

type AttributeType string

const (
	AttributeTypeString  AttributeType = "STRING"
	AttributeTypeLong    AttributeType = "LONG"
	AttributeTypeBoolean AttributeType = "BOOLEAN"
	AttributeTypeDouble  AttributeType = "DOUBLE"
)

type Library struct {
	Repository          string          `yaml:"repository"`
	Name                string          `yaml:"name"`
	DisplayName         string          `yaml:"display_name,omitempty"`
	Description         string          `yaml:"description,omitempty"`
	SemanticConventions []string        `yaml:"semantic_conventions,omitempty"`
	LibraryLink         string          `yaml:"library_link,omitempty"`
	SourcePath          string          `yaml:"source_path"`
	MinimumGoVersion    string          `yaml:"minimum_go_version,omitempty"`
	Scope               Scope           `yaml:"scope"`
	TargetVersions      *TargetVersions `yaml:"target_versions,omitempty"`
	Configurations      []Configuration `yaml:"configurations,omitempty"`
	Telemetry           []Telemetry     `yaml:"telemetry,omitempty"`
}

type Scope struct {
	Name string `yaml:"name"`
}

type TargetVersions struct {
	Library string `yaml:"library,omitempty"`
}

type Configuration struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	Default     string `yaml:"default,omitempty"`
}

type Telemetry struct {
	When    string   `yaml:"when,omitempty"`
	Spans   []Span   `yaml:"spans,omitempty"`
	Metrics []Metric `yaml:"metrics,omitempty"`
}

type Span struct {
	Kind       SpanKind    `yaml:"kind,omitempty"`
	Attributes []Attribute `yaml:"attributes,omitempty"`
}

type Metric struct {
	Name       string      `yaml:"name"`
	Type       MetricType  `yaml:"type"`
	Unit       string      `yaml:"unit,omitempty"`
	Attributes []Attribute `yaml:"attributes,omitempty"`
}

type Attribute struct {
	Name string        `yaml:"name"`
	Type AttributeType `yaml:"type"`
}

func (a Attribute) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "name"},
			{Kind: yaml.ScalarNode, Value: a.Name},
			{Kind: yaml.ScalarNode, Value: "type"},
			{Kind: yaml.ScalarNode, Value: string(a.Type)},
		},
	}
	return node, nil
}

type Stats struct {
	LibrariesWithTelemetry           int
	LibrariesWithSemanticConventions int
	TotalSpans                       int
	TotalMetrics                     int
	TotalAttributes                  int
	SpansByKind                      map[SpanKind]int
	MetricsByType                    map[MetricType]int
}

func (s Stats) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("libraries", s.LibrariesWithTelemetry),
		slog.Int("semconv", s.LibrariesWithSemanticConventions),
		slog.Int("spans", s.TotalSpans),
		slog.Int("metrics", s.TotalMetrics),
		slog.Int("attributes", s.TotalAttributes),
		slog.Int("server", s.SpansByKind[SpanKindServer]),
		slog.Int("client", s.SpansByKind[SpanKindClient]),
		slog.Int("internal", s.SpansByKind[SpanKindInternal]),
	)
}

func CalculateStats(librariesByRepo map[string][]Library) map[string]Stats {
	repoStats := make(map[string]Stats)

	for repoName, libraries := range librariesByRepo {
		stats := Stats{
			SpansByKind:   make(map[SpanKind]int),
			MetricsByType: make(map[MetricType]int),
		}

		for _, lib := range libraries {
			if len(lib.Telemetry) > 0 {
				stats.LibrariesWithTelemetry++
			}

			if len(lib.SemanticConventions) > 0 {
				stats.LibrariesWithSemanticConventions++
			}

			for _, tel := range lib.Telemetry {
				for _, span := range tel.Spans {
					stats.TotalSpans++
					if span.Kind != "" {
						stats.SpansByKind[span.Kind]++
					}
					stats.TotalAttributes += len(span.Attributes)
				}

				for _, metric := range tel.Metrics {
					stats.TotalMetrics++
					if metric.Type != "" {
						stats.MetricsByType[metric.Type]++
					}
					stats.TotalAttributes += len(metric.Attributes)
				}
			}
		}

		repoStats[repoName] = stats
	}

	return repoStats
}
