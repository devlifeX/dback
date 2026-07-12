/** Unwrap CJS default export when Vite/Rolldown returns a module namespace object. */
export function importDefault<T>(mod: T): T {
  if (mod && typeof mod === 'object' && 'default' in mod) {
    const nested = (mod as { default?: T }).default
    if (nested !== undefined) return nested
  }
  return mod
}
