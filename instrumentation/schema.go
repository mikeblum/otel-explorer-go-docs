package instrumentation

type Library struct {
	Name                string          `yaml:"name"`
	Description         string          `yaml:"description,omitempty"`
	SemanticConventions []string        `yaml:"semantic_conventions,omitempty"`
	LibraryLink         string          `yaml:"library_link,omitempty"`
	SourcePath          string          `yaml:"source_path"`
	Scope               Scope           `yaml:"scope"`
	TargetVersions      TargetVersions  `yaml:"target_versions"`
	Configurations      []Configuration `yaml:"configurations,omitempty"`
	Telemetry           []Telemetry     `yaml:"telemetry,omitempty"`
}

type Scope struct {
	Name string `yaml:"name"`
}

type TargetVersions struct {
	Library string `yaml:"library"`
}

type Configuration struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	Default     string `yaml:"default,omitempty"`
}

type Telemetry struct {
	When    string   `yaml:"when"`
	Spans   []Span   `yaml:"spans,omitempty"`
	Metrics []Metric `yaml:"metrics,omitempty"`
}

type Span struct {
	Kind       string      `yaml:"kind"`
	Attributes []Attribute `yaml:"attributes,omitempty"`
}

type Metric struct {
	Name       string      `yaml:"name"`
	Type       string      `yaml:"type"`
	Unit       string      `yaml:"unit,omitempty"`
	Attributes []Attribute `yaml:"attributes,omitempty"`
}

type Attribute struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}
