package openapidocs

import (
	"fmt"
	"regexp"
	"strings"

	"lore-master/libs/content-generation/generatedfile"
)

const (
	schemasPage = "schemas.md"
	defaultTag  = "default"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// api is one description with the names its pages go by.
type api struct {
	desc   description
	source string
	// title is the unique page title of the API's overview; dir is its folder in the output.
	title string
	dir   string
}

// group is the operations that share a tag, which becomes a page.
type group struct {
	tag        string
	title      string
	file       string
	operations []groupedOperation
}

type groupedOperation struct {
	method string
	path   string
	item   pathItem
	op     *operation
}

// renderAPI writes an API's pages, all inside its own folder: the overview (README.md, so the
// other pages nest under it), a page per tag, and the schemas page when there are schemas.
func renderAPI(a api) []generatedfile.File {
	groups := groupOperations(a)

	files := []generatedfile.File{{Path: a.dir + "/README.md", Body: []byte(renderOverview(a, groups))}}
	for _, g := range groups {
		files = append(files, generatedfile.File{Path: a.dir + "/" + g.file, Body: []byte(renderGroup(a, g))})
	}
	if len(a.desc.Components.Schemas.Keys) > 0 {
		files = append(files, generatedfile.File{Path: a.dir + "/" + schemasPage, Body: []byte(renderSchemas(a))})
	}

	return files
}

// groupOperations sorts the operations into tag pages: tags in the order the description
// declares them, then tags only operations mention, then "default" for the untagged. An
// operation lives on the page of its first tag.
func groupOperations(a api) []group {
	byTag := map[string]*group{}
	var order []string
	add := func(tag string) *group {
		if found, ok := byTag[tag]; ok {
			return found
		}
		byTag[tag] = &group{tag: tag}
		order = append(order, tag)

		return byTag[tag]
	}
	declared := map[string]bool{}
	for _, tag := range a.desc.Tags {
		declared[tag.Name] = true
	}

	var collected []groupedOperation
	a.desc.Paths.each(func(path string, item pathItem) {
		for _, candidate := range item.operations() {
			collected = append(collected, groupedOperation{method: candidate.method, path: path, item: item, op: candidate.operation})
		}
	})
	used := map[string]bool{}
	for _, op := range collected {
		tag := defaultTag
		if len(op.op.Tags) > 0 {
			tag = op.op.Tags[0]
		}
		used[tag] = true
	}
	for _, tag := range a.desc.Tags {
		if used[tag.Name] {
			add(tag.Name)
		}
	}
	for _, op := range collected {
		tag := defaultTag
		if len(op.op.Tags) > 0 {
			tag = op.op.Tags[0]
		}
		add(tag).operations = append(add(tag).operations, op)
	}

	files := map[string]bool{"readme": true, "schemas": true}
	groups := make([]group, 0, len(order))
	for _, tag := range order {
		g := *byTag[tag]
		slug := slugOf(tag)
		unique := slug
		for n := 2; files[unique]; n++ {
			unique = fmt.Sprintf("%s-%d", slug, n)
		}
		files[unique] = true
		g.file, g.title = unique+".md", a.title+": "+tag
		groups = append(groups, g)
	}

	return groups
}

func renderOverview(a api, groups []group) string {
	desc := a.desc
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", a.title)
	if desc.Info.Version != "" {
		fmt.Fprintf(&out, "Version %s, OpenAPI %s. Source: %s.\n\n", cell(desc.Info.Version), cell(desc.OpenAPI), code(a.source))
	} else {
		fmt.Fprintf(&out, "OpenAPI %s. Source: %s.\n\n", cell(desc.OpenAPI), code(a.source))
	}
	if text := strings.TrimSpace(desc.Info.Description); text != "" {
		out.WriteString(text + "\n\n")
	}
	if len(desc.Servers) > 0 {
		out.WriteString("## Servers\n\n| URL | Description |\n|---|---|\n")
		for _, server := range desc.Servers {
			fmt.Fprintf(&out, "| %s | %s |\n", code(server.URL), cell(server.Description))
		}
		out.WriteString("\n")
	}
	if len(groups) == 0 {
		out.WriteString("This description defines no operations.\n")
	} else {
		out.WriteString("## Operations\n\n| Method | Path | Summary | Group |\n|---|---|---|---|\n")
		for _, g := range groups {
			for _, op := range g.operations {
				fmt.Fprintf(&out, "| %s | %s | %s | [%s](%s) |\n", op.method, code(op.path), cell(op.op.Summary), cell(g.tag), g.file)
			}
		}
		out.WriteString("\n")
	}
	if len(desc.Components.Schemas.Keys) > 0 {
		fmt.Fprintf(&out, "The data types are described on the [schemas](%s) page.\n", schemasPage)
	}

	return strings.TrimRight(out.String(), "\n") + "\n"
}

func renderGroup(a api, g group) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", g.title)
	for _, tag := range a.desc.Tags {
		if tag.Name == g.tag && strings.TrimSpace(tag.Description) != "" {
			out.WriteString(strings.TrimSpace(tag.Description) + "\n\n")
		}
	}
	for _, op := range g.operations {
		writeOperation(&out, a, op)
	}

	return strings.TrimRight(out.String(), "\n") + "\n"
}

func writeOperation(out *strings.Builder, a api, g groupedOperation) {
	op := g.op
	fmt.Fprintf(out, "## %s %s\n\n", g.method, g.path)
	var facts []string
	if op.Summary != "" {
		facts = append(facts, "**"+cell(op.Summary)+"**")
	}
	if op.OperationID != "" {
		facts = append(facts, "operation "+code(op.OperationID))
	}
	if op.Deprecated {
		facts = append(facts, "**deprecated**")
	}
	if len(facts) > 0 {
		out.WriteString(strings.Join(facts, " · ") + "\n\n")
	}
	if text := strings.TrimSpace(op.Description); text != "" {
		out.WriteString(text + "\n\n")
	}

	if parameters := resolvedParameters(a, g); len(parameters) > 0 {
		out.WriteString("### Parameters\n\n| Name | In | Type | Required | Description |\n|---|---|---|---|---|\n")
		for _, p := range parameters {
			fmt.Fprintf(out, "| %s | %s | %s | %s | %s |\n", code(p.Name), p.In, schemaType(p.Schema), yesNo(p.Required), cell(describe(p.Description, p.Schema)))
		}
		out.WriteString("\n")
	}
	if body := resolveBody(a, op.RequestBody); body != nil {
		out.WriteString("### Request body\n\n")
		if body.Required {
			out.WriteString("Required.")
		}
		if text := strings.TrimSpace(body.Description); text != "" {
			if body.Required {
				out.WriteString(" ")
			}
			out.WriteString(cell(text))
		}
		out.WriteString("\n\n")
		writeContent(out, body.Content)
	}
	if len(op.Responses.Keys) > 0 {
		out.WriteString("### Responses\n\n| Status | Description | Content |\n|---|---|---|\n")
		op.Responses.each(func(status string, r response) {
			r = resolveResponse(a, r)
			fmt.Fprintf(out, "| %s | %s | %s |\n", code(status), cell(r.Description), contentSummary(r.Content))
		})
		out.WriteString("\n")
	}
}

// resolvedParameters are the parameters of an operation: those of its path, then its own,
// with $refs into components resolved and an operation's parameter replacing a path-level one
// of the same name and location.
func resolvedParameters(a api, g groupedOperation) []parameter {
	var parameters []parameter
	indexOf := map[string]int{}
	for _, candidate := range append(append([]parameter{}, g.item.Parameters...), g.op.Parameters...) {
		resolved := candidate
		if candidate.Ref != "" {
			if found, ok := a.desc.Components.Parameters.Values[refName(candidate.Ref)]; ok {
				resolved = found
			} else {
				resolved = parameter{Name: refName(candidate.Ref), Description: "unresolved reference " + candidate.Ref}
			}
		}
		key := resolved.In + " " + resolved.Name
		if at, seen := indexOf[key]; seen {
			parameters[at] = resolved

			continue
		}
		indexOf[key] = len(parameters)
		parameters = append(parameters, resolved)
	}

	return parameters
}

func resolveBody(a api, body *requestBody) *requestBody {
	if body == nil {
		return nil
	}
	if body.Ref != "" {
		if found, ok := a.desc.Components.RequestBodies.Values[refName(body.Ref)]; ok {
			return &found
		}

		return &requestBody{Description: "unresolved reference " + body.Ref}
	}

	return body
}

func resolveResponse(a api, r response) response {
	if r.Ref == "" {
		return r
	}
	if found, ok := a.desc.Components.Responses.Values[refName(r.Ref)]; ok {
		return found
	}

	return response{Description: "unresolved reference " + r.Ref}
}

// writeContent lists a body's media types with the type of each one's schema.
func writeContent(out *strings.Builder, content ordered[mediaType]) {
	if len(content.Keys) == 0 {
		return
	}
	out.WriteString("| Content type | Schema |\n|---|---|\n")
	content.each(func(kind string, media mediaType) {
		fmt.Fprintf(out, "| %s | %s |\n", code(kind), schemaType(media.Schema))
	})
	out.WriteString("\n")
}

// contentSummary is a response's media types and schemas on one line, for a table cell.
func contentSummary(content ordered[mediaType]) string {
	var parts []string
	content.each(func(kind string, media mediaType) {
		parts = append(parts, code(kind)+": "+schemaType(media.Schema))
	})
	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "<br>")
}

func renderSchemas(a api) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# %s: Schemas\n\n", a.title)
	a.desc.Components.Schemas.each(func(name string, s *schema) {
		fmt.Fprintf(&out, "## %s\n\n", name)
		if s == nil {
			return
		}
		if text := strings.TrimSpace(s.Description); text != "" {
			out.WriteString(text + "\n\n")
		}
		fmt.Fprintf(&out, "Type: %s\n\n", schemaType(s))
		if len(s.Enum) > 0 {
			fmt.Fprintf(&out, "One of: %s\n\n", enumValues(s.Enum))
		}
		writeProperties(&out, s)
	})

	return strings.TrimRight(out.String(), "\n") + "\n"
}

// writeProperties lists a schema's properties, including those of the inline parts of an
// allOf, which together make up the object; a part that is a reference stays a link in the
// type line.
func writeProperties(out *strings.Builder, s *schema) {
	properties, required := collectProperties(s)
	if len(properties.Keys) == 0 {
		return
	}
	out.WriteString("| Property | Type | Required | Description |\n|---|---|---|---|\n")
	properties.each(func(name string, property *schema) {
		description := ""
		if property != nil {
			description = describe(property.Description, property)
		}
		fmt.Fprintf(out, "| %s | %s | %s | %s |\n", code(name), schemaType(property), yesNo(required[name]), cell(description))
	})
	out.WriteString("\n")
}

// collectProperties gathers a schema's own properties and those of its inline allOf parts, in
// the order written, with the names any of them requires.
func collectProperties(s *schema) (ordered[*schema], map[string]bool) {
	merged := ordered[*schema]{Values: map[string]*schema{}}
	required := map[string]bool{}
	var visit func(part *schema)
	visit = func(part *schema) {
		if part == nil || part.Ref != "" {
			return
		}
		part.Properties.each(func(name string, property *schema) {
			if _, seen := merged.Values[name]; !seen {
				merged.Keys = append(merged.Keys, name)
			}
			merged.Values[name] = property
		})
		for _, name := range part.Required {
			required[name] = true
		}
		for _, inner := range part.AllOf {
			visit(inner)
		}
	}
	visit(s)

	return merged, required
}

// schemaType is a schema's type as Markdown safe inside a table cell: a reference to a
// component schema links to the schemas page, an array reads "array of X", and a 3.1 type
// list reads "string or null".
func schemaType(s *schema) string {
	switch {
	case s == nil:
		return "any"
	case s.Ref != "":
		if name, local := strings.CutPrefix(s.Ref, "#/components/schemas/"); local {
			return fmt.Sprintf("[%s](%s)", cell(name), schemasPage)
		}

		return code(s.Ref)
	case len(s.AllOf) > 0:
		return joinTypes("all of", s.AllOf)
	case len(s.OneOf) > 0:
		return joinTypes("one of", s.OneOf)
	case len(s.AnyOf) > 0:
		return joinTypes("any of", s.AnyOf)
	}

	kind := strings.Join(s.Type, " or ")
	switch {
	case kind == "" && len(s.Properties.Keys) > 0:
		kind = "object"
	case kind == "":
		kind = "any"
	}
	if len(s.Type) == 1 && s.Type[0] == "array" || kind == "array" {
		kind = "array of " + schemaType(s.Items)
	}
	if s.Format != "" {
		kind += " (" + cell(s.Format) + ")"
	}
	if s.Nullable {
		kind += " or null"
	}

	return kind
}

func joinTypes(label string, parts []*schema) string {
	types := make([]string, len(parts))
	for i, part := range parts {
		types[i] = schemaType(part)
	}

	return label + " " + strings.Join(types, ", ")
}

// describe is a description followed by what the schema restricts its value to.
func describe(text string, s *schema) string {
	parts := []string{strings.TrimSpace(text)}
	if s != nil && len(s.Enum) > 0 {
		parts = append(parts, "One of: "+enumValues(s.Enum)+".")
	}
	if s != nil && s.Default != nil {
		parts = append(parts, fmt.Sprintf("Default: %v.", s.Default))
	}
	if s != nil && s.Deprecated {
		parts = append(parts, "Deprecated.")
	}

	return strings.TrimSpace(strings.Join(nonEmpty(parts), " "))
}

func nonEmpty(parts []string) []string {
	var kept []string
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}

	return kept
}

func enumValues(values []any) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = code(fmt.Sprint(value))
	}

	return strings.Join(parts, ", ")
}

// refName is the last segment of a "#/components/..." reference.
func refName(ref string) string {
	return ref[strings.LastIndex(ref, "/")+1:]
}

// slugOf is a file or folder name: lower-case letters and digits, with no dots, because a
// dotted file name nests the page under another one.
func slugOf(name string) string {
	slug := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if slug == "" {
		return "api"
	}

	return slug
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}

// code is text as inline code safe inside a table cell.
func code(text string) string {
	return "`" + strings.NewReplacer("`", "'", "|", `\|`, "\n", " ", "\r", "").Replace(strings.TrimSpace(text)) + "`"
}

// cell is text safe inside a table cell.
func cell(text string) string {
	return strings.NewReplacer("|", `\|`, "\n", " ", "\r", "", "<", "&lt;").Replace(strings.TrimSpace(text))
}
