// Package authentication turns a credential into the Authorization header Confluence
// expects, and refuses a credential the edition cannot accept before any request is
// sent. It is pure: no HTTP, no storage. Secrets never appear in a credential's String.
package authentication
