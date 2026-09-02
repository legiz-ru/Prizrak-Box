// Minimal concurrency limiter (p-limit-compatible subset: `limit(fn)` runs fn
// once a slot is free and returns its result). Implemented locally instead of
// adding the `p-limit` package for a ~20-line utility.
export function pLimit(concurrency: number) {
  const queue: Array<() => void> = [];
  let active = 0;

  const runNext = () => {
    if (active >= concurrency || queue.length === 0) return;
    active++;
    const run = queue.shift()!;
    run();
  };

  return function limit<T>(fn: () => Promise<T>): Promise<T> {
    return new Promise<T>((resolve, reject) => {
      queue.push(() => {
        fn()
            .then(resolve, reject)
            .finally(() => {
              active--;
              runNext();
            });
      });
      runNext();
    });
  };
}

// Shared, module-level limiter for individual proxy-node delay tests
// (GET /proxies/:name/delay). A single global cap — rather than a fresh
// limiter per group — matters because the "test all groups" button already
// runs up to 3 groups concurrently (Proxies.vue testDelay); without a shared
// cap, each of those groups fans out an unbounded request per node, which
// can stack into hundreds of simultaneous dials through the local backend
// and starve the browser's connection pool until requests queue past the
// shared axios timeout and fail with a generic Network Error. Concurrency
// value matches Zashboard's reference implementation (pLimit(5)).
export const proxyNodeDelayLimiter = pLimit(5);
