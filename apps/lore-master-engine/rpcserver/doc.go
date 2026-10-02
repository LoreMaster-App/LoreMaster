// Package rpcserver serves JSON-RPC 2.0 over one connection: it dispatches each request
// to its method concurrently, so a long sync never blocks a cancellation or the
// editor's answer to a reverse request, and turns errors into codes the editor reads.
package rpcserver
