# Embedded pgo-fleet profiler

```go
import "github.com/t2bot/pgo-fleet/embedded"
```

## Usage

See [godoc](http://godoc.org/github.com/t2bot/pgo-fleet/embedded) for detailed docs.

Quickstart:

```go
endpoint, err := pgo.NewCollectorEndpoint("https://collector.example.org/v1/submit", "YourSecretKeyGoesHere")
if err != nil {
	panic(err)
}

// Run a profile about once an hour for 5 minutes, submitting to the given endpoint
pgo.Enable(1 * time.Hour, 5 * time.Minute, endpoint)

// Stop collecting profiles
pgo.Disable()
```
