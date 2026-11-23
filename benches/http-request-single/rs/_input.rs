use std::io::{BufRead, BufReader, Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::Arc;

pub struct Server {
    pub url: String,
    pub ca_pem: Vec<u8>,
    pub ca_file: String,
    pub key: String,
    pub body: String,
    pub requests: usize,
}

const RESPONSE_BYTES: usize = 2048;

pub fn make_input(n: usize) -> Server {
    let _ = rustls::crypto::aws_lc_rs::default_provider().install_default();
    let issued = rcgen::generate_simple_self_signed(vec!["localhost".to_string()]).expect("certificate");
    let ca_pem = issued.cert.pem();
    let ca_file = std::env::temp_dir().join("jmark-http-localhost.pem");
    std::fs::write(&ca_file, &ca_pem).expect("certificate file");
    let key = rustls::pki_types::PrivateKeyDer::Pkcs8(issued.signing_key.serialize_der().into());
    let config = Arc::new(
        rustls::ServerConfig::builder()
            .with_no_client_auth()
            .with_single_cert(vec![issued.cert.der().clone()], key)
            .expect("server config"),
    );
    let listener = TcpListener::bind("127.0.0.1:0").expect("listener");
    let port = listener.local_addr().expect("address").port();
    std::thread::spawn(move || {
        for stream in listener.incoming().flatten() {
            let config = config.clone();
            std::thread::spawn(move || serve(stream, config));
        }
    });
    let query = "query Issues { issues(first: 50, orderBy: updatedAt) { nodes { id identifier title priority state { name type } } pageInfo { hasNextPage endCursor } } }";
    Server {
        url: format!("https://localhost:{port}/graphql"),
        ca_pem: ca_pem.into_bytes(),
        ca_file: ca_file.to_string_lossy().into_owned(),
        key: "fixture_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx".to_string(),
        body: format!("{{\"query\":\"{query}\",\"variables\":{{}}}}"),
        requests: n,
    }
}

fn serve(stream: TcpStream, config: Arc<rustls::ServerConfig>) {
    let Ok(connection) = rustls::ServerConnection::new(config) else {
        return;
    };
    let mut reader = BufReader::new(rustls::StreamOwned::new(connection, stream));
    let payload = vec![b'x'; RESPONSE_BYTES];
    loop {
        let mut line = String::new();
        if reader.read_line(&mut line).unwrap_or(0) == 0 {
            return;
        }
        let mut length = 0;
        loop {
            line.clear();
            if reader.read_line(&mut line).unwrap_or(0) == 0 {
                return;
            }
            if line == "\r\n" {
                break;
            }
            if let Some((name, value)) = line.split_once(':') {
                if name.eq_ignore_ascii_case("content-length") {
                    length = value.trim().parse().unwrap_or(0);
                }
            }
        }
        let mut request = vec![0; length];
        if reader.read_exact(&mut request).is_err() {
            return;
        }
        let head = format!("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: {RESPONSE_BYTES}\r\n\r\n");
        let stream = reader.get_mut();
        if stream.write_all(head.as_bytes()).is_err() || stream.write_all(&payload).is_err() || stream.flush().is_err() {
            return;
        }
    }
}
