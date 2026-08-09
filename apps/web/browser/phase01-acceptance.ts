async page => {
  const origin = "http://127.0.0.1:14173";
  const csrf = "synthetic-browser-csrf";
  const oneTimeSecret = "SYNTHETIC_BROWSER_SECRET_DO_NOT_USE";
  const now = "2026-08-09T10:00:00Z";
  const later = "2026-11-07T10:00:00Z";
  let signedOut = false;
  let createCount = 0;
  let credentials = [];

  const assert = (condition, message) => {
    if (!condition) throw new Error(message);
  };
  const json = (route, body, status = 200, extraHeaders = {}) => route.fulfill({
    status,
    contentType: "application/json",
    headers: {"Cache-Control": "no-store", "X-Atlas-CSRF-Token": csrf, ...extraHeaders},
    body: JSON.stringify(body),
  });

  const context = page.context();
  await context.route("**/v1/**", async route => {
    const request = route.request();
    const path = request.url().replace(origin, "").split("?")[0];
    const method = request.method();

    if (path === "/v1/me" && method === "GET") {
      if (signedOut) return json(route, {
        type: "about:blank", title: "Unauthorized", status: 401, code: "SESSION_INVALID",
        request_id: "req_browser", retryable: false,
      }, 401);
      return json(route, {
        id: "usr_browser_merchant_admin", type: "merchant_user",
        display_name: "Synthetic <img src=x onerror=alert(1)>",
        active_tenant_id: "org_browser_atlas", assurance: "stepped_up",
        permissions: [
          "organization.list", "organization.members.read", "organization.members.roles.update",
          "organization.members.remove", "organization.invitations.create", "approvals.read",
          "api_credentials.read", "api_credentials.create", "api_credentials.rotate", "api_credentials.revoke",
        ],
        authorization_version: 7, session_expires_at: later,
      });
    }
    if (path === "/v1/sessions" && method === "GET") return json(route, {data: [{
      id: "ses_browser_current", population: "merchant", assurance: "stepped_up", current: true,
      client_label: "Synthetic browser", created_at: now, last_seen_at: now,
      idle_expires_at: later, absolute_expires_at: later, revoked_at: null,
    }]});
    if (path === "/v1/organizations" && method === "GET") return json(route, {data: [{
      id: "org_browser_atlas", display_name: "Atlas Synthetic Merchant",
      principal_role: "merchant_security_admin", membership_version: 3,
    }]});
    if (path === "/v1/organizations/org_browser_atlas/members" && method === "GET") return json(route, {data: [{
      id: "mem_browser_admin", organization_id: "org_browser_atlas", principal_id: "usr_browser_admin",
      email_hint: "a***@example.invalid", role: "merchant_admin", status: "active", version: 4,
      created_at: now, revoked_at: null,
    }]});
    if (path === "/v1/approvals" && method === "GET") return json(route, {data: []});
    if (path === "/v1/api-credentials" && method === "GET") return json(route, {data: credentials});
    if (path === "/v1/api-credentials" && method === "POST") {
      createCount++;
      assert(request.headers()["x-atlas-csrf-token"] === csrf, "credential mutation omitted the in-memory CSRF capability");
      assert((request.headers()["idempotency-key"] || "").startsWith("credential-create-"), "credential mutation omitted its caller-owned idempotency key");
      assert(!request.headers().authorization, "browser credential mutation exposed an Authorization token");
      const body = request.postDataJSON();
      assert(body.purpose === "credential_management", "credential mutation omitted its purpose binding");
      const credential = {
        id: "crd_browser_once", organization_id: "org_browser_atlas", name: body.name,
        secret_hint: "…USE", scopes: ["identity:read"], status: "active", expires_at: later,
        overlap_ends_at: null, last_used_at: null, created_at: now, revoked_at: null,
      };
      credentials = [credential];
      return json(route, {credential, secret: oneTimeSecret, secret_disclosed: true}, 201);
    }
    if (path === "/v1/logout" && method === "POST") {
      assert(request.headers()["x-atlas-csrf-token"] === csrf, "logout omitted the in-memory CSRF capability");
      signedOut = true;
      return route.fulfill({status: 204, headers: {"Cache-Control": "no-store"}});
    }
    return json(route, {
      type: "about:blank", title: "Not found", status: 404, code: "NOT_FOUND",
      request_id: "req_browser", retryable: false,
    }, 404);
  });

  await page.goto(`${origin}/merchant`);
  await page.getByRole("heading", {name: "Merchant member"}).waitFor();
  assert(await page.getByText("Synthetic <img src=x onerror=alert(1)>").count() >= 1, "server text was not rendered as inert text");
  assert(await page.locator("img").count() === 0, "server text created executable markup");
  assert(await page.getByRole("button", {name: "Removal unavailable"}).isDisabled(), "administrator removal was not visibly fail-closed");
  assert((await page.evaluate(() => Object.keys(localStorage))).length === 0, "localStorage contains Atlas state");
  assert(!(await page.evaluate(() => JSON.stringify(sessionStorage))).includes("AtlasKey"), "sessionStorage contains a credential");

  await page.getByLabel("Credential name").fill("Browser acceptance key");
  await page.getByRole("button", {name: "Create credential", exact: true}).click();
  const dialog = page.getByRole("dialog", {name: "Store this secret now"});
  await dialog.waitFor();
  assert(await dialog.getByText(oneTimeSecret).count() === 1, "one-time credential secret was not displayed exactly once");
  assert(createCount === 1, "browser duplicated the credential command");
  const storageText = await page.evaluate(() => `${JSON.stringify(localStorage)} ${JSON.stringify(sessionStorage)}`);
  assert(!storageText.includes(oneTimeSecret), "one-time credential secret reached browser storage");
  await dialog.getByLabel("I copied the secret into the approved synthetic client.").check();
  await dialog.getByRole("button", {name: "Close and clear from memory"}).click();
  assert(await page.getByText(oneTimeSecret).count() === 0, "one-time credential secret remained in the DOM after close");

  await page.getByRole("link", {name: "Workforce"}).click();
  await page.getByRole("heading", {name: "This route is isolated"}).waitFor();
  assert(await page.getByText("Staff impersonation is not available.").count() === 1, "population isolation copy is absent");
  await page.getByRole("link", {name: "Merchant"}).click();
  await page.getByRole("heading", {name: "Merchant member"}).waitFor();

  const secondPage = await context.newPage();
  await secondPage.goto(`${origin}/merchant`);
  await secondPage.getByRole("heading", {name: "Merchant member"}).waitFor();
  await page.getByRole("button", {name: "Sign out securely"}).click();
  await page.waitForURL("**/signed-out");
  await secondPage.waitForURL("**/signed-out");
  assert(await page.getByText(oneTimeSecret).count() === 0, "logout page retained the credential secret");
  assert((await page.evaluate(() => Object.keys(localStorage))).length === 0, "logout left localStorage state");

  await page.goBack();
  await page.waitForLoadState("domcontentloaded");
  assert(await page.getByText(oneTimeSecret).count() === 0, "back navigation restored a one-time secret");
  assert(await page.getByText("Synthetic <img src=x onerror=alert(1)>").count() === 0, "back navigation restored privileged content after logout");
  await secondPage.close();

  return {
    result: "PASS",
    scenarios: [
      "generated-contract merchant route", "population isolation", "admin-removal fail-closed",
      "one-time secret memory-only", "idempotent command headers", "cross-tab logout", "post-logout back navigation",
    ],
    createCount,
  };
}
