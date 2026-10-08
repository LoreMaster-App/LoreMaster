package openapidocs

import (
	"gopkg.in/yaml.v3"
)

// ordered is a mapping that remembers the order its keys were written in, which YAML maps
// in Go do not: an API reads best in the order its author chose.
type ordered[T any] struct {
	Keys   []string
	Values map[string]T
}

// UnmarshalYAML reads a mapping node entry by entry.
func (o *ordered[T]) UnmarshalYAML(node *yaml.Node) error {
	o.Keys, o.Values = nil, map[string]T{}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		var value T
		if err := node.Content[i+1].Decode(&value); err != nil {
			return err
		}
		key := node.Content[i].Value
		if _, repeated := o.Values[key]; !repeated {
			o.Keys = append(o.Keys, key)
		}
		o.Values[key] = value
	}

	return nil
}

// each calls visit for every entry in the order it was written.
func (o ordered[T]) each(visit func(key string, value T)) {
	for _, key := range o.Keys {
		visit(key, o.Values[key])
	}
}

// names is what a schema's "type" says: OpenAPI 3.0 writes one name, 3.1 may write a list
// such as [string, "null"].
type names []string

// UnmarshalYAML accepts a scalar or a sequence.
func (n *names) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		*n = names{node.Value}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			*n = append(*n, item.Value)
		}
	}

	return nil
}

// description is the part of an OpenAPI 3 document the generator reads.
type description struct {
	OpenAPI string `yaml:"openapi"`
	Swagger string `yaml:"swagger"`
	Info    struct {
		Title       string `yaml:"title"`
		Version     string `yaml:"version"`
		Description string `yaml:"description"`
	} `yaml:"info"`
	Servers []struct {
		URL         string `yaml:"url"`
		Description string `yaml:"description"`
	} `yaml:"servers"`
	Tags []struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	} `yaml:"tags"`
	Paths      ordered[pathItem] `yaml:"paths"`
	Components struct {
		Schemas       ordered[*schema]     `yaml:"schemas"`
		Parameters    ordered[parameter]   `yaml:"parameters"`
		Responses     ordered[response]    `yaml:"responses"`
		RequestBodies ordered[requestBody] `yaml:"requestBodies"`
	} `yaml:"components"`
}

// pathItem is one path with the operations it offers.
type pathItem struct {
	Summary     string      `yaml:"summary"`
	Description string      `yaml:"description"`
	Parameters  []parameter `yaml:"parameters"`
	Get         *operation  `yaml:"get"`
	Put         *operation  `yaml:"put"`
	Post        *operation  `yaml:"post"`
	Delete      *operation  `yaml:"delete"`
	Options     *operation  `yaml:"options"`
	Head        *operation  `yaml:"head"`
	Patch       *operation  `yaml:"patch"`
	Trace       *operation  `yaml:"trace"`
}

// operations lists the path's operations in the conventional order, with their methods.
func (p pathItem) operations() []methodOperation {
	var found []methodOperation
	for _, candidate := range []methodOperation{
		{"GET", p.Get}, {"PUT", p.Put}, {"POST", p.Post}, {"DELETE", p.Delete},
		{"OPTIONS", p.Options}, {"HEAD", p.Head}, {"PATCH", p.Patch}, {"TRACE", p.Trace},
	} {
		if candidate.operation != nil {
			found = append(found, candidate)
		}
	}

	return found
}

type methodOperation struct {
	method    string
	operation *operation
}

type operation struct {
	Tags        []string          `yaml:"tags"`
	Summary     string            `yaml:"summary"`
	Description string            `yaml:"description"`
	OperationID string            `yaml:"operationId"`
	Deprecated  bool              `yaml:"deprecated"`
	Parameters  []parameter       `yaml:"parameters"`
	RequestBody *requestBody      `yaml:"requestBody"`
	Responses   ordered[response] `yaml:"responses"`
}

type parameter struct {
	Ref         string  `yaml:"$ref"`
	Name        string  `yaml:"name"`
	In          string  `yaml:"in"`
	Description string  `yaml:"description"`
	Required    bool    `yaml:"required"`
	Deprecated  bool    `yaml:"deprecated"`
	Schema      *schema `yaml:"schema"`
}

type requestBody struct {
	Ref         string             `yaml:"$ref"`
	Description string             `yaml:"description"`
	Required    bool               `yaml:"required"`
	Content     ordered[mediaType] `yaml:"content"`
}

type response struct {
	Ref         string             `yaml:"$ref"`
	Description string             `yaml:"description"`
	Content     ordered[mediaType] `yaml:"content"`
}

type mediaType struct {
	Schema *schema `yaml:"schema"`
}

// schema is a JSON Schema as OpenAPI uses it, as far as documentation needs it.
type schema struct {
	Ref         string           `yaml:"$ref"`
	Type        names            `yaml:"type"`
	Format      string           `yaml:"format"`
	Title       string           `yaml:"title"`
	Description string           `yaml:"description"`
	Enum        []any            `yaml:"enum"`
	Default     any              `yaml:"default"`
	Nullable    bool             `yaml:"nullable"`
	Deprecated  bool             `yaml:"deprecated"`
	Required    []string         `yaml:"required"`
	Properties  ordered[*schema] `yaml:"properties"`
	Items       *schema          `yaml:"items"`
	AllOf       []*schema        `yaml:"allOf"`
	OneOf       []*schema        `yaml:"oneOf"`
	AnyOf       []*schema        `yaml:"anyOf"`
}
