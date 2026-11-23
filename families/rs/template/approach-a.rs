pub fn run(xs: &[u64]) -> u64 {
    let mut sum = 0;
    for x in xs {
        sum += x;
    }
    sum
}
