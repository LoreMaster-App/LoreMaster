<!-- file: README.md -->
# Go API reference

8 packages.

| Package | Synopsis |
|---|---|
| [example.com/app](app.md) | Package app is the application's entry point. |
| [example.com/app/broken](broken/README.md) | Package broken has one file that does not parse. |
| [example.com/app/cmd/tool](cmd/tool/README.md) | Command tool prints the app's name. |
| [example.com/app/internal/secret](internal/secret/README.md) | Package secret holds what only this module may use. |
| [example.com/app/mixed](mixed/README.md) | Package mixed shares its folder with a stray package clause. |
| [example.com/app/platform](platform/README.md) | Package platform has files for several operating systems. |
| [example.com/app/store](store/README.md) | Package store keeps key/value pairs in memory. |
| [example.com/tools/ext](tools/ext/README.md) | Package ext belongs to a nested module. |

<!-- file: app.md -->
# example.com/app

```go
import "example.com/app"
```

Package app is the application's entry point.

### Usage

Create an \[App] with \[New] and call Run:

	a := app.New("demo")
	a.Run()

It supports:

  - running once
  - running on a schedule

## Constants

```go
const DefaultName = "app"
```

DefaultName is used when no name is given.

## Variables

```go
var Version = "dev"
```

Version is the build version.

## Functions

### func Describe

```go
func Describe(a *App, verbose bool) string
```

Describe says what the app is. It is a plain function.

## Types

### type App

```go
type App struct {
	// Name labels the app in logs.
	Name string
	// contains filtered or unexported fields
}
```

App runs the application.

#### func New

```go
func New(name string) *App
```

New returns an App called name, or DefaultName when name is empty.

#### func (*App) Run

```go
func (a *App) Run() error
```

Run starts the app and blocks until it stops.

<!-- file: broken/README.md -->
# example.com/app/broken

```go
import "example.com/app/broken"
```

Package broken has one file that does not parse.

## Functions

### func Fine

```go
func Fine()
```

Fine is documented.

<!-- file: cmd/tool/README.md -->
# example.com/app/cmd/tool

This is a command (package main), not an importable package.

Command tool prints the app's name.

<!-- file: internal/secret/README.md -->
# example.com/app/internal/secret

```go
import "example.com/app/internal/secret"
```

Package secret holds what only this module may use.

## Constants

```go
const Token = "x"
```

Token is a placeholder.

<!-- file: mixed/README.md -->
# example.com/app/mixed

```go
import "example.com/app/mixed"
```

Package mixed shares its folder with a stray package clause.

## Functions

### func A

```go
func A()
```

A is in the documented package.

### func B

```go
func B()
```

B is in the documented package too.

<!-- file: platform/README.md -->
# example.com/app/platform

```go
import "example.com/app/platform"
```

Package platform has files for several operating systems.

## Functions

### func LinuxOnly

```go
func LinuxOnly()
```

LinuxOnly exists on Linux builds.

### func WindowsOnly

```go
func WindowsOnly()
```

WindowsOnly exists on Windows builds.

<!-- file: store/README.md -->
# example.com/app/store

```go
import "example.com/app/store"
```

Package store keeps key/value pairs in memory.

## Variables

```go
var ErrMissing = missing{}
```

ErrMissing is returned when a key is not in the store.

## Types

### type Store

```go
type Store struct {
	// contains filtered or unexported fields
}
```

Store is safe for concurrent use.

#### func New

```go
func New() *Store
```

New returns an empty Store.

#### func (*Store) Get

```go
func (s *Store) Get(key string) (string, error)
```

Get returns the value for key, or \[ErrMissing].

#### func (*Store) Set

```go
func (s *Store) Set(key, value string)
```

Set stores value under key.

<!-- file: tools/ext/README.md -->
# example.com/tools/ext

```go
import "example.com/tools/ext"
```

Package ext belongs to a nested module.

## Functions

### func Hook

```go
func Hook()
```

Hook does nothing.

