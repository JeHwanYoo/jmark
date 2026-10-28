def run(xs: list[int]) -> int:
    total = 0
    for v in xs:
        value = v * 2
        if value > 10:
            total += value
    return total
