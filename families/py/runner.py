# Loop driver shared by every py-family runtime. Never reads a clock; the host times this process.
# argv: <input> <approach> <n> <iters> [<probe_out> <hook>...]  (probe args only in the untimed probe pass)
import importlib.util
import sys

sys.dont_write_bytecode = True


def load(path, name):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def main():
    input_path, approach_path, n, iters = sys.argv[1:5]
    probe_out, hook_paths = (sys.argv[5], sys.argv[6:]) if len(sys.argv) > 5 else (None, [])
    make_input = load(input_path, "_jmark_input").make_input
    run = load(approach_path, "_jmark_approach").run
    count = int(iters)
    data = make_input(int(n))

    hooks = [load(p, f"_jmark_hook_{i}") for i, p in enumerate(hook_paths)] if probe_out else []
    before = [h.snapshot() for h in hooks]
    sink = 0
    for _ in range(count):
        sink += run(data)
    after = [h.snapshot() for h in hooks]

    sys.stdout.write(f"{sink}\n")
    sys.stdout.flush()
    if probe_out:
        with open(probe_out, "w") as out:
            for start, end in zip(before, after):
                for k, v in end.items():
                    out.write(f"{k}\t{v - start.get(k, 0)}\n")


main()
