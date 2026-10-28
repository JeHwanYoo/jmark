def run(xs: list[int]) -> int:
    return sum(x for x in (v * 2 for v in xs) if x > 10)
