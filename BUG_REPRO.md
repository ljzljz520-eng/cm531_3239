# BUG_REPRO

The following failures were observed while validating the initial project state.
Each section records what failed, how to reproduce it, and the complete command output.
They are preserved intentionally; only failing build gates are omitted from the generated Dockerfile.

## Failure 1: Go test (.)

- Observed problem: `Go test (.)` failed in the initial project state.
- Working directory: `.`
- Command: `cd /app && GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -count=1 ./...`
- Exit status: `1`

```text
ok  	example.com/mestransform	0.284s
ok  	example.com/mestransform/audit	0.024s
?   	example.com/mestransform/cmd/mesctl	[no test files]
ok  	example.com/mestransform/domain	0.005s
ok  	example.com/mestransform/mapping	0.008s
ok  	example.com/mestransform/parser	0.006s
--- FAIL: TestRepeatedRequestReturnsStoredResult (0.17s)
    conversion_test.go:72: first=conversion-1 second=conversion-2 records=2
FAIL
FAIL	example.com/mestransform/service	0.583s
ok  	example.com/mestransform/storage	0.080s
FAIL
```

## Architecture reproduction

### linux/amd64
- Go toolchain version: exit `0`
- Go build (.): exit `0`
- Go test (.): exit `1`
- Go run smoke (cmd/mesctl): exit `0`
### linux/arm64
- Go toolchain version: exit `0`
- Go build (.): exit `0`
- Go test (.): exit `1`
- Go run smoke (cmd/mesctl): exit `0`
