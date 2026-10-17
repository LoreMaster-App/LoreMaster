package configcatalog

import "strings"

// Describe is everything that can be configured. Each call returns fresh values a caller may change.
func Describe() Description {
	types := generatorTypes()
	fields := generatorFields()
	for i := range fields {
		if fields[i].Key == "type" {
			for _, generator := range types {
				fields[i].Choices = append(fields[i].Choices, Choice{Value: generator.ID, Label: generator.Label, Help: generator.Description})
			}
		}
	}

	return Description{
		Platforms:       Platforms(),
		OutputFields:    outputFields(),
		Generators:      types,
		GeneratorFields: fields,
		Sections:        sections(),
	}
}

// Platforms are the storages that can be synced to today, in the order to offer them.
func Platforms() []Platform {
	return []Platform{confluencePlatform(), githubPagesPlatform(), githubWikiPlatform()}
}

// PlatformIDs are the platform ids an output may use.
func PlatformIDs() []string {
	platforms := Platforms()
	ids := make([]string, len(platforms))
	for i, platform := range platforms {
		ids[i] = platform.ID
	}

	return ids
}

// GeneratorIDs are the generator types that exist.
func GeneratorIDs() []string {
	types := generatorTypes()
	ids := make([]string, len(types))
	for i, generator := range types {
		ids[i] = generator.ID
	}

	return ids
}

// FindPlatform returns the platform with this id.
func FindPlatform(id string) (Platform, bool) {
	for _, platform := range Platforms() {
		if platform.ID == id {
			return platform, true
		}
	}

	return Platform{}, false
}

// MakeUpName fills the platform's NameTemplate with the output's values, read through value
// (the output's setting by key). "{space|no space yet}" becomes the space, or the fallback.
func (platform Platform) MakeUpName(value func(key string) string) string {
	var name strings.Builder
	rest := platform.NameTemplate
	for {
		open := strings.Index(rest, "{")
		closing := strings.Index(rest, "}")
		if open < 0 || closing < open {
			name.WriteString(rest)

			break
		}
		name.WriteString(rest[:open])
		key, fallback, _ := strings.Cut(rest[open+1:closing], "|")
		if found := value(key); found != "" {
			name.WriteString(found)
		} else {
			name.WriteString(fallback)
		}
		rest = rest[closing+1:]
	}

	return name.String()
}
