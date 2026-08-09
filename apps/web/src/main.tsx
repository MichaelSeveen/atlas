import {createRoot, type Root} from "react-dom/client";
import {App, RouteFailure, type RuntimeConfig} from "./product-app";
import {clearSensitiveClientState} from "./session";

async function start(): Promise<void> {
  const response = await fetch("/runtime-config.json", {cache: "no-store"});
  if (!response.ok) {
    throw new Error("runtime configuration unavailable");
  }
  const config = await response.json() as RuntimeConfig;
  if (
    !config.syntheticData ||
    !config.banner.toUpperCase().includes("SYNTHETIC") ||
    !config.apiOrigin.startsWith("http://127.0.0.1:")
  ) {
    throw new Error("unsafe runtime configuration");
  }
  const root = document.getElementById("root");
  if (!root) {
    throw new Error("application root unavailable");
  }
  let applicationRoot: Root | undefined;
  let fallbackScheduled = false;
  const showSafeRouteFailure = () => {
    if (fallbackScheduled) {
      return;
    }
    fallbackScheduled = true;
    clearSensitiveClientState();
    const failedPath = window.location.pathname;
    queueMicrotask(() => {
      applicationRoot?.render(<RouteFailure config={config} path={failedPath} />);
    });
  };

  applicationRoot = createRoot(root, {onUncaughtError: showSafeRouteFailure});
  applicationRoot.render(<App config={config} />);
}

void start();
