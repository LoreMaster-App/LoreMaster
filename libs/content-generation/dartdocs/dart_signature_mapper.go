package dartdocs

import (
	"strings"
)

// signature writes a declaration the way it is declared in Dart, one line, without its body.
// fieldTypes gives the types of the enclosing class's fields, for "this.x" parameters, which
// carry no type of their own.
func signature(d declaration, fieldTypes map[string]string) string {
	switch d.Kind {
	case "class":
		return classSignature(d)
	case "mixin":
		return join("mixin", d.Name+typeParameters(d.TypeParameters), onClause(d), implementsClause(d))
	case "enum":
		return join("enum", d.Name+typeParameters(d.TypeParameters), withClause(d), implementsClause(d))
	case "extension":
		return join("extension", d.Name+typeParameters(d.TypeParameters), onClause(d))
	case "extension-type":
		return "extension type " + d.Name + typeParameters(d.TypeParameters) + representation(d)
	case "typedef":
		return "typedef " + d.Name + typeParameters(d.TypeParameters)
	case "function":
		return join(d.Returns, d.Name+typeParameters(d.TypeParameters)+parameterListText(d.Parameters, fieldTypes))
	case "variable", "field":
		return join(modifiers(d), d.Type, d.Name)
	case "constructor":
		prefix := ""
		if d.Factory {
			prefix = "factory "
		}

		return prefix + d.Name + parameterListText(d.Parameters, fieldTypes)
	case "getter":
		return join(staticWord(d), d.Returns, "get", d.Name)
	case "setter":
		return join(staticWord(d), "set", d.Name+parameterListText(d.Parameters, fieldTypes))
	case "method":
		return join(staticWord(d), d.Returns, methodName(d.Name)+typeParameters(d.TypeParameters)+parameterListText(d.Parameters, fieldTypes))
	}

	return join(d.Kind, d.Name)
}

func classSignature(d declaration) string {
	words := []string{}
	for _, flag := range []struct {
		set  bool
		word string
	}{{d.Abstract, "abstract"}, {d.Sealed, "sealed"}, {d.Base, "base"}, {d.Final, "final"}, {d.Interface, "interface"}, {d.MixinClass, "mixin"}} {
		if flag.set {
			words = append(words, flag.word)
		}
	}
	extends := ""
	if d.Extends != "" {
		extends = "extends " + d.Extends
	}

	return join(strings.Join(words, " "), "class", d.Name+typeParameters(d.TypeParameters), extends, withClause(d), implementsClause(d))
}

func withClause(d declaration) string {
	if len(d.With) == 0 {
		return ""
	}

	return "with " + strings.Join(d.With, ", ")
}

func implementsClause(d declaration) string {
	if len(d.Implements) == 0 {
		return ""
	}

	return "implements " + strings.Join(d.Implements, ", ")
}

// onClause reads the "on" of an extension ("on String", already worded) or of a mixin (a list).
func onClause(d declaration) string {
	switch on := d.On.(type) {
	case string:
		if strings.HasPrefix(on, "on ") {
			return on
		}

		return "on " + on
	case []any:
		names := make([]string, 0, len(on))
		for _, name := range on {
			if text, ok := name.(string); ok {
				names = append(names, text)
			}
		}
		if len(names) > 0 {
			return "on " + strings.Join(names, ", ")
		}
	}

	return ""
}

func representation(d declaration) string {
	if d.Representation == nil {
		return ""
	}

	return "(" + join(d.Representation.Type, d.Representation.Name) + ")"
}

func typeParameters(parameters []typeParameter) string {
	if len(parameters) == 0 {
		return ""
	}
	parts := make([]string, len(parameters))
	for i, parameter := range parameters {
		parts[i] = parameter.Name
		if parameter.Extends != "" {
			parts[i] += " extends " + parameter.Extends
		}
	}

	return "<" + strings.Join(parts, ", ") + ">"
}

// modifiers are the words before a variable's type: static, late, final, const.
func modifiers(d declaration) string {
	words := []string{}
	if d.Static {
		words = append(words, "static")
	}
	if d.Late {
		words = append(words, "late")
	}
	if d.Final {
		words = append(words, "final")
	}
	if d.Const {
		words = append(words, "const")
	}

	return strings.Join(words, " ")
}

func staticWord(d declaration) string {
	if d.Static {
		return "static"
	}

	return ""
}

// methodName keeps an operator readable: dartdoc_json writes "operator+".
func methodName(name string) string {
	if rest, found := strings.CutPrefix(name, "operator"); found && rest != "" {
		return "operator " + rest
	}

	return name
}

// parameterListText writes "(a, [b], {required c})": the leading required parameters, then the
// optional positional ones in brackets, then the named ones in braces.
func parameterListText(list *parameterList, fieldTypes map[string]string) string {
	if list == nil {
		return "()"
	}
	all := list.All
	named := min(list.Named, len(all))
	positional := min(list.Positional, len(all)-named)
	required := len(all) - named - positional

	var parts []string
	for _, p := range all[:required] {
		parts = append(parts, parameterText(p, fieldTypes))
	}
	if positional > 0 {
		parts = append(parts, "["+joinParameters(all[required:required+positional], fieldTypes)+"]")
	}
	if named > 0 {
		parts = append(parts, "{"+joinParameters(all[len(all)-named:], fieldTypes)+"}")
	}

	return "(" + strings.Join(parts, ", ") + ")"
}

func joinParameters(parameters []parameter, fieldTypes map[string]string) string {
	parts := make([]string, len(parameters))
	for i, p := range parameters {
		parts[i] = parameterText(p, fieldTypes)
	}

	return strings.Join(parts, ", ")
}

func parameterText(p parameter, fieldTypes map[string]string) string {
	typed := p.Type
	if typed == "" {
		if field, found := strings.CutPrefix(p.Name, "this."); found {
			typed = fieldTypes[field]
		}
	}
	text := join(typed, p.Name)
	if p.Required {
		text = "required " + text
	}
	if p.Default != "" {
		text += " = " + p.Default
	}

	return text
}

// join puts the non-empty words on one line.
func join(words ...string) string {
	kept := make([]string, 0, len(words))
	for _, word := range words {
		if word != "" {
			kept = append(kept, word)
		}
	}

	return strings.Join(kept, " ")
}
