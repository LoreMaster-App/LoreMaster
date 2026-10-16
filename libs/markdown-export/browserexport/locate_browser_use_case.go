// Package browserexport renders an HTML page to PDF or PNG with the browser already on the machine
// (Chrome, Edge or Chromium, run headless), so no browser is bundled with LoreMaster.
package browserexport

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ErrNoBrowser says no usable browser was found. Its text is what the person sees.
var ErrNoBrowser = errors.New("PDF and PNG need Chrome, Edge or Chromium, and none was found: install one, or set the browser's path (the loreMaster.export.browserPath setting, --browser, or the LORE_MASTER_BROWSER variable)")

// BrowserVariable names the environment variable that points at a browser.
const BrowserVariable = "LORE_MASTER_BROWSER"

// machine is what the search asks of the computer, so a test can describe a computer without having one.
type machine struct {
	goos     string
	getenv   func(string) string
	isFile   func(string) bool
	lookPath func(string) (string, error)
}

func realMachine() machine {
	return machine{
		goos:   runtime.GOOS,
		getenv: os.Getenv,
		isFile: func(path string) bool {
			info, err := os.Stat(path)

			return err == nil && !info.IsDir()
		},
		lookPath: exec.LookPath,
	}
}

// LocateBrowser finds a browser to print with: the configured path when there is one, else the one the
// LORE_MASTER_BROWSER variable names, else the first Chrome, Edge or Chromium installed in the usual place.
// A configured path that does not exist is an error, not a reason to look elsewhere, so a typo is not hidden.
func LocateBrowser(configured string) (string, error) {
	return locate(realMachine(), configured)
}

func locate(computer machine, configured string) (string, error) {
	if path := strings.TrimSpace(configured); path != "" {
		return explicit(computer, path, "the configured browser path")
	}
	if path := strings.TrimSpace(computer.getenv(BrowserVariable)); path != "" {
		return explicit(computer, path, BrowserVariable)
	}
	for _, path := range installedPaths(computer) {
		if computer.isFile(path) {
			return path, nil
		}
	}
	for _, name := range commandNames(computer.goos) {
		if path, err := computer.lookPath(name); err == nil {
			return path, nil
		}
	}

	return "", ErrNoBrowser
}

func explicit(computer machine, path, source string) (string, error) {
	if computer.isFile(path) {
		return path, nil
	}
	if found, err := computer.lookPath(path); err == nil {
		return found, nil
	}

	return "", errors.New(source + " (" + path + ") is not a browser that can be run")
}
