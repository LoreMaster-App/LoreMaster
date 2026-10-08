// Package dartdocs is the dart-docs generator: it runs dartdoc_json, which parses the Dart
// files under a package's lib/ folder into JSON, and renders that as a Markdown page per
// library file (titled by its package: import path), each with its classes, mixins, enums,
// extensions, functions and variables, their signatures and doc comments. dartdoc itself has no
// Markdown output, hence the renderer here. dartdoc_json is expected as a pub global tool; when
// it is not there the generator says what to install.
package dartdocs
