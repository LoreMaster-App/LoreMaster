// Package tsdocs is the ts-docs generator: it runs TypeDoc, with its Markdown plugin, over the
// TypeScript and JavaScript projects of the workspace and writes the result as pages — an
// index, and a page per class, interface, function and variable — under the output folder,
// one section per project. TypeDoc is expected in the workspace's node_modules; when it is
// not there the generator says what to install.
package tsdocs
