import {createBoundedShutdown} from "./server-lifecycle";

const port = Number(Bun.env.ATLAS_WEB_PORT ?? "3000");
const environment = Bun.env.ATLAS_ENVIRONMENT ?? "local";
const banner = Bun.env.ATLAS_ENVIRONMENT_BANNER ?? "LOCAL — SYNTHETIC DATA ONLY";
const mockMode = Bun.env.ATLAS_MOCK_MODE === "true";

if (!Number.isInteger(port) || port < 1024 || port > 65535 || !banner.toUpperCase().includes("SYNTHETIC")) {
  throw new Error("invalid web runtime configuration");
}

const index = Bun.file(new URL("../public/index.html", import.meta.url));
const stylesheet = Bun.file(new URL("../public/styles.css", import.meta.url));
const bundle = Bun.file(new URL("../dist/main.js", import.meta.url));

const commonHeaders = {
  "Cache-Control": "no-store",
  "Content-Security-Policy": "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; font-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'",
  "Cross-Origin-Resource-Policy": "same-origin",
  "Permissions-Policy": "camera=(), microphone=(), geolocation=(), payment=()",
  "Referrer-Policy": "no-referrer",
  "X-Content-Type-Options": "nosniff",
  "X-Frame-Options": "DENY",
};

const server = Bun.serve({
  hostname: "0.0.0.0",
  port,
  maxRequestBodySize: 16 * 1024,
  routes: {
    "/runtime-config.json": () => Response.json({environment, banner, syntheticData: true, mockMode}, {headers: commonHeaders}),
    "/favicon.ico": new Response(null, {status: 204, headers: commonHeaders}),
    "/styles.css": new Response(stylesheet, {headers: {...commonHeaders, "Content-Type": "text/css; charset=utf-8"}}),
    "/main.js": new Response(bundle, {headers: {...commonHeaders, "Content-Type": "text/javascript; charset=utf-8"}}),
  },
  fetch(request) {
    if (request.method !== "GET" && request.method !== "HEAD") {
      return new Response("Method not allowed", {status: 405, headers: {...commonHeaders, Allow: "GET, HEAD"}});
    }
    const {pathname, search} = new URL(request.url);
    if (search !== "" || !["/", "/customer", "/merchant", "/workforce", "/signed-out"].includes(pathname)) {
      return new Response("Not found", {status: 404, headers: commonHeaders});
    }
    return new Response(index, {headers: {...commonHeaders, "Content-Type": "text/html; charset=utf-8"}});
  },
});

const shutdown = createBoundedShutdown(server, 4_000);
let terminationStarted = false;

function handleTermination() {
  if (terminationStarted) {
    return;
  }
  terminationStarted = true;
  void shutdown().then(
    () => process.exit(0),
    () => process.exit(1),
  );
}

process.once("SIGINT", handleTermination);
process.once("SIGTERM", handleTermination);
