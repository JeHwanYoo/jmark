Compare 20 sequential HTTPS POSTs: one curl child process each vs one reused Rust client (reqwest, ureq, isahc). Metric: wall/CPU time, peak RSS. Input: local TLS server, 2 KiB reply.
