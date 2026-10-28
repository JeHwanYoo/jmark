# One approach per file; the file stem is its result name. Define run(input) -> number.
def run(xs: list[int]) -> int:
    total = 0
    for x in xs:
        total += x
    return total
