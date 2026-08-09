import {describe, expect, test} from "bun:test";
import {createAPIClient} from "./api";

describe("Phase 01 browser API boundary", () => {
  test("captures CSRF only in memory and sends credentialed no-store mutations", async () => {
    const calls: Array<{url: string; init?: RequestInit}> = [];
    const fetcher = (async (input: RequestInfo | URL, init?: RequestInit) => {
      calls.push({url: String(input), init});
      if (calls.length === 1) {
        return Response.json({
          id: "usr_test", type: "merchant", display_name: "Synthetic Merchant",
          active_tenant_id: "ten_test", assurance: "baseline", permissions: [],
          authorization_version: 1, session_expires_at: "2026-08-09T01:00:00Z",
        }, {headers: {"X-Atlas-CSRF-Token": "memory-only-csrf"}});
      }
      return Response.json({
        id: "usr_test", type: "merchant", display_name: "Synthetic Merchant",
        active_tenant_id: "ten_next", assurance: "baseline", permissions: [],
        authorization_version: 2, session_expires_at: "2026-08-09T01:00:00Z",
      }, {headers: {"X-Atlas-CSRF-Token": "rotated-memory-only-csrf"}});
    }) as unknown as typeof fetch;
    const client = createAPIClient("http://127.0.0.1:18080/", fetcher);

    await client.current();
    await client.switchOrganization("ten_next");

    expect(calls[0]?.init?.credentials).toBe("include");
    expect(calls[0]?.init?.cache).toBe("no-store");
    const mutationHeaders = new Headers(calls[1]?.init?.headers);
    expect(mutationHeaders.get("X-Atlas-CSRF-Token")).toBe("memory-only-csrf");
    expect(mutationHeaders.get("Authorization")).toBeNull();
    expect(calls[1]?.init?.body).toBe(JSON.stringify({organization_id: "ten_next"}));
  });

  test("response-loss retry can reuse the caller-owned idempotency key", async () => {
    const observedKeys: string[] = [];
    let call = 0;
    const fetcher = (async (_input: RequestInfo | URL, init?: RequestInit) => {
      call += 1;
      if (call === 1) {
        return Response.json({
          id: "usr_test", type: "merchant", display_name: "Synthetic Merchant",
          active_tenant_id: "ten_test", assurance: "stepped_up", permissions: [],
          authorization_version: 1, session_expires_at: null,
        }, {headers: {"X-Atlas-CSRF-Token": "csrf"}});
      }
      observedKeys.push(new Headers(init?.headers).get("Idempotency-Key") || "");
      if (call === 2) throw new TypeError("synthetic connection reset");
      return new Response(null, {status: 204});
    }) as unknown as typeof fetch;
    const client = createAPIClient("http://127.0.0.1:18080", fetcher);
    await client.current();
    const key = "credential-revoke-same-command";

    await expect(client.revokeCredential("key_test", key)).rejects.toMatchObject({
      code: "RESPONSE_LOST",
      responseLost: true,
    });
    await client.revokeCredential("key_test", key);
    expect(observedKeys).toEqual([key, key]);
  });

  test("clears the in-memory CSRF capability after authentication rejection", async () => {
    let call = 0;
    const fetcher = (async () => {
      call += 1;
      if (call === 1) {
        return Response.json({
          id: "usr_test", type: "customer", display_name: "Synthetic Customer",
          active_tenant_id: "ten_test", assurance: "baseline", permissions: [],
          authorization_version: 1, session_expires_at: null,
        }, {headers: {"X-Atlas-CSRF-Token": "csrf"}});
      }
      return Response.json({code: "AUTHENTICATION_REQUIRED", title: "Authentication required"}, {status: 401});
    }) as unknown as typeof fetch;
    const client = createAPIClient("http://127.0.0.1:18080", fetcher);
    await client.current();
    await expect(client.sessions()).rejects.toMatchObject({status: 401});
    await expect(client.logout()).rejects.toMatchObject({code: "CSRF_TOKEN_UNAVAILABLE"});
    expect(call).toBe(2);
  });

  test("login URL contains only the closed population and return route", () => {
    const client = createAPIClient("http://127.0.0.1:18080", (async () => new Response()) as unknown as typeof fetch);
    expect(client.loginURL("workforce", "/workforce")).toBe(
      "http://127.0.0.1:18080/v1/auth/login?population=workforce&return_to=%2Fworkforce",
    );
  });
});
