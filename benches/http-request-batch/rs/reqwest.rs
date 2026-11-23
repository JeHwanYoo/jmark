use std::io::Read;

pub fn run(server: &input::Server) -> u64 {
    let certificate = reqwest::Certificate::from_pem(&server.ca_pem).expect("certificate");
    let client = reqwest::blocking::Client::builder()
        .tls_certs_only([certificate])
        .user_agent("jmark")
        .build()
        .expect("client");
    let mut received = 0;
    for _ in 0..server.requests {
        let mut response = client
            .post(&server.url)
            .header("Authorization", &server.key)
            .header("Content-Type", "application/json")
            .body(server.body.clone())
            .send()
            .expect("response");
        let mut body = Vec::new();
        response.read_to_end(&mut body).expect("body");
        if response.status() == 200 {
            received += body.len() as u64;
        }
    }
    received
}
