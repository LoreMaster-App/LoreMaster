package formatdetection

import "testing"

func TestSniffRecognisesEachFormatByItsContent(t *testing.T) {
	cases := []struct {
		name string
		file string
		body string
		want string
	}{
		{"junit suite", "reports/out.xml", `<?xml version="1.0"?><!-- c --><testsuite name="a" tests="1"></testsuite>`, TestResults},
		{"junit suites", "TEST-all.xml", `<testsuites><testsuite/></testsuites>`, TestResults},
		{"cobertura", "build/c.xml", `<?xml version="1.0"?><!DOCTYPE coverage SYSTEM "x.dtd"><coverage line-rate="0.5"></coverage>`, Coverage},
		{"lcov", "coverage/lcov.info", "TN:\nSF:src/a.ts\nDA:1,1\nend_of_record\n", Coverage},
		{"yaml openapi", "api/spec.yaml", "openapi: 3.0.3\ninfo:\n  title: x\n", OpenAPIDocs},
		{"json openapi", "api/spec.json", "{\n  \"openapi\": \"3.1.0\",\n  \"info\": {}\n}", OpenAPIDocs},
		{"swagger", "swagger.yml", "swagger: '2.0'\n", OpenAPIDocs},
		{"markdown", "docs/README.md", "# Title", MarkdownDoc},
		{"storybook", "storybook-static/index.json", `{"v":5,"entries":{"a--b":{"id":"a--b","type":"story","title":"A","importPath":"./a.tsx"}}}`, Storybook},
		{"other xml", "pom.xml", `<project><modelVersion>4</modelVersion></project>`, ""},
		{"plain json", "package.json", `{"name":"x","version":"1.0.0"}`, ""},
		{"yaml without a version key", "ci.yml", "name: build\non: push\n", ""},
		{"empty lcov", "x.info", "hello", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Sniff(tc.file, []byte(tc.body)); got != tc.want {
				t.Fatalf("Sniff(%q) = %q, want %q", tc.file, got, tc.want)
			}
		})
	}
}
