# pgo-fleet

Collects profiles to enable [PGO](https://go.dev/doc/pgo) builds in Go projects. This is primarily intended for fleets
of Go processes, either to inform better default builds in open source projects or to produce performance-tailored builds.

Please consider [donating](https://t2bot.io/donations/) to t2bot.io if you deploy this.

## Architecture

Somewhere, a `collector` process is deployed. Profiles are submitted to this collector for later merging/retrieval in
the build process. Profiles are typically sampled with the `embedded` profiler, but the collector will accept manual
profiles as well (such as via `curl`).

When it's time to build the Go project, or at whatever frequency is desirable, a merged profile is requested from the
collector and placed into the build pipeline. It's the caller's responsibility to figure out how to get the profile into
said build pipeline ;)

Open source projects can (anonymously) collect profiles from users to improve their published build artifacts, and consumers
might deploy their own collector to create tailored builds.

## Usage: Collector

The collector provided here assumes that profiles are relatively infrequent (dozens per hour rather than dozens per second),
and that losing some profiles is *not* a problem. For example, if the container running the collector were to crash or
become unavailable due to high load, some profiles might be dropped. This shouldn't affect the PGO-enabled build too badly
though, provided some profiles do actually make it into the build pipeline.

It's also assumed that there is a dedicated collector process for each project. The collector will verify that what it
received is in fact a CPU profile, but will not attempt to identify the Go project it is for. If multiple projects are
sent to a single collector, the merged profiles will be extremely weird.

The collector can be run either inside private infrastructure or be exposed as public-facing via a reverse proxy. Regardless
of what network it's servicing, only give auth keys to trusted sources. "Trusted" here meaning processes/people who won't
provide useless CPU profiles to you.

Note that if exposed externally that the reverse proxy will be responsible for rate limiting. The collector will dutifully
attempt to respond to *every* request it receives, good or bad.

If you'd like to write your own collector, see the API definitions in [`./collector/README.md`](./collector/README.md).

Now that the disclaimers are out of the way, the actual collector can be configured using environment variables:

```bash
# Listening address for HTTP server.
# In the official Docker container, this is set to ":8080" for exposure to the host.
# Otherwise, this defaults to the value shown here.
export PGOF_BIND_ADDRESS="127.0.0.1:8080"

# Where to store all the profiles. Process must have read and write access, and will
# attempt to create the directory if it doesn't exist.
# In the official Docker container, this is set to "/data" for mounting.
# Otherwise, this defaults to the value shown here.
export PGOF_DIRECTORY="./profiles"

# The file which has auth keys (1 per line) that are valid for submitting profiles.
# Note that profiles are not associated with the keys themselves, but if multiple
# submission sources are in use then individually rotatable keys can be useful.
# Defaults to the value shown here.
export PGOF_SUBMIT_AUTH_KEYS_FILE="/secret/submit_keys"

# Like the submit keys file above, but for the merge API instead.
# Defaults to the value shown here.
export PGOF_MERGE_AUTH_KEYS_FILE="/secret/merge_keys"
```

Any changes to the environment variables require the process to be restarted. This includes adding/removing/rotating keys.

This documentation does not include directions for how to deploy an HTTP service behind a reverse proxy, sorry. Once deployed
though, the submit URL will be at the path `/v1/submit` and merge at `/v1/merge`. See [`./collector/README.md`](./collector/README.md)
for more details on the API.

**Tip**: When requesting a merge, set `?and_combine=true` to reduce the file count on disk.

A health endpoint is exposed at `/healthz`. It will return `200 OK` with body `OK`.

Pre-built Docker images are available [here](https://github.com/t2bot/pgo-fleet/pkgs/container/pgo-fleet).

## Usage: Embedded Profiler

See [`./embedded/README.md`](./embedded/README.md) for usage.
