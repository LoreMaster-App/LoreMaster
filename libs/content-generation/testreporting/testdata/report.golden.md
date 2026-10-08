<!-- file: README.md -->
# Test results

8 tests in 4 suites: 5 passed, 1 failed, 1 error, 1 skipped, in 0.271s.

## Failing tests

- [lore-master/libs/confluence-client/storageformat](lore-master-libs-confluence-client-storageformat.md): `lore-master/libs/confluence-client/storageformat.TestPanics` (error)
- [lore-master/libs/markdown-workspace/documentparsing](lore-master-libs-markdown-workspace-documentparsing.md): `lore-master/libs/markdown-workspace/documentparsing.TestGolden` (failed)

| Suite | Tests | Passed | Failed | Errors | Skipped | Time |
|---|---:|---:|---:|---:|---:|---:|
| [lore-master/libs/confluence-client/storageformat](lore-master-libs-confluence-client-storageformat.md) | 2 | 1 | 0 | 1 | 0 | 0.232s |
| [lore-master/libs/markdown-workspace/documentparsing](lore-master-libs-markdown-workspace-documentparsing.md) | 3 | 1 | 1 | 0 | 1 | 0.030s |
| [Pages view](pages-view.md) | 2 | 2 | 0 | 0 | 0 | 0.007s |
| [Status policy](status-policy.md) | 1 | 1 | 0 | 0 | 0 | 0.002s |

<!-- file: lore-master-libs-confluence-client-storageformat.md -->
# lore-master/libs/confluence-client/storageformat

2 tests: 1 passed, 0 failed, 1 error, 0 skipped, in 0.232s. Report: `reports/go-junit.xml`. Run at 2026-10-08T21:00:01+01:00.

## Failures

### `lore-master/libs/confluence-client/storageformat.TestPanics`

Error.

```text
panic: runtime error: index out of range [3] with length 3

goroutine 7 [running]:
lore-master/libs/confluence-client/storageformat.TestPanics(0xc000102000)
	/repo/storage_test.go:42 +0x1d
```


## Tests

| Test | Result | Time |
|---|---|---:|
| `lore-master/libs/confluence-client/storageformat.TestEscaping` | passed | 0.100s |
| `lore-master/libs/confluence-client/storageformat.TestPanics` | error | 0.132s |

<!-- file: lore-master-libs-markdown-workspace-documentparsing.md -->
# lore-master/libs/markdown-workspace/documentparsing

3 tests: 1 passed, 1 failed, 0 errors, 1 skipped, in 0.030s. Report: `reports/go-junit.xml`. Run at 2026-10-08T21:00:00+01:00.

## Failures

### `lore-master/libs/markdown-workspace/documentparsing.TestGolden`

Failed.

````text
Failed

parse_document_use_case_test.go:107: rendered HTML differs from testdata/full.golden.html
         got: <h1>Getting started</h1> | a table cell
         want: ```code fence``` inside
````


## Tests

| Test | Result | Time |
|---|---|---:|
| `lore-master/libs/markdown-workspace/documentparsing.TestParse` | passed | 0.010s |
| `lore-master/libs/markdown-workspace/documentparsing.TestGolden` | failed | 0.020s |
| `lore-master/libs/markdown-workspace/documentparsing.TestOnlyOnLinux` | skipped | 0.000s |

<!-- file: pages-view.md -->
# Pages view

2 tests: 2 passed, 0 failed, 0 errors, 0 skipped, in 0.007s. Report: `reports/jest-junit.xml`. Run at 2026-10-08T20:00:00.

## Tests

| Test | Result | Time |
|---|---|---:|
| `Pages view nests children` | passed | 0.003s |
| `Pages view sorts by file name` | passed | 0.004s |

<!-- file: status-policy.md -->
# Status policy

1 test: 1 passed, 0 failed, 0 errors, 0 skipped, in 0.002s. Report: `reports/jest-junit.xml`. Run at 2026-10-08T20:00:01.

## Tests

| Test | Result | Time |
|---|---|---:|
| `Status policy combines` | passed | 0.002s |

