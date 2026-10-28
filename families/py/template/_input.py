# Builds the shared input from n. Pure: no clocks, no JSON, no IO.
def make_input(n: int) -> list[int]:
    return list(range(n))
