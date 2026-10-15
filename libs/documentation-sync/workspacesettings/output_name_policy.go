package workspacesettings

import (
	"fmt"
	"strings"
	"unicode"
)

// maxNameLength keeps an output's name short enough for a tree row and a picker.
const maxNameLength = 60

// validateName checks an output's name when it has one: not blank, short, plain text and not
// the name of another output, so a name always says which output it is.
func validateName(add func(string, ...any), where string, output Output, seen map[string]int, index int) {
	if output.Name == "" {
		return
	}
	trimmed := strings.TrimSpace(output.Name)
	switch {
	case trimmed == "":
		add("%s.name is blank; give the output a name or remove the field", where)

		return
	case trimmed != output.Name:
		add("%s.name %q starts or ends with a space", where, output.Name)
	case len([]rune(output.Name)) > maxNameLength:
		add("%s.name is longer than %d characters; use a shorter name", where, maxNameLength)
	case strings.IndexFunc(output.Name, unicode.IsControl) >= 0:
		add("%s.name must not contain control characters", where)
	}
	key := strings.ToLower(trimmed)
	if first, taken := seen[key]; taken {
		add("outputs[%d].name %q is already the name of outputs[%d]; names must be different", index, output.Name, first)

		return
	}
	seen[key] = index
}

// DisplayName is what an output is called: its name, or one made from the platform and where
// it goes. A made-up name shared by two outputs is numbered so each is still told apart.
func DisplayName(outputs []Output, index int) string {
	output := outputs[index]
	if strings.TrimSpace(output.Name) != "" {
		return output.Name
	}
	base := madeUpName(output)
	earlier := 0
	for _, other := range outputs[:index] {
		if strings.TrimSpace(other.Name) == "" && madeUpName(other) == base {
			earlier++
		}
	}
	if earlier == 0 {
		return base
	}

	return fmt.Sprintf("%s (%d)", base, earlier+1)
}

func madeUpName(output Output) string {
	switch output.Platform {
	case "github-pages":
		return "GitHub Pages · " + orDefault(output.Repo, "this repository")
	case "github-wiki":
		return "GitHub Wiki · " + orDefault(output.Repo, "this repository")
	}

	return "Confluence · " + orDefault(output.Space, "no space yet")
}

func orDefault(value string, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}
