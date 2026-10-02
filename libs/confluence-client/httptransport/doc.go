// Package httptransport is the one place the client speaks HTTP: authentication header,
// JSON and multipart bodies, retries that respect Confluence's rate limits, both
// pagination styles, and logs that never carry a header or a body. It depends on
// nothing above it: it takes the base URL and the Authorization value as strings.
package httptransport
