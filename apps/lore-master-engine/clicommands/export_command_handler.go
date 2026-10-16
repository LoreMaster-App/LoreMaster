package clicommands

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"lore-master/apps/lore-master-engine/rpcserver"
	"lore-master/libs/markdown-export/documentexport"
)

// exportCommand converts Markdown files to another format. It needs no settings, no engine and no
// credentials: it reads each file and writes the result, so it can run anywhere a file can be read.
func exportCommand(_ context.Context, env Environment, _ rpcserver.Methods, args []string) int {
	flags := flag.NewFlagSet("lore-master-engine export", flag.ContinueOnError)
	flags.SetOutput(env.Stderr)
	var format, out string
	var toStdout, list bool
	var options documentexport.Options
	flags.StringVar(&format, "format", "", "what to export to (see --list)")
	flags.StringVar(&out, "out", "", "the folder to write into (default: next to each file)")
	flags.BoolVar(&toStdout, "stdout", false, "print the result instead of writing a file (one file only)")
	flags.BoolVar(&list, "list", false, "list the formats and stop")
	flags.StringVar(&options.BrowserPath, "browser", "", "the Chrome, Edge or Chromium that prints pdf and png (default: look for one; or set LORE_MASTER_BROWSER)")
	flags.StringVar(&options.PageSize, "page-size", "", "the paper of a pdf: A4, A3, A5, Letter or Legal (default A4)")
	flags.IntVar(&options.Width, "width", 0, "the width of a png in pixels (default 1000)")
	if err := flags.Parse(args); err != nil {
		return ExitUsage
	}

	if list {
		for _, each := range documentexport.Formats() {
			_, _ = fmt.Fprintf(env.Stdout, "%-10s %s\n", each.ID, each.Label)
		}

		return ExitOK
	}
	if format == "" || flags.NArg() == 0 {
		_, _ = fmt.Fprintln(env.Stderr, "usage: lore-master-engine export --format ID [--out DIR | --stdout] FILE...")

		return ExitUsage
	}
	if toStdout && flags.NArg() > 1 {
		_, _ = fmt.Fprintln(env.Stderr, "--stdout takes one file")

		return ExitUsage
	}

	code := ExitOK
	for _, name := range flags.Args() {
		if err := exportFile(env, name, format, out, toStdout, options); err != nil {
			_, _ = fmt.Fprintf(env.Stderr, "%s: %v\n", name, err)
			code = ExitFailed
		}
	}

	return code
}

func exportFile(env Environment, name, format, out string, toStdout bool, options documentexport.Options) error {
	path := name
	if !filepath.IsAbs(path) {
		path = filepath.Join(env.WorkingDir, path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	options.BaseDir = filepath.Dir(path)
	file, err := documentexport.Export(path, content, format, options)
	if err != nil {
		return err
	}
	if toStdout {
		_, err = env.Stdout.Write(file.Content)

		return err
	}

	folder := out
	if folder == "" {
		folder = filepath.Dir(path)
	} else if !filepath.IsAbs(folder) {
		folder = filepath.Join(env.WorkingDir, folder)
	}
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return err
	}
	target := filepath.Join(folder, file.Name)
	if err := os.WriteFile(target, file.Content, 0o644); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(env.Stdout, target)

	return nil
}
