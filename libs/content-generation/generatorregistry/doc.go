// Package generatorregistry says which generator runs for which configured type. It is the
// only place that knows every generator, so the runner and the settings stay independent of
// them and a new generator is one line here plus its own slice.
package generatorregistry
