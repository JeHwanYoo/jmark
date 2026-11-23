# curl child process vs Rust HTTP clients

Measured on 2026-11-23 to decide whether Jenesis should keep calling external HTTP APIs through the system `curl` or move to an in-process Rust client.

## Setup

| Item | Value |
| --- | --- |
| Host | Apple M4 Pro, 14 cores, macOS (Darwin 27.2.0), hyperfine 2.0.0 |
| Toolchain | rustc 1.97.1, release build |
| Server | HTTPS/1.1 server in the benchmark process on `localhost`, self-signed certificate trusted by every client, 2 KiB response |
| Request | `POST` with a GraphQL body and an `Authorization` header, like a Linear API call |
| `curl-process` | `/usr/bin/curl` started once per request with the flags Jenesis uses; config read from stdin |
| `reqwest` | reqwest 0.13.5 blocking client, rustls |
| `ureq` | ureq 3.5.2 agent, rustls |
| `isahc` | isahc 2.0.2, libcurl 8.22 linked in-process, rustls |

Each Rust client is created once per operation and reused for every request in it. Two projects measure the same sources:

- `http-request-single`: one request per operation, so every approach pays connection setup.
- `http-request-batch`: 20 sequential requests per operation, so libraries reuse one connection while `curl-process` starts a new process and TLS session each time.

Raw results and source snapshots: [single](../../artifacts/public/http-request-single/20261123-113129-2460691241/report.html), [batch](../../artifacts/public/http-request-batch/20261123-113253-3667199068/report.html).

## Results

### One request per operation

| Approach | Wall time | CPU time | Peak RSS |
| --- | ---: | ---: | ---: |
| curl-process | 12.83 ms | 5.96 ms | 5.0 MiB |
| isahc | 0.65 ms | 0.56 ms | 12.2 MiB |
| reqwest | 0.87 ms | 0.61 ms | 11.1 MiB |
| ureq | 0.62 ms | 0.45 ms | 7.1 MiB |

### 20 requests per operation

| Approach | Wall time | Per request | CPU time | Peak RSS |
| --- | ---: | ---: | ---: | ---: |
| curl-process | 237.9 ms | 11.9 ms | 115.3 ms | 5.0 MiB |
| isahc | 2.12 ms | 0.11 ms | 1.76 ms | 12.3 MiB |
| reqwest | 1.97 ms | 0.10 ms | 1.56 ms | 11.1 MiB |
| ureq | 3.95 ms | 0.20 ms | 2.00 ms | 7.2 MiB |

### Binary size

Size of the release runner. Every runner already links rustls and the test server, so the increase for Jenesis, which has no TLS stack today, would be larger.

| Approach | Size | Increase |
| --- | ---: | ---: |
| curl-process | 3.7 MiB | — |
| ureq | 5.0 MiB | +1.4 MiB |
| isahc | 6.1 MiB | +2.4 MiB |
| reqwest | 6.2 MiB | +2.5 MiB |

## Real-network context

Five unauthenticated requests from the same machine to `https://api.linear.app/graphql` with `curl -w` (not a jmark run): TCP connect 10–93 ms, TLS done at 25–109 ms, total 0.49–0.62 s. The server's response time dominates a Linear call.

## Findings

- Starting `curl` costs about 12 ms of wall time and 6 ms of CPU per request on this machine. An in-process client needs 0.6–0.9 ms for the first request and 0.1–0.2 ms for each reused request.
- Localhost hides network round trips. Against Linear, a reused connection also skips the TCP and TLS handshakes, about 25 ms per request on this network.
- A switch would save roughly 12 ms on a single Linear call and about 35–40 ms per call in a burst. Linear calls take about 500 ms, so the saving is 2–8% of the call.
- Peak RSS of `curl-process` counts only the benchmark process; each `curl` child uses its own memory. The libraries add 2–7 MiB to the process and 1.4–2.5 MiB to the binary.
- Among the libraries, reqwest and isahc are fastest when reusing a connection, and ureq is the smallest. reqwest's blocking client runs its own tokio runtime, which conflicts with Jenesis's one-runtime rule; ureq and isahc do not.

## Limits

- The server and clients share one machine and one process, so CPU time includes the server's share for every approach.
- The server speaks HTTP/1.1 only. Linear supports HTTP/2, which `curl`, reqwest, and isahc could negotiate.
- The Rust toolchain comes from `PATH`, not mise, so the version is recorded but not pinned.
