import {useEffect, useState, type ReactNode} from "react";
import {createRoot, type Root} from "react-dom/client";
import {
  cacheSyntheticValue,
  classifyPageRestore,
  clearSensitiveClientState,
  clearSyntheticSignedOut,
  isProtectedShell,
  isSyntheticSignedOut,
  markSyntheticSignedOut,
  summarizeBrowserStorage,
  type PageRestoreKind,
} from "./session";

type RuntimeConfig = {
  environment: string;
  banner: string;
  syntheticData: boolean;
  mockMode: boolean;
};

const shellLabels: Record<string, string> = {
  "/customer": "Customer shell",
  "/merchant": "Merchant shell",
  "/workforce": "Workforce shell",
};

function usePath(): string {
  const [path, setPath] = useState(window.location.pathname);
  useEffect(() => {
    const update = () => setPath(window.location.pathname);
    window.addEventListener("popstate", update);
    return () => window.removeEventListener("popstate", update);
  }, []);
  return path;
}

function navigate(path: string): void {
  window.history.pushState({}, "", path);
  window.dispatchEvent(new PopStateEvent("popstate"));
}

function App({config}: {config: RuntimeConfig}): ReactNode {
  const path = usePath();
  const [sessionActive, setSessionActive] = useState(
    () => !isSyntheticSignedOut(window.sessionStorage),
  );
  const [pageRestore, setPageRestore] = useState<PageRestoreKind>(
    () => classifyPageRestore(false, currentNavigationType()),
  );
  const shell = shellLabels[path];

  useEffect(() => {
    const enforceSignedOutState = () => {
      if (!sessionActive && isProtectedShell(window.location.pathname)) {
        window.history.replaceState({}, "", "/signed-out");
        window.dispatchEvent(new PopStateEvent("popstate"));
      }
    };
    const handlePageShow = (event: PageTransitionEvent) => {
      setPageRestore(classifyPageRestore(event.persisted, currentNavigationType()));
      enforceSignedOutState();
    };
    window.addEventListener("pageshow", handlePageShow);
    window.addEventListener("popstate", enforceSignedOutState);
    enforceSignedOutState();
    return () => {
      window.removeEventListener("pageshow", handlePageShow);
      window.removeEventListener("popstate", enforceSignedOutState);
    };
  }, [sessionActive]);

  useEffect(() => {
    if (shell && sessionActive) {
      cacheSyntheticValue("active-shell", shell);
    }
  }, [sessionActive, shell]);

  const logout = () => {
    clearSensitiveClientState();
    try {
      markSyntheticSignedOut(window.sessionStorage);
    } catch {
      // The in-memory state still fails closed for this document.
    }
    setSessionActive(false);
    navigate("/signed-out");
  };

  const resetSyntheticShell = () => {
    clearSensitiveClientState();
    try {
      clearSyntheticSignedOut(window.sessionStorage);
    } catch {
      // Storage denial leaves the next protected route fail-closed.
    }
    setSessionActive(true);
    navigate("/");
  };

  return (
    <>
      <div className="environment-banner" role="status" data-testid="environment-banner">
        {config.banner}{config.mockMode ? " — MOCK MODE" : ""}
      </div>
      <main className="shell-frame">
        <header className="masthead">
          <div>
            <div className="phase-label">Phase 01 foundation</div>
            <h1 className="brand">Atlas</h1>
          </div>
          <div className="status-row"><span className="status-icon" aria-hidden="true" />Synthetic shell only</div>
        </header>
        <nav className="route-nav" aria-label="Foundation shells">
          {["/", "/customer", "/merchant", "/workforce"].map((route) => (
            <a
              key={route}
              href={route}
              aria-current={path === route ? "page" : undefined}
              onClick={(event) => { event.preventDefault(); navigate(route); }}
            >
              {route === "/" ? "Overview" : shellLabels[route]}
            </a>
          ))}
        </nav>
        {path === "/" && <Overview />}
        {shell && sessionActive && <ActorShell label={shell} onLogout={logout} />}
        {path === "/signed-out" && <SignedOut onReset={resetSyntheticShell} pageRestore={pageRestore} />}
        {path !== "/" && !shell && path !== "/signed-out" && <UnknownRoute />}
      </main>
    </>
  );
}

function RouteFailure({config, path}: {config: RuntimeConfig; path: string}): ReactNode {
  const route = shellLabels[path] ?? "Requested shell";
  return (
    <>
      <div className="environment-banner" role="status" data-testid="environment-banner">
        {config.banner}{config.mockMode ? " — MOCK MODE" : ""}
      </div>
      <main className="shell-frame">
        <section className="panel" role="alert" data-testid="route-failure">
          <h2>Shell unavailable</h2>
          <p>{route} could not render. No command was submitted and no financial state exists.</p>
          <a className="button" href="/">Return to the foundation home</a>
        </section>
      </main>
    </>
  );
}

function Overview(): ReactNode {
  return (
    <section className="panel">
      <h2>Environment shell verification</h2>
      <p>These routes prove separation, accessibility, synthetic labeling, and safe client-state handling while remaining detached from product APIs.</p>
      <div className="safe-note">No wallet, balance, payment, or financial command is implemented. This shell does not consume the Phase 01 identity API.</div>
    </section>
  );
}

function ActorShell({label, onLogout}: {label: string; onLogout: () => void}): ReactNode {
  return (
    <section className="panel" data-testid="actor-shell">
      <h2>{label}</h2>
      <p>This is an isolated route shell with synthetic fixture labels only.</p>
      <div className="safe-note">No authentication token or sensitive value is stored in localStorage or sessionStorage.</div>
      <BrowserStorageState />
      <div className="actions">
        <button className="button danger" type="button" onClick={onLogout}>Clear client state and sign out</button>
      </div>
    </section>
  );
}

function SignedOut({
  onReset,
  pageRestore,
}: {
  onReset: () => void;
  pageRestore: PageRestoreKind;
}): ReactNode {
  return (
    <section className="panel" data-testid="signed-out">
      <h2>Client state cleared</h2>
      <p>Protected synthetic shells stay unavailable through browser back/forward navigation and reload until a new synthetic shell session is explicitly started.</p>
      <div className="safe-note" data-testid="page-restore" data-page-restore={pageRestore}>
        Navigation restore: {pageRestore}
      </div>
      <BrowserStorageState />
      <div className="actions">
        <button className="button" type="button" onClick={onReset}>Start a new synthetic shell session</button>
      </div>
    </section>
  );
}

function BrowserStorageState(): ReactNode {
  const summary = summarizeBrowserStorage(window.localStorage, window.sessionStorage);
  return (
    <div
      className="safe-note"
      data-testid="browser-storage-state"
      data-local-atlas-entries={summary.localAtlasEntries}
      data-session-atlas-entries={summary.sessionAtlasEntries}
      data-unexpected-atlas-entries={summary.unexpectedAtlasEntries}
    >
      Browser credential storage: {summary.unexpectedAtlasEntries ? "unexpected Atlas state detected" : "absent"}
    </div>
  );
}

function currentNavigationType(): string | undefined {
  const [navigation] = window.performance.getEntriesByType("navigation");
  if (!navigation || !("type" in navigation)) {
    return undefined;
  }
  return String(navigation.type);
}

function UnknownRoute(): ReactNode {
  return (
    <section className="panel" role="alert">
      <h2>Route unavailable</h2>
      <p>This route is not part of the Phase 00 shell inventory.</p>
    </section>
  );
}

async function start(): Promise<void> {
  const response = await fetch("/runtime-config.json", {cache: "no-store"});
  if (!response.ok) {
    throw new Error("runtime configuration unavailable");
  }
  const config = await response.json() as RuntimeConfig;
  if (!config.syntheticData || !config.banner.toUpperCase().includes("SYNTHETIC")) {
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

  applicationRoot = createRoot(root, {onUncaughtError: showSafeRouteFailure });
  applicationRoot.render(<App config={config} />);
}

void start();
