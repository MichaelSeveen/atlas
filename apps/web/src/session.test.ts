import {describe, expect, test} from "bun:test";
import {cacheSyntheticValue, canAccessShell, clearSensitiveClientState, isProtectedShell, queryCacheSize} from "./session";

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
});
