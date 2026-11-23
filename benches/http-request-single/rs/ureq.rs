pub fn run(server: &input::Server) -> u64 {
    let certificate = ureq::tls::Certificate::from_pem(&server.ca_pem).expect("certificate").to_owned();
    let tls = ureq::tls::TlsConfig::builder()
        .root_certs(ureq::tls::RootCerts::new_with_certs(&[certificate]))
        .build();
    let agent: ureq::Agent = ureq::Agent::config_builder()
        .tls_config(tls)
        .user_agent("jmark")
        .build()
        .into();
    let mut received = 0;
    for _ in 0..server.requests {
        let mut response = agent
            .post(&server.url)
            .header("Authorization", &server.key)
            .header("Content-Type", "application/json")
            .send(server.body.as_bytes())
            .expect("response");
        let body = response.body_mut().read_to_vec().expect("body");
        if response.status() == 200 {
            received += body.len() as u64;
        }
    }
    received
}
