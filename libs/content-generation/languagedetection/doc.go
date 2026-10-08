// Package languagedetection finds the projects in a workspace and the language each is
// written in, from the files that mark a project: tsconfig.json, pyproject.toml, go.mod,
// pubspec.yaml, a .csproj. The code-documentation generators use it to know where to run their
// tool when the configuration does not say.
package languagedetection
