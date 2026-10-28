// Builds the shared input from n. Pure: no clocks, no JSON, no IO.
export function makeInput(n) {
  return Array.from({ length: n }, (_, i) => i);
}
