// Models of how Go sizes backing arrays, so capacities shown on the stage
// match what the real packages would report.

// runtime size classes up to 32 KiB (runtime/sizeclasses.go).
const SIZE_CLASSES = [
  0, 8, 16, 24, 32, 48, 64, 80, 96, 112, 128, 144, 160, 176, 192, 208, 224,
  240, 256, 288, 320, 352, 384, 416, 448, 480, 512, 576, 640, 704, 768, 896,
  1024, 1152, 1280, 1408, 1536, 1792, 2048, 2304, 2688, 3072, 3200, 3456,
  4096, 4864, 5376, 6144, 6528, 6784, 6912, 8192, 9472, 9728, 10240, 10880,
  12288, 13568, 14336, 16384, 18432, 19072, 20480, 21760, 24576, 27264, 28672,
  32768,
];

function roundUpSize(bytes: number): number {
  for (const c of SIZE_CLASSES) if (c >= bytes) return c;
  return Math.ceil(bytes / 8192) * 8192;
}

/**
 * Capacity chosen by runtime.growslice when a slice of capacity `oldCap` must
 * hold `need` elements of `elem` bytes (append, slices.Grow).
 */
export function appendCap(oldCap: number, need: number, elem = 8): number {
  if (need <= oldCap) return oldCap;
  let cap = oldCap;
  const dbl = oldCap * 2;
  if (need > dbl) cap = need;
  else if (oldCap < 256) cap = dbl;
  else while (cap < need) cap += (cap + 3 * 256) >> 2;
  return Math.floor(roundUpSize(cap * elem) / elem);
}

/**
 * internal/ring nextCap: starting from 1, double below 1024 and grow by a
 * quarter at or above it, until the capacity reaches `need`.
 */
export function ringCap(current: number, need: number): number {
  let c = Math.max(current, 1);
  while (c < need) c = c < 1024 ? c * 2 : c + Math.floor(c / 4);
  return c;
}
