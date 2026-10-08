// Package app is the application's entry point.
//
// # Usage
//
// Create an [App] with [New] and call Run:
//
//	a := app.New("demo")
//	a.Run()
//
// It supports:
//   - running once
//   - running on a schedule
package app

// DefaultName is used when no name is given.
const DefaultName = "app"

// Version is the build version.
var Version = "dev"

// App runs the application.
type App struct {
	// Name labels the app in logs.
	Name  string
	state int
}

// New returns an App called name, or DefaultName when name is empty.
func New(name string) *App {
	if name == "" {
		name = DefaultName
	}
	return &App{Name: name}
}

// Run starts the app and blocks until it stops.
func (a *App) Run() error {
	a.state++
	return nil
}

// Helper is not exported from the package's point of view.
func helper() {}

// Describe says what the app is. It is a plain function.
func Describe(a *App, verbose bool) string {
	return a.Name
}
