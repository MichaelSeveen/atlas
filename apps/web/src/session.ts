const syntheticQueryCache = new Map<string, string>();

export function cacheSyntheticValue(key: string, value: string): void {
  syntheticQueryCache.set(key, value);
}

export function queryCacheSize(): number {
  return syntheticQueryCache.size;
}

export function clearSensitiveClientState(): void {
  syntheticQueryCache.clear();
}

export function isProtectedShell(path: string): boolean {
  return path === "/customer" || path === "/merchant" || path === "/workforce";
}

export function canAccessShell(sessionActive: boolean, path: string): boolean {
  return sessionActive || !isProtectedShell(path);
}
