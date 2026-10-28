export function makeInput(n) {
  return Array.from({ length: n }, (_, i) => (i % 20) + 1);
}
