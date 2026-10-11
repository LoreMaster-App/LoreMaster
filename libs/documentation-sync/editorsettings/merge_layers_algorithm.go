package editorsettings

import "encoding/json"

// merged is one key's winning value and the file it came from.
type merged struct {
	value  json.RawMessage
	source string
}

// mergeLayers adds layers up, given least specific first (user, then folder): for each
// key the last layer that sets it wins, whole. That is how the editor treats arrays and
// scalars, so a folder's outputs replace the user's rather than being appended to them.
func mergeLayers(layers []Layer) map[string]merged {
	result := map[string]merged{}
	for _, layer := range layers {
		for key, value := range layer.Values {
			result[key] = merged{value: value, source: layer.Source}
		}
	}

	return result
}
