export function run(xs) {
  return xs
    .map((x) => x * 2)
    .filter((x) => x > 10)
    .reduce((sum, x) => sum + x, 0);
}
