# Existing bootstrap tests race with initial fetch scheduling

Observed on unchanged main (`cda595b`) before test refactoring, and again on this
branch when instrumenting the full suite. This is a separate test isolation issue;
no production behavior was changed to address it.

## Reproduction

```sh
cd backend
go test ./... -coverpkg=./... -coverprofile=/tmp/backend.cover
```

The failure is intermittent. In `internal/bootstrap/app_sqlite_test.go`,
`TestSQLiteApplicationSupportsAPIWritesFetchJobsAndReopen` expected `201` at line
88 but received `409`. The executor-alongside-writes test also returned `409`
instead of `201` at line 165. Fetch history tests occasionally encountered an extra
job or the same conflict.

`NewWithConfig` starts the fetch loops. The scheduler runs a round immediately,
even when its interval is one hour. If the test creates its RSS source before that
round reads sources, the scheduler can enqueue a job before the test's explicit
POST. The POST then correctly returns `source already has an unfinished fetch job`.
Coverage instrumentation makes this interleaving easier to encounter.

A future test-only fixture should control the initial scheduler round or coordinate
its completion before creating subscriptions. Increasing intervals or adding sleeps
does not establish that ordering. The new Entry/Source module HTTP fixture avoids
starting workers; existing bootstrap wiring and worker integration tests remain.

The final ordinary and race suites both passed, but this does not resolve the
intermittent bootstrap issue. No production bug was established by this finding.
