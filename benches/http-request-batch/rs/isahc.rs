use isahc::config::Configurable;
use isahc::tls::{TlsConfig, TrustStore};
use isahc::{ReadResponseExt, Request};

pub fn run(server: &input::Server) -> u64 {
    let trust = TrustStore::builder().certificate_from_pem(&server.ca_pem).build();
    let client = isahc::HttpClient::builder()
        .tls_config(TlsConfig::builder().trust_store(trust).build())
        .default_header("User-Agent", "jmark")
        .build()
        .expect("client");
    let mut received = 0;
    for _ in 0..server.requests {
        let request = Request::post(&server.url)
            .header("Authorization", &server.key)
            .header("Content-Type", "application/json")
            .body(server.body.clone())
            .expect("request");
        let mut response = client.send(request).expect("response");
        let body = response.bytes().expect("body");
        if response.status() == 200 {
            received += body.len() as u64;
        }
    }
    received
}
