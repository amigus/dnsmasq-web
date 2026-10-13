# Testing

## Authoritative Sources

- [Go testing package](https://pkg.go.dev/testing)
- [httptest](https://pkg.go.dev/net/http/httptest)
- [testify/assert](https://pkg.go.dev/github.com/stretchr/testify/assert) (the project's assertion lib)
- [Table-driven tests in Go](https://go.dev/wiki/TableDrivenTests) (community convention)

## Test Layout

| File | What it tests | Style |
|---|---|---|
| `main.go` (no test) | Startup wiring | Smoke-tested via the built `dnsmasq-web` binary |
| `daemon_test.go` | `id`, `Listener`, `RunDaemon` | Pure Go + a `GO_WANT_HELPER_PROCESS` helper |
| `lease_database_test.go` | `/leases`, `/addresses`, `/devices`, `/clients` | `httptest` + `:memory:` SQLite |
| `lease_database_test_data.go` | Shared `testDatabaseSQL` | Constant string fed into `:memory:` |
| `host_dir_test.go` | `/reservations` CRUD | `httptest` + a per-test `test_hosts/` dir |
| `token_test.go` | `auth.Checker` semantics | Pure Go |
| `token_gin_test.go` | Middleware + publisher | `httptest` + `gin.New()` |
| `cmd/root_test.go` | `applyFlagOverrides` precedence | Cobra flags directly |
| `internal/output/table_test.go` | Table formatter | Pure Go, includes a width-scaling case |

No mocks. No external test containers. The SQLite tests use `:memory:`; the reservation tests use a per-test temp directory cleaned up via `defer os.RemoveAll`.

## Conventions in Use

- **Testify `assert`, not `require`, for the happy path.** The codebase uses `assert.Equal`, `assert.NoError`, `assert.True`, `assert.NotEmpty`. `require` is not used — there's no test that needs to abort on the first failure. Match this.
- **`t.Fatalf` is used directly** in `daemon_test.go` and `lease_database_test.go` for setup errors (DB open, PID file creation) where a subsequent assertion would just panic anyway. Match this.
- **Helper-process tests** for code that calls `os.Args`, `os.Exit`, or signals. See `TestRunDaemonHelperProcess` — the test sets `GO_WANT_HELPER_PROCESS=1` in the child's env, the child branches on it, and the parent asserts on the side effects. Standard Go pattern, no extra dep.
- **No benchmarks** in the repo. Don't add one without a reason.

## Test Names

The pattern is `TestXxx` for top-level tests and `TestXxxYyy` for the same function under a different scenario. See `TestTokenChecker_Get` / `TestTokenChecker_GetAfterTimeout` / `TestTokenChecker_GetTooManyTokens` — the underscore + scenario suffix is the convention.

## Handler Tests — The Canonical Pattern

```go
func setupRouter() *gin.Engine {
    r := gin.Default()
    db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    db.Exec(testDatabaseSQL)
    return LeaseDatabase(r, db, 30, "192.168.1.0/24")
}

func TestLeasesEndpoint(t *testing.T) {
    router := setupRouter()

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/leases", nil)
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    // ... assert body
}
```

Notes:
- **`gin.Default()` includes the logger middleware**, which prints to stdout during tests. `host_dir_test.go` switches to `gin.SetMode(gin.TestMode)` and `gin.New()` to suppress this. Use `gin.New()` for new test setups unless you specifically want the default middleware.
- **Re-creating the router per test** is the existing style. Don't share state across tests — even if it would be faster, the per-test fixture makes failures easier to localize.
- **Anonymous struct as response type** mirrors what the handler does. Don't import the handler's internal types into the test.
- **Cleanup via `defer`** in tests that create files: `defer os.RemoveAll("./test_hosts")` (the host-dir tests do this). If you add a test that creates a file, add a cleanup.

## Assertion Patterns

For HTTP responses, the common sequence is:

```go
assert.Equal(t, http.StatusOK, w.Code)
assert.Equal(t, http.StatusBadRequest, w.Code)
assert.FileExists(t, filepath.Join("./test_hosts", "00:1a:2b:3c:4d:5e"))
assert.NoFileExists(t, filepath.Join("./test_hosts", "00:1a:2b:3c:4d:5e"))
assert.Contains(t, string(content), "192.168.1.101")
```

Use `Contains` when you want to assert on a fragment (a substring of a reservation file). Use `Equal` only when the exact body is the contract.

For unit tests in `internal/`:

```go
assert.Equal(t, token1, token2, "Expected the same token")
assert.True(t, ttc.Check(token1), "Expected token to be valid")
assert.False(t, ttc.Check("not-a-real-token"), "Expected an unknown token to be invalid")
```

The message after the values is a habit here — keep it, it makes test failures readable.

## What to Test

- **Every new HTTP handler**: one happy-path test and at least one validation-failure test.
- **Every new query in `lease_database.go`**: a test against `:memory:` SQLite using the existing `testDatabaseSQL` or a focused new fixture.
- **Every new validation rule in `host_dir.go`**: a test that submits the bad input and asserts the status code.
- **New public functions in `internal/`**: at least one direct test (the auth table tests are the model).
- **CLI flag precedence changes**: extend `cmd/root_test.go`.

## What *Not* to Test

- **Build output**. CI's smoke test (`./dnsmasq-web --help`, `server -V`) covers this; no need to duplicate in unit tests.
- **The output formatter's exact string output** beyond header presence and a width-scaling case. The reflection-based table is intentionally lossy.
- **Pure stdlib wrappers**. The `id` function does a tiny bit of name/numeric lookup logic; the existing test covers both branches. Don't write a test for `lookupGroup` in isolation.

## Running Tests

```bash
go test ./...                    # all
go test -v ./...                 # verbose, used in CI
go test -run TestLeases ./...    # single test
go test -run TestToken ./...     # by pattern
go test -cover ./...             # coverage
```

No `Makefile` test target. `mage` does not wrap `go test`. The CI workflow is the canonical command.

## Fuzzing / Property Tests

None in the repo. The handlers are simple enough that this is fine. If you add a parser (e.g. for a new env-var format), add a `FuzzXxx` test in the same file.

## Test Files and Tags

No build tags on test files. `Magefile.go` uses `//go:build mage` to keep the mage target out of the normal build — nothing else does. Don't add tags to tests.
