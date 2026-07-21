import React, {useEffect, useState} from "react";
import {createRoot} from "react-dom/client";
import {cacheSyntheticValue, clearSensitiveClientState, isProtectedShell} from "./session";

type RuntimeConfig = {
  environment: string;
  banner: string;
  syntheticData: boolean;
  mockMode: boolean;
};

type ErrorBoundaryState = {failed: boolean};

class RouteErrorBoundary extends React.Component<React.PropsWithChildren, ErrorBoundaryState> {
  state: ErrorBoundaryState = {failed: false};

  static getDerivedStateFromError(): ErrorBoundaryState {
    return {failed: true};
  }

  render(): React.ReactNode {
    if (this.state.failed) {
      return (
        <section className="panel" role="alert">
          <h2>Shell unavailable</h2>
          <p>The synthetic shell could not render. No command was submitted and no financial state exists.</p>
          <a className="button" href="/">Return to the foundation home</a>
        </section>
      );
    }
    return this.props.children;
  }
}

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

function App({config}: {config: RuntimeConfig}): React.ReactNode {
  const path = usePath();
  const [sessionActive, setSessionActive] = useState(true);

  useEffect(() => {
    const enforceSignedOutState = () => {
      if (!sessionActive && isProtectedShell(window.location.pathname)) {
        window.history.replaceState({}, "", "/signed-out");
        window.dispatchEvent(new PopStateEvent("popstate"));
      }
    };
    window.addEventListener("pageshow", enforceSignedOutState);
    window.addEventListener("popstate", enforceSignedOutState);
    enforceSignedOutState();
    return () => {
      window.removeEventListener("pageshow", enforceSignedOutState);
      window.removeEventListener("popstate", enforceSignedOutState);
    };
  }, [sessionActive]);

  const logout = () => {
    clearSensitiveClientState();
    setSessionActive(false);
    navigate("/signed-out");
  };

  const shell = shellLabels[path];
  if (shell && sessionActive) {
    cacheSyntheticValue("active-shell", shell);
  }

  return (
    <>
      <div className="environment-banner" role="status" data-testid="environment-banner">
        {config.banner}{config.mockMode ? " — MOCK MODE" : ""}
      </div>
      <main className="shell-frame">
        <header className="masthead">
          <div>
            <div className="phase-label">Phase 00 foundation</div>
            <h1 className="brand">Atlas</h1>
          </div>
          <div className="status-row"><span className="status-icon" aria-hidden="true" />No product capability</div>
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
        <RouteErrorBoundary key={path}>
          {path === "/" && <Overview />}
          {shell && sessionActive && <ActorShell label={shell} onLogout={logout} />}
          {path === "/signed-out" && <SignedOut />}
          {path !== "/" && !shell && path !== "/signed-out" && <UnknownRoute />}
        </RouteErrorBoundary>
      </main>
    </>
  );
}

function Overview(): React.ReactNode {
  return (
    <section className="panel">
      <h2>Environment shell verification</h2>
      <p>These routes prove separation, accessibility, synthetic labeling, and safe client-state handling before product work begins.</p>
      <div className="safe-note">No wallet, balance, payment, identity session, or financial command is implemented.</div>
    </section>
  );
}

function ActorShell({label, onLogout}: {label: string; onLogout: () => void}): React.ReactNode {
  return (
    <section className="panel" data-testid="actor-shell">
      <h2>{label}</h2>
      <p>This is an isolated route shell with synthetic fixture labels only.</p>
      <div className="safe-note">No authentication token or sensitive value is stored in localStorage or sessionStorage.</div>
      <div className="actions">
        <button className="button danger" type="button" onClick={onLogout}>Clear client state and sign out</button>
      </div>
    </section>
  );
}

function SignedOut(): React.ReactNode {
  return (
    <section className="panel" data-testid="signed-out">
      <h2>Client state cleared</h2>
      <p>Protected synthetic shells stay unavailable through browser back/forward navigation until this page is reloaded.</p>
    </section>
  );
}

function UnknownRoute(): React.ReactNode {
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
  createRoot(root).render(<App config={config} />);
}

void start();
