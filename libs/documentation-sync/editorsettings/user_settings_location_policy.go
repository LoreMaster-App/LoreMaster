package editorsettings

import "path/filepath"

// UserSettingsOverrideVariable names the environment variable that points at the user
// settings file directly: for CI, for another editor (VSCodium, Insiders), or a test.
const UserSettingsOverrideVariable = "LORE_MASTER_USER_SETTINGS"

// UserSettingsPath is where VS Code keeps the user's settings.json on this system. env
// reads an environment variable (os.Getenv in production); goos is runtime.GOOS. It
// returns "" when the location cannot be known, which a caller treats as an empty layer.
func UserSettingsPath(goos string, env func(string) string, home string) string {
	if override := env(UserSettingsOverrideVariable); override != "" {
		return override
	}
	switch goos {
	case "windows":
		if appData := env("APPDATA"); appData != "" {
			return filepath.Join(appData, "Code", "User", "settings.json")
		}
	case "darwin":
		if home != "" {
			return filepath.Join(home, "Library", "Application Support", "Code", "User", "settings.json")
		}
	default:
		if config := env("XDG_CONFIG_HOME"); config != "" {
			return filepath.Join(config, "Code", "User", "settings.json")
		}
		if home != "" {
			return filepath.Join(home, ".config", "Code", "User", "settings.json")
		}
	}

	return ""
}
