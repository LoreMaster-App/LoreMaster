package editorsettings

import "strings"

// keyPrefix is what every LoreMaster setting starts with in settings.json.
const keyPrefix = "loreMaster."

// The settings that carry configuration. Everything else under keyPrefix is either an
// editor-only preference (extensionOnlyKeys) or a mistake.
const (
	keyOutputs         = "loreMaster.outputs"
	keyIgnore          = "loreMaster.ignore"
	keySkipGitignored  = "loreMaster.skipGitignored"
	keyGenerators      = "loreMaster.generators"
	keyConfluenceURL   = "loreMaster.confluence.baseUrl"
	keyDefaultsPrefix  = "loreMaster.defaults."
	keyDefaultsDirect  = keyDefaultsPrefix + "direction"
	keyDefaultsMermaid = keyDefaultsPrefix + "mermaidMode"
	keyDefaultsLink    = keyDefaultsPrefix + "linkMode"
	keyDefaultsCollide = keyDefaultsPrefix + "titleCollision"
)

var configKeys = map[string]bool{
	keyOutputs: true, keyIgnore: true, keySkipGitignored: true, keyGenerators: true,
	keyConfluenceURL: true, keyDefaultsDirect: true, keyDefaultsMermaid: true,
	keyDefaultsLink: true, keyDefaultsCollide: true,
}

// extensionOnlyKeys are the editor extension's own preferences; they are not part of the
// configuration and are ignored here.
var extensionOnlyKeys = []string{
	"loreMaster.engine.", "loreMaster.pages.", "loreMaster.generateBeforeSync", "loreMaster.watchDebounceSeconds",
}

func isExtensionOnly(key string) bool {
	for _, prefix := range extensionOnlyKeys {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}
