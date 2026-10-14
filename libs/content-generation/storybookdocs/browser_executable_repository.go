package storybookdocs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// BrowserEnvironmentVariable names the browser to use when the usual places hold none.
const BrowserEnvironmentVariable = "LORE_MASTER_BROWSER"

// browserNames are looked up on PATH, in order.
var browserNames = []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome", "msedge", "microsoft-edge"}

// locator is what finding a browser asks of the machine, so tests can supply a fake one.
type locator struct {
	goos     string
	getenv   func(string) string
	lookPath func(string) (string, error)
	exists   func(string) bool
}

func systemLocator() locator {
	return locator{
		goos:     runtime.GOOS,
		getenv:   os.Getenv,
		lookPath: exec.LookPath,
		exists: func(path string) bool {
			info, err := os.Stat(path)

			return err == nil && !info.IsDir()
		},
	}
}

// findBrowser answers the path of a Chromium-based browser: the one named by
// LORE_MASTER_BROWSER, then one on PATH, then the install folders of Chrome, Chromium and Edge.
func findBrowser(l locator) (string, error) {
	if named := l.getenv(BrowserEnvironmentVariable); named != "" {
		if l.exists(named) {
			return named, nil
		}
		if found, err := l.lookPath(named); err == nil {
			return found, nil
		}

		return "", fmt.Errorf("%s names %q, which is not a browser that can be started", BrowserEnvironmentVariable, named)
	}
	for _, name := range browserNames {
		if found, err := l.lookPath(name); err == nil {
			return found, nil
		}
	}
	for _, candidate := range installedBrowsers(l) {
		if l.exists(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no Chrome, Chromium or Edge was found to render the stories; install one, or set %s to its path", BrowserEnvironmentVariable)
}

// installedBrowsers are the places the browsers install themselves on each system.
func installedBrowsers(l locator) []string {
	switch l.goos {
	case "windows":
		var paths []string
		for _, variable := range []string{"ProgramFiles", "ProgramFiles(x86)", "LocalAppData"} {
			root := l.getenv(variable)
			if root == "" {
				continue
			}
			paths = append(paths,
				filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe"),
				filepath.Join(root, "Chromium", "Application", "chrome.exe"),
			)
		}

		return paths
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	default:
		return []string{"/usr/bin/google-chrome", "/usr/bin/chromium", "/usr/bin/chromium-browser", "/snap/bin/chromium", "/usr/bin/microsoft-edge"}
	}
}
