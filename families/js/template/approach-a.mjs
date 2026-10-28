// One approach per file; the file stem is its result name. Export run(input) → number.
export function run(xs) {
  let sum = 0;
  for (let i = 0; i < xs.length; i++) sum += xs[i];
  return sum;
}
