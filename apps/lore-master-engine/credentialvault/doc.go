// Package credentialvault keeps Confluence connections (the site, who signs in, and the
// credential) in the operating system's keychain: Windows Credential Manager, macOS Keychain or
// the Linux Secret Service. Because it belongs to the user and not to an editor profile, the
// extension in any VS Code profile and the command line see the same connections, so a token is
// entered once and not again for every profile and every shell.
//
// It is opt-in: nothing is written here unless the user asks for it.
package credentialvault
