package synccommands

import (
	"fmt"
	"strings"

	"lore-master/libs/documentation-sync/platformport"
)

// matchSpace finds the space a setting names. The key wins; then the alias the space's address
// uses, because a Cloud space keeps its key for life while its URL alias can change, and the
// alias is what people copy from the browser's address bar. Comparison ignores case.
func matchSpace(spaces []platformport.Space, wanted string) (platformport.Space, bool) {
	wanted = strings.TrimSpace(wanted)
	if wanted == "" {
		return platformport.Space{}, false
	}
	for _, space := range spaces {
		if space.Key == wanted {
			return space, true
		}
	}
	for _, space := range spaces {
		if strings.EqualFold(space.Key, wanted) {
			return space, true
		}
	}
	for _, space := range spaces {
		if space.Alias != "" && strings.EqualFold(space.Alias, wanted) {
			return space, true
		}
	}

	return platformport.Space{}, false
}

// describeSpaces lists spaces for an error message, each with its key, alias and name, and says
// how many more there are past the limit.
func describeSpaces(spaces []platformport.Space, limit int) string {
	shown := spaces
	if len(shown) > limit {
		shown = shown[:limit]
	}
	parts := make([]string, len(shown))
	for i, space := range shown {
		switch {
		case space.Alias != "" && space.Name != "":
			parts[i] = fmt.Sprintf("%s (alias %s, %q)", space.Key, space.Alias, space.Name)
		case space.Name != "":
			parts[i] = fmt.Sprintf("%s (%q)", space.Key, space.Name)
		default:
			parts[i] = space.Key
		}
	}
	text := strings.Join(parts, ", ")
	if len(spaces) > limit {
		text += fmt.Sprintf(", and %d more", len(spaces)-limit)
	}

	return text
}
