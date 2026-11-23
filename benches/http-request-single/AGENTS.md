Compare one HTTPS POST sent by a curl child process vs in-process Rust clients (reqwest, ureq, isahc). Metric: wall/CPU time, peak RSS. Input: local TLS server, 2 KiB reply, same request.
