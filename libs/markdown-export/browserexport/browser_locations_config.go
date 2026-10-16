package browserexport

import "path/filepath"

// installedPaths are where each system puts the browsers, best first: Chrome, then Edge (which every
// Windows 10 and 11 has), then Chromium.
func installedPaths(computer machine) []string {
	switch computer.goos {
	case "windows":
		var paths []string
		for _, variable := range []string{"ProgramFiles", "ProgramFiles(x86)", "LocalAppData"} {
			base := computer.getenv(variable)
			if base == "" {
				continue
			}
			paths = append(paths,
				filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"),
				filepath.Join(base, "Chromium", "Application", "chrome.exe"),
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
		return nil
	}
}

// commandNames are the commands a Linux system, and any other that is not covered above, runs a browser by.
func commandNames(goos string) []string {
	if goos == "windows" || goos == "darwin" {
		return nil
	}

	return []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge", "microsoft-edge-stable", "chrome"}
}
