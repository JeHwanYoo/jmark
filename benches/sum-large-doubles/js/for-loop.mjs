export function run(xs) {
  const length = xs.length;
  let sum = 0;
  for (let i = 0; i < length; i++) {
    const value = xs[i] * 2;
    if (value > 10) sum += value;
  }
  return sum;
}
