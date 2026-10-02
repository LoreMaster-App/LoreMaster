// Package pagecontent reads and writes Confluence pages. Cloud speaks REST v2 and Data
// Center and Server speak v1; page_api_client.go is the only file that knows which,
// and every use case goes through its pageAPI interface.
package pagecontent
