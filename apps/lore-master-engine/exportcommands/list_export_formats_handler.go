package exportcommands

import (
	"context"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/markdown-export/documentexport"
)

// ListFormats handles export/formats.
func ListFormats() rpcserver.Method {
	return func(context.Context, rpcserver.Call) (any, error) {
		var result rpcprotocol.ExportFormatsResult
		for _, format := range documentexport.Formats() {
			result.Formats = append(result.Formats, rpcprotocol.ExportFormat{ID: format.ID, Label: format.Label, Extension: format.Extension, Binary: format.Binary})
		}

		return result, nil
	}
}
