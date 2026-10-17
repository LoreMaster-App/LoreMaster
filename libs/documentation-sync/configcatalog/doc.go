// Package configcatalog describes everything a user can configure: the storages a project syncs
// to (their fields, how a user signs in, what can be looked up on the platform), the kinds of
// generator, and the settings of the project and of the user. The editor shells build their forms,
// pickers, labels and settings pages from this description (the engine's config/describe method)
// instead of each carrying its own copy, so a new storage, generator or setting is one entry here
// and not a change in every shell. A shell stays an interface to the engine: it asks what can be
// configured and shows it.
package configcatalog
