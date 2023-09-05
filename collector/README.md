# Collector

See the parent [README](../README.md) for usage.

A health endpoint which always returns `200 OK` is available at `/healthz`.

## API

All requests are authenticated with an `Authorization` header containing a `Bearer` token.

### `POST /v1/submit`

Request body: `<pprof profile bytes>`

Submits a pprof CPU profile. The profile should be validated to ensure it is actually a pprof CPU profile.

Returns `200 OK` or `204 No Content` to indicate success, any other status code for failure. No useful response
body should be expected.

### `POST /v1/merge?and_combine=true|false`

Requests a merged pprof CPU profile of all stored profiles. When `and_combine` is `true`, the merged profile
will replace all the stored profiles on the backend. When `and_combine` is `false` (or not `true`, the default),
the stored profiles will be kept as-is.

Combining profiles is largely a space-saving measure for the backend. If 240 profiles are submitted each day, then
it may be in the caller's interest to combine that down to 1 profile for the next day.

Returns `200 OK` with the pprof profile to indicate success, any other status code for failure.
