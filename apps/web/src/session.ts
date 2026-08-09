const syntheticQueryCache = new Map<string, string>();
const syntheticSignedOutKey = "atlas.synthetic.signed-out";

export interface ClientStorage {
  readonly length: number;
  getItem(key: string): string | null;
  key(index: number): string | null;
  removeItem(key: string): void;
  setItem(key: string, value: string): void;
}

export type BrowserStorageSummary = {
  localAtlasEntries: number;
  sessionAtlasEntries: number;
  unexpectedAtlasEntries: boolean;
};

export type PageRestoreKind = "ordinary" | "history-reload" | "bfcache";
export type SecurityBoundary = "logout" | "session-invalid" | "tenant-switch" | "membership-change";

export type SecurityBoundaryChannel = {
  publish(boundary: SecurityBoundary): void;
  close(): void;
};

export function cacheSyntheticValue(key: string, value: string): void {
  syntheticQueryCache.set(key, value);
}

export function queryCacheSize(): number {
  return syntheticQueryCache.size;
}

export function clearSensitiveClientState(): void {
  syntheticQueryCache.clear();
}

export function markSyntheticSignedOut(storage: ClientStorage): void {
  storage.setItem(syntheticSignedOutKey, "1");
}

export function clearSyntheticSignedOut(storage: ClientStorage): void {
  storage.removeItem(syntheticSignedOutKey);
}

export function isSyntheticSignedOut(storage: ClientStorage): boolean {
  try {
    return storage.getItem(syntheticSignedOutKey) === "1";
  } catch {
    return true;
  }
}

export function summarizeBrowserStorage(
  localStorage: ClientStorage,
  sessionStorage: ClientStorage,
): BrowserStorageSummary {
  try {
    const localAtlasKeys = atlasKeys(localStorage);
    const sessionAtlasKeys = atlasKeys(sessionStorage);
    const sessionGuardIsExact =
      sessionAtlasKeys.length === 0 ||
      (
        sessionAtlasKeys.length === 1 &&
        sessionAtlasKeys[0] === syntheticSignedOutKey &&
        sessionStorage.getItem(syntheticSignedOutKey) === "1"
      );
    return {
      localAtlasEntries: localAtlasKeys.length,
      sessionAtlasEntries: sessionAtlasKeys.length,
      unexpectedAtlasEntries: localAtlasKeys.length !== 0 || !sessionGuardIsExact,
    };
  } catch {
    return {
      localAtlasEntries: -1,
      sessionAtlasEntries: -1,
      unexpectedAtlasEntries: true,
    };
  }
}

export function classifyPageRestore(
  persisted: boolean,
  navigationType: string | undefined,
): PageRestoreKind {
  if (persisted) {
    return "bfcache";
  }
  if (navigationType === "back_forward") {
    return "history-reload";
  }
  return "ordinary";
}

export function isProtectedShell(path: string): boolean {
  return path === "/customer" || path === "/merchant" || path === "/workforce";
}

export function canAccessShell(sessionActive: boolean, path: string): boolean {
  return sessionActive || !isProtectedShell(path);
}

export function createSecurityBoundaryChannel(
  onBoundary: (boundary: SecurityBoundary) => void,
  channelFactory: ((name: string) => BroadcastChannel) | undefined = defaultChannelFactory(),
): SecurityBoundaryChannel {
  if (!channelFactory) {
    return {publish() {}, close() {}};
  }
  const channel = channelFactory("atlas.security-boundary.v1");
  channel.addEventListener("message", (event: MessageEvent<unknown>) => {
    if (isSecurityBoundary(event.data)) {
      onBoundary(event.data);
    }
  });
  return {
    publish(boundary) {
      channel.postMessage(boundary);
    },
    close() {
      channel.close();
    },
  };
}

function atlasKeys(storage: ClientStorage): string[] {
  const keys: string[] = [];
  for (let index = 0; index < storage.length; index += 1) {
    const key = storage.key(index);
    if (key?.startsWith("atlas.")) {
      keys.push(key);
    }
  }
  return keys.sort();
}

function defaultChannelFactory(): ((name: string) => BroadcastChannel) | undefined {
  if (typeof BroadcastChannel === "undefined") {
    return undefined;
  }
  return (name) => new BroadcastChannel(name);
}

function isSecurityBoundary(value: unknown): value is SecurityBoundary {
  return value === "logout" || value === "session-invalid" || value === "tenant-switch" || value === "membership-change";
}
