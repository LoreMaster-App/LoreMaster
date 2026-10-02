// Package rpcprotocol is the engine's wire contract: every JSON-RPC method's name and
// its request and response shapes, and the error codes. It is the single source of
// truth; the editor shells mirror it (#57), so a field renamed here is renamed there.
//
// Transport: JSON-RPC 2.0 over the engine's stdin and stdout, each message framed by a
// "Content-Length: <bytes>\r\n\r\n" header, as the Language Server Protocol does, which
// vscode-jsonrpc speaks by default. stderr carries the engine's log, never a secret.
package rpcprotocol
