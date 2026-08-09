async page => {
  const assert = (condition, message) => {
    if (!condition) throw new Error(message);
  };
  await page.waitForURL("**/customer");
  await page.getByRole("heading", {name: "Customer identity"}).waitFor();
  assert(await page.getByText("Financial features are intentionally absent").count() === 1, "customer identity route invented or exposed a financial surface");
  assert((await page.evaluate(() => Object.keys(localStorage))).length === 0, "real OIDC journey wrote Atlas state to localStorage");
  const sessionStorageText = await page.evaluate(() => JSON.stringify(sessionStorage));
  assert(!/(access_token|refresh_token|id_token|authorization_code)/i.test(sessionStorageText), "real OIDC journey wrote a provider token to sessionStorage");
  assert(!/(access_token|refresh_token|id_token|authorization_code)/i.test(await page.evaluate(() => document.cookie)), "real OIDC journey exposed a provider token to JavaScript cookies");

  await page.getByRole("button", {name: "Sign out securely"}).click();
  await page.waitForURL("**/signed-out");
  await page.goBack();
  await page.waitForLoadState("domcontentloaded");
  assert(await page.getByRole("heading", {name: "Customer identity"}).count() === 0, "real OIDC customer content returned after logout and back navigation");
  // The rejected callback contains short-lived synthetic authorization
  // parameters. Leave the diagnostic and screenshot on a parameter-free page.
  await page.goto("http://127.0.0.1:13000/signed-out");
  await page.getByRole("heading", {name: "Client state cleared"}).waitFor();
  return {result: "PASS", provider: "synthetic-keycloak", tokenStorage: "absent", logoutBackNavigation: "denied"};
}
