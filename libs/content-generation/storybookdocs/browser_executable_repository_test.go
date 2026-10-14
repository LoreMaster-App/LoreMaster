package storybookdocs

import (
	"errors"
	"strings"
	"testing"
)

func fakeLocator(goos string, env map[string]string, onPath []string, installed []string) locator {
	return locator{
		goos:   goos,
		getenv: func(name string) string { return env[name] },
		lookPath: func(name string) (string, error) {
			for _, found := range onPath {
				if found == name {
					return "/bin/" + name, nil
				}
			}

			return "", errors.New("not found")
		},
		exists: func(path string) bool {
			for _, found := range installed {
				if found == path {
					return true
				}
			}

			return false
		},
	}
}

func TestFindBrowserPrefersTheOneTheEnvironmentNames(t *testing.T) {
	l := fakeLocator("linux", map[string]string{BrowserEnvironmentVariable: "/opt/brave"}, []string{"chromium"}, []string{"/opt/brave"})
	got, err := findBrowser(l)
	if err != nil || got != "/opt/brave" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestFindBrowserRefusesANamedBrowserThatIsNotThere(t *testing.T) {
	l := fakeLocator("linux", map[string]string{BrowserEnvironmentVariable: "/opt/none"}, []string{"chromium"}, nil)
	if _, err := findBrowser(l); err == nil || !strings.Contains(err.Error(), BrowserEnvironmentVariable) {
		t.Fatalf("err = %v", err)
	}
}

func TestFindBrowserFallsBackToThePathAndThenTheInstallFolders(t *testing.T) {
	if got, err := findBrowser(fakeLocator("linux", nil, []string{"chromium"}, nil)); err != nil || got != "/bin/chromium" {
		t.Fatalf("got %q, %v", got, err)
	}
	env := map[string]string{"ProgramFiles(x86)": `C:\Program Files (x86)`}
	edge := `C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`
	l := fakeLocator("windows", env, nil, nil)
	l.exists = func(path string) bool { return strings.EqualFold(strings.ReplaceAll(path, "/", `\`), edge) }
	got, err := findBrowser(l)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, "msedge.exe") {
		t.Fatalf("got %q", got)
	}
}

func TestFindBrowserSaysHowToFixItWhenThereIsNone(t *testing.T) {
	_, err := findBrowser(fakeLocator("linux", nil, nil, nil))
	if err == nil || !strings.Contains(err.Error(), BrowserEnvironmentVariable) {
		t.Fatalf("err = %v", err)
	}
}
