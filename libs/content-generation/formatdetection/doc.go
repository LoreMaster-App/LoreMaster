// Package formatdetection tells which generator reads a set of files by looking at them, so a
// generator does not need to be told its type: a JUnit report is recognised by its root
// element, a coverage report by its structure, an OpenAPI description by its version key,
// Markdown by its extension and a Storybook by its story index. The names of the files never
// decide on their own. It is what makes "run this command, read what it wrote" work without the
// user naming a format.
package formatdetection
