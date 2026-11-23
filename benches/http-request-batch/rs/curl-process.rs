use std::io::Write;
use std::process::{Command, Stdio};

pub fn run(server: &input::Server) -> u64 {
    let config = format!(
        "header = \"Authorization: {}\"\ndata-binary = \"{}\"\n",
        server.key,
        server.body.replace('"', "\\\"")
    );
    let mut received = 0;
    for _ in 0..server.requests {
        let mut child = Command::new("/usr/bin/curl")
            .args([
                "--disable",
                "--silent",
                "--max-time",
                "15",
                "--proto",
                "=https",
                "--cacert",
                &server.ca_file,
                "--header",
                "Content-Type: application/json",
                "--user-agent",
                "jmark",
                "--write-out",
                "\n%{http_code}",
                "--config",
                "-",
                &server.url,
            ])
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::null())
            .spawn()
            .expect("curl starts");
        child.stdin.take().expect("stdin").write_all(config.as_bytes()).expect("config");
        let output = child.wait_with_output().expect("curl finishes");
        let separator = output.stdout.iter().rposition(|byte| *byte == b'\n').expect("status line");
        if &output.stdout[separator + 1..] == b"200" {
            received += separator as u64;
        }
    }
    received
}
