package csharpdocs

import (
	"regexp"
	"strings"
)

var testProjectName = regexp.MustCompile(`(?i)(^|[._-])(unit|integration|e2e)?tests?$|\.tests?\.`)

// isTestProject reports whether a project file is a test project by its name (Shop.Tests.csproj,
// Shop.UnitTests.csproj): tests are not API.
func isTestProject(markerFile string) bool {
	name := strings.TrimSuffix(markerFile, ".csproj")

	return testProjectName.MatchString(name)
}

// highestFramework picks the last target framework of a multi-targeting project, which is the
// newest by convention.
func highestFramework(frameworks string) string {
	parts := strings.Split(frameworks, ";")
	for i := len(parts) - 1; i >= 0; i-- {
		if framework := strings.TrimSpace(parts[i]); framework != "" {
			return framework
		}
	}

	return ""
}
