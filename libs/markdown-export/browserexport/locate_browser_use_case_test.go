package browserexport

import (
	"errors"
	"path/filepath"
	"testing"
)

func computer(goos string, files []string, env map[string]string, commands map[string]string) machine {
	return machine{
		goos:   goos,
		getenv: func(name string) string { return env[name] },
		isFile: func(path string) bool {
			for _, file := range files {
				if file == path {
					return true
				}
			}

			return false
		},
		lookPath: func(name string) (string, error) {
			if found, ok := commands[name]; ok {
				return found, nil
			}

			return "", errors.New("not found")
		},
	}
}

func TestChromeIsPreferredToEdgeOnWindows(t *testing.T) {
	programFiles := filepath.Join("C:", "Program Files")
	chrome := filepath.Join(programFiles, "Google", "Chrome", "Application", "chrome.exe")
	edge := filepath.Join(programFiles, "Microsoft", "Edge", "Application", "msedge.exe")
	env := map[string]string{"ProgramFiles": programFiles}

	if got, err := locate(computer("windows", []string{edge, chrome}, env, nil), ""); err != nil || got != chrome {
		t.Fatalf("got %q, %v", got, err)
	}
	if got, err := locate(computer("windows", []string{edge}, env, nil), ""); err != nil || got != edge {
		t.Fatalf("with only Edge: got %q, %v", got, err)
	}
}

func TestMacAndLinuxFindTheirBrowsers(t *testing.T) {
	mac := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	if got, err := locate(computer("darwin", []string{mac}, nil, nil), ""); err != nil || got != mac {
		t.Fatalf("mac: got %q, %v", got, err)
	}
	if got, err := locate(computer("linux", nil, nil, map[string]string{"chromium": "/usr/bin/chromium"}), ""); err != nil || got != "/usr/bin/chromium" {
		t.Fatalf("linux: got %q, %v", got, err)
	}
}

func TestAConfiguredPathWinsAndIsNotSilentlyReplaced(t *testing.T) {
	mine := filepath.Join("/opt", "mybrowser")
	env := map[string]string{BrowserVariable: "/from/env"}

	if got, err := locate(computer("linux", []string{mine, "/from/env"}, env, nil), mine); err != nil || got != mine {
		t.Fatalf("configured: got %q, %v", got, err)
	}
	if got, err := locate(computer("linux", []string{"/from/env"}, env, nil), ""); err != nil || got != "/from/env" {
		t.Fatalf("variable: got %q, %v", got, err)
	}
	if _, err := locate(computer("linux", nil, nil, map[string]string{"chromium": "/usr/bin/chromium"}), "/typo/browser"); err == nil {
		t.Fatal("a configured path that does not exist fell back to another browser")
	}
}

func TestNoBrowserSaysHowToFixIt(t *testing.T) {
	if _, err := locate(computer("linux", nil, nil, nil), ""); !errors.Is(err, ErrNoBrowser) {
		t.Fatalf("got %v", err)
	}
}
