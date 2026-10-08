package openapidocs

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// parseDescription reads an OpenAPI 3 description in YAML or JSON (JSON is YAML). It refuses
// what is not one, saying why, so a stray file that matched the input patterns is reported
// rather than rendered as an empty API.
func parseDescription(data []byte) (description, error) {
	var parsed description
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return description{}, fmt.Errorf("not valid YAML or JSON: %w", err)
	}
	switch {
	case parsed.Swagger != "":
		return description{}, fmt.Errorf("this is a Swagger %s description, which is not supported; convert it to OpenAPI 3", parsed.Swagger)
	case parsed.OpenAPI == "":
		return description{}, errors.New("not an OpenAPI description: there is no openapi field")
	case !strings.HasPrefix(parsed.OpenAPI, "3."):
		return description{}, fmt.Errorf("OpenAPI %s is not supported; only OpenAPI 3.x is", parsed.OpenAPI)
	}

	return parsed, nil
}
