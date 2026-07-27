import {describe, expect, test} from "bun:test";
import {
  cacheSyntheticValue,
  canAccessShell,
  classifyPageRestore,
  clearSensitiveClientState,
  clearSyntheticSignedOut,
  isProtectedShell,
  isSyntheticSignedOut,
  markSyntheticSignedOut,
  queryCacheSize,
  summarizeBrowserStorage,
  type ClientStorage,
} from "./session";

function memoryStorage(): ClientStorage {
  const values = new Map<string, string>();
  return {
    get length(): number {
      return values.size;
    },
    getItem(key: string): string | null {
      return values.get(key) ?? null;
    },
    key(index: number): string | null {
      return Array.from(values.keys())[index] ?? null;
    },
    removeItem(key: string): void {
      values.delete(key);
    },
    setItem(key: string, value: string): void {
      values.set(key, value);
    },
  };
}

describe("synthetic shell client state", () => {
  test("most-agents-skip #6: logout clears cache and blocks back-forward shell access", () => {
    cacheSyntheticValue("synthetic-profile", "fixture-only");
    expect(queryCacheSize()).toBe(1);
    clearSensitiveClientState();
    expect(queryCacheSize()).toBe(0);
    expect(canAccessShell(false, "/customer")).toBe(false);
    expect(canAccessShell(false, "/merchant")).toBe(false);
    expect(canAccessShell(false, "/workforce")).toBe(false);
    expect(canAccessShell(false, "/signed-out")).toBe(true);
  });

  test("only actor shell routes are protected", () => {
    expect(isProtectedShell("/customer")).toBe(true);
    expect(isProtectedShell("/merchant")).toBe(true);
    expect(isProtectedShell("/workforce")).toBe(true);
    expect(isProtectedShell("/")).toBe(false);
    expect(isProtectedShell("/signed-out")).toBe(false);
  });

  test("signed-out guard survives reload without storing a credential", () => {
    const local = memoryStorage();
    const session = memoryStorage();

    expect(isSyntheticSignedOut(session)).toBe(false);
    markSyntheticSignedOut(session);
    expect(isSyntheticSignedOut(session)).toBe(true);
    expect(summarizeBrowserStorage(local, session)).toEqual({
      localAtlasEntries: 0,
      sessionAtlasEntries: 1,
      unexpectedAtlasEntries: false,
    });

    clearSyntheticSignedOut(session);
    expect(isSyntheticSignedOut(session)).toBe(false);
    expect(summarizeBrowserStorage(local, session)).toEqual({
      localAtlasEntries: 0,
      sessionAtlasEntries: 0,
      unexpectedAtlasEntries: false,
    });
  });

  test("unknown Atlas-owned browser state fails the storage diagnostic", () => {
    const local = memoryStorage();
    const session = memoryStorage();
    local.setItem("unrelated.example", "not-owned");
    session.setItem("atlas.synthetic.unknown", "fixture-only");

    expect(summarizeBrowserStorage(local, session)).toEqual({
      localAtlasEntries: 0,
      sessionAtlasEntries: 1,
      unexpectedAtlasEntries: true,
    });
  });

  test("navigation restore distinguishes BFCache from safe history reload", () => {
    expect(classifyPageRestore(true, "back_forward")).toBe("bfcache");
    expect(classifyPageRestore(false, "back_forward")).toBe("history-reload");
    expect(classifyPageRestore(false, "reload")).toBe("ordinary");
    expect(classifyPageRestore(false, undefined)).toBe("ordinary");
  });
});
