package instrumentation

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mikeblum/otel-explorer-go-docs/repo"
	"gopkg.in/yaml.v3"
)

func encodeYAMLFile(path string, data interface{}) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	defer encoder.Close()

	return encoder.Encode(data)
}

func Generate(groups []Group) error {
	registryDir := "registry"
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		return err
	}

	signalsPath := filepath.Join(registryDir, "signals.yaml")
	signalsOutput := map[string]interface{}{
		"groups": groups,
	}
	if err := encodeYAMLFile(signalsPath, signalsOutput); err != nil {
		return err
	}

	customAttrs := extractAttributeGroups(groups)
	if len(customAttrs) > 0 {
		attributesPath := filepath.Join(registryDir, "attributes.yaml")
		attributesOutput := map[string]interface{}{
			"groups": []AttributeGroup{
				{
					ID:         "registry.otel.go",
					Type:       "attribute_group",
					Name:       "OpenTelemetry Go Instrumentation Attributes",
					Brief:      "Custom attributes used in OpenTelemetry Go instrumentation",
					Attributes: customAttrs,
				},
			},
		}
		if err := encodeYAMLFile(attributesPath, attributesOutput); err != nil {
			return err
		}
	}

	return nil
}

func extractAttributeGroups(groups []Group) []AttributeDef {
	attributeMap := make(map[string]AttributeDef)

	for _, group := range groups {
		for _, attrRef := range group.Attributes {
			if _, ok := GetSemconvAttribute(attrRef.Ref); ok {
				continue
			}

			if _, exists := attributeMap[attrRef.Ref]; !exists {
				brief := generateAttributeBrief(attrRef.Ref)
				attrType := inferAttributeType(attrRef.Ref)

				attr := AttributeDef{
					ID:        attrRef.Ref,
					Type:      attrType,
					Brief:     brief,
					Stability: StabilityDevelopment,
				}

				if attrType == AttributeTypeString {
					attr.Examples = []interface{}{attrRef.Ref}
				}

				attributeMap[attrRef.Ref] = attr
			}
		}
	}

	var attrs []AttributeDef
	for _, attr := range attributeMap {
		attrs = append(attrs, attr)
	}

	return attrs
}

func inferAttributeType(attrName string) AttributeType {
	if strings.Contains(attrName, "port") || strings.Contains(attrName, "status_code") {
		return AttributeTypeLong
	}
	if strings.Contains(attrName, "duration") {
		return AttributeTypeDouble
	}
	return AttributeTypeString
}

func generateAttributeBrief(attrName string) string {
	// First check official semantic conventions
	if attr, ok := GetSemconvAttribute(attrName); ok && attr.Brief != "" {
		return attr.Brief
	}

	// Fall back to smart casing
	wellKnownTerms := map[string]string{
		"rpc":         "RPC",
		"grpc":        "gRPC",
		"http":        "HTTP",
		"https":       "HTTPS",
		"url":         "URL",
		"uri":         "URI",
		"aws":         "AWS",
		"db":          "Database",
		"sql":         "SQL",
		"nosql":       "NoSQL",
		"mongodb":     "MongoDB",
		"id":          "ID",
		"api":         "API",
		"tcp":         "TCP",
		"udp":         "UDP",
		"ip":          "IP",
		"dns":         "DNS",
		"tls":         "TLS",
		"ssl":         "SSL",
		"faas":        "FaaS",
		"k8s":         "Kubernetes",
		"os":          "OS",
		"cpu":         "CPU",
		"io":          "I/O",
		"sdk":         "SDK",
		"json":        "JSON",
		"xml":         "XML",
		"yaml":        "YAML",
		"status_code": "status code",
	}

	parts := strings.Split(attrName, ".")
	var result []string

	for _, part := range parts {
		part = strings.ReplaceAll(part, "_", " ")

		if replacement, ok := wellKnownTerms[strings.ToLower(part)]; ok {
			result = append(result, replacement)
		} else {
			subParts := strings.Split(part, " ")
			for i, subPart := range subParts {
				if replacement, ok := wellKnownTerms[strings.ToLower(subPart)]; ok {
					subParts[i] = replacement
				} else {
					subParts[i] = strings.Title(subPart)
				}
			}
			result = append(result, strings.Join(subParts, " "))
		}
	}

	return strings.Join(result, " ")
}

func Scan(repoName, repoPath string) ([]Group, error) {
	var scanPaths []string

	switch repoName {
	case repo.RepoContrib:
		scanPaths = []string{filepath.Join(repoPath, "instrumentation")}
	case repo.RepoGo:
		scanPaths = []string{repoPath}
	default:
		scanPaths = []string{filepath.Join(repoPath, "instrumentation")}
	}

	groupMap := make(map[string]*Group)
	for _, scanPath := range scanPaths {
		packages, err := Walk(scanPath)
		if err != nil {
			continue
		}

		for _, pkg := range packages {
			pkgGroups, err := Parse(pkg.GoModPath, repoPath, repoName)
			if err != nil {
				continue
			}
			for _, group := range pkgGroups {
				if existing, ok := groupMap[group.ID]; ok {
					attrMap := make(map[string]bool)
					for _, attr := range existing.Attributes {
						attrMap[attr.Ref] = true
					}
					for _, attr := range group.Attributes {
						if !attrMap[attr.Ref] {
							existing.Attributes = append(existing.Attributes, attr)
						}
					}
				} else {
					groupCopy := group
					groupMap[group.ID] = &groupCopy
				}
			}
		}
	}

	var groups []Group
	for _, group := range groupMap {
		groups = append(groups, *group)
	}

	return groups, nil
}
