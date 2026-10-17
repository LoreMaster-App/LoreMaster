package rpcprotocol

// MethodConfigDescribe returns everything that can be configured, so an editor shell builds its
// forms, pickers, labels and settings pages from the engine instead of carrying its own copy:
// the storages a project syncs to (their fields, how a user signs in, what can be looked up), the
// kinds of generator, and the settings of the project and of the user. It needs no session. The
// result is configcatalog.Description; its fields are documented there and in docs/configuration.md.
const MethodConfigDescribe = "config/describe"
