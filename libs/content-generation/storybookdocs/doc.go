// Package storybookdocs is the storybook-docs generator: it renders every story of a Storybook
// in a real browser and writes one page per component with the screenshots, the props table
// and the story source. The images are ordinary PNG files, so any output that takes Markdown
// takes them. The Storybook is read from `source`: the URL of a running or deployed Storybook,
// or the folder of its static build, which is served locally for the render. The browser is the
// Chrome, Chromium or Edge installed on the machine, driven over the DevTools protocol; set
// LORE_MASTER_BROWSER to name one.
package storybookdocs
