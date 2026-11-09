<!-- file: README.md -->
# Test results

8 tests in 4 suites in 2 reports: 5 passed, 1 failed, 1 error, 1 skipped, in 0.271s.

## Failing suites

- [reports/go-junit.xml](reports-go-junit-xml.md): lore-master/libs/confluence-client/storageformat (1 test)
- [reports/go-junit.xml](reports-go-junit-xml.md): lore-master/libs/markdown-workspace/documentparsing (1 test)

| Report | Suites | Tests | Passed | Failed | Errors | Skipped | Time |
|---|---:|---:|---:|---:|---:|---:|---:|
| [reports/go-junit.xml](reports-go-junit-xml.md) | 2 | 5 | 2 | 1 | 1 | 1 | 0.262s |
| [reports/jest-junit.xml](reports-jest-junit-xml.md) | 2 | 3 | 3 | 0 | 0 | 0 | 0.009s |

<!-- file: reports-go-junit-xml.md -->
# Test results: reports/go-junit.xml

5 tests in 2 suites: 2 passed, 1 failed, 1 error, 1 skipped, in 0.262s. Report: `reports/go-junit.xml`. Run at 2026-10-08T21:00:01+01:00.

## Failures

### lore-master/libs/confluence-client/storageformat

#### `lore-master/libs/confluence-client/storageformat.TestPanics`

Error.

```text
panic: runtime error: index out of range [3] with length 3

goroutine 7 [running]:
lore-master/libs/confluence-client/storageformat.TestPanics(0xc000102000)
	/repo/storage_test.go:42 +0x1d
```

### lore-master/libs/markdown-workspace/documentparsing

#### `lore-master/libs/markdown-workspace/documentparsing.TestGolden`

Failed.

````text
Failed

parse_document_use_case_test.go:107: rendered HTML differs from testdata/full.golden.html
         got: <h1>Getting started</h1> | a table cell
         want: ```code fence``` inside
````


## Suites

| Suite | Tests | Passed | Failed | Errors | Skipped | Time |
|---|---:|---:|---:|---:|---:|---:|
| lore-master/libs/confluence-client/storageformat | 2 | 1 | 0 | 1 | 0 | 0.232s |
| lore-master/libs/markdown-workspace/documentparsing | 3 | 1 | 1 | 0 | 1 | 0.030s |

<!-- file: reports-jest-junit-xml.md -->
# Test results: reports/jest-junit.xml

3 tests in 2 suites: 3 passed, 0 failed, 0 errors, 0 skipped, in 0.009s. Report: `reports/jest-junit.xml`. Run at 2026-10-08T20:00:00.

## Suites

| Suite | Tests | Passed | Failed | Errors | Skipped | Time |
|---|---:|---:|---:|---:|---:|---:|
| Pages view | 2 | 2 | 0 | 0 | 0 | 0.007s |
| Status policy | 1 | 1 | 0 | 0 | 0 | 0.002s |

