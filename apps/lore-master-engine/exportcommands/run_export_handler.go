package exportcommands

import (
	"context"
	"os"
	"path/filepath"

	"lore-master/apps/lore-master-engine/rpcprotocol"
	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/markdown-export/documentexport"
)

// RunExport handles export/run: one document converted and written next to it or into a folder, or
// returned as text.
func RunExport() rpcserver.Method {
	return func(_ context.Context, call rpcserver.Call) (any, error) {
		var params rpcprotocol.ExportRunParams
		if err := call.Decode(&params); err != nil {
			return nil, err
		}
		if !filepath.IsAbs(params.Path) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "path must be absolute, got %q", params.Path)
		}
		format, found := documentexport.FormatByID(params.Format)
		if !found {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "unknown export format %q", params.Format)
		}
		if params.Inline && format.Binary {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "%s is not text, so it can only be written to a file", format.Label)
		}
		if params.OutDir != "" && !filepath.IsAbs(params.OutDir) {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "outDir must be absolute, got %q", params.OutDir)
		}

		source, err := sourceOf(params)
		if err != nil {
			return nil, err
		}
		file, err := documentexport.Export(params.Path, source, params.Format, documentexport.Options{BaseDir: filepath.Dir(params.Path), BrowserPath: params.BrowserPath, PageSize: params.PageSize, Width: params.ImageWidth, Preview: params.Preview})
		if err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInternalError, "export %s: %s", filepath.Base(params.Path), err.Error())
		}
		if params.Inline {
			return rpcprotocol.ExportRunResult{Name: file.Name, Text: string(file.Content)}, nil
		}

		folder := params.OutDir
		if folder == "" {
			folder = filepath.Dir(params.Path)
		}
		if err := os.MkdirAll(folder, 0o755); err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInternalError, "create %s: %s", folder, err.Error())
		}
		target := filepath.Join(folder, file.Name)
		if err := os.WriteFile(target, file.Content, 0o644); err != nil {
			return nil, rpcprotocol.Errorf(rpcprotocol.CodeInternalError, "write %s: %s", target, err.Error())
		}

		return rpcprotocol.ExportRunResult{Name: file.Name, Path: target}, nil
	}
}

// sourceOf is the text to convert: what the editor holds when it says so, else the file.
func sourceOf(params rpcprotocol.ExportRunParams) ([]byte, error) {
	if params.Content != nil {
		return []byte(*params.Content), nil
	}
	content, err := os.ReadFile(params.Path)
	if err != nil {
		return nil, rpcprotocol.Errorf(rpcprotocol.CodeInvalidParams, "read %s: %s", params.Path, err.Error())
	}

	return content, nil
}
