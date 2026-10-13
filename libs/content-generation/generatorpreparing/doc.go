// Package generatorpreparing runs a generator's optional prepare command: the step that
// produces what the generator reads (a Storybook build, a test run, a docs build). It runs only
// when asked, in the workspace folder, through the platform's shell, with a time limit; a
// failure carries the tail of the command's output so the cause is visible.
package generatorpreparing
