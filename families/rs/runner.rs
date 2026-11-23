// Loop driver shared by every rs-family runtime. Never reads a clock; the host times this process.
// The build compiles one binary per approach with JMARK_INPUT and JMARK_APPROACH set to source paths.
// argv: <n> <iters>
#[allow(dead_code)]
mod input {
    include!(env!("JMARK_INPUT"));
}

mod approach {
    #[allow(unused_imports)]
    use super::input;
    include!(env!("JMARK_APPROACH"));
}

fn main() {
    let args: Vec<String> = std::env::args().collect();
    let n: usize = args[1].parse().expect("n is an integer");
    let iters: usize = args[2].parse().expect("iters is an integer");
    let data = input::make_input(n);
    let mut sink: u64 = 0;
    for _ in 0..iters {
        sink += approach::run(&data);
    }
    println!("{sink}");
}
