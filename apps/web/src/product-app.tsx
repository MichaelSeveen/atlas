import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type Dispatch,
  type FormEvent,
  type ReactNode,
  type SetStateAction,
} from "react";
import {
  createAPIClient,
  isAPIError,
  newIdempotencyKey,
  type APIClient,
  type Approval,
  type Credential,
  type CredentialSecret,
  type Organization,
  type OrganizationMember,
  type Principal,
  type Session,
} from "./api";
import {
  cacheSyntheticValue,
  classifyPageRestore,
  clearSensitiveClientState,
  clearSyntheticSignedOut,
  createSecurityBoundaryChannel,
  isProtectedShell,
  markSyntheticSignedOut,
  summarizeBrowserStorage,
  type PageRestoreKind,
  type SecurityBoundary,
  type SecurityBoundaryChannel,
} from "./session";

export type RuntimeConfig = {
  environment: string;
  banner: string;
  syntheticData: boolean;
  mockMode: boolean;
  apiOrigin: string;
};

type AuthState =
  | {status: "loading"}
  | {status: "signed-out"}
  | {status: "failed"}
  | {status: "active"; principal: Principal};

type Notice = {
  kind: "pending" | "success" | "denied" | "stale" | "failed" | "ambiguous" | "step-up";
  message: string;
  retry?: () => void;
  verify?: () => void;
};

type NoticeSetter = Dispatch<SetStateAction<Notice | undefined>>;

const routePopulation: Record<string, "customer" | "merchant" | "workforce"> = {
  "/customer": "customer",
  "/merchant": "merchant",
  "/workforce": "workforce",
};

export function App({config}: {config: RuntimeConfig}): ReactNode {
  const path = usePath();
  const client = useMemo(() => createAPIClient(config.apiOrigin), [config.apiOrigin]);
  const [auth, setAuth] = useState<AuthState>({status: "loading"});
  const [pageRestore, setPageRestore] = useState<PageRestoreKind>(
    () => classifyPageRestore(false, currentNavigationType()),
  );
  const channel = useRef<SecurityBoundaryChannel | undefined>(undefined);

  const refreshPrincipal = useCallback(async () => {
    try {
      const principal = await client.current();
      try {
        clearSyntheticSignedOut(window.sessionStorage);
      } catch {
        // The server session remains authoritative when storage is denied.
      }
      cacheSyntheticValue("principal-view", String(principal.authorization_version ?? 0));
      setAuth({status: "active", principal});
    } catch (error) {
      clearSensitiveClientState();
      if (isAPIError(error) && error.status === 401) {
        setAuth({status: "signed-out"});
        return;
      }
      setAuth({status: "failed"});
    }
  }, [client]);

  const enforceBoundary = useCallback((boundary: SecurityBoundary) => {
    clearSensitiveClientState();
    client.clearSecurityState();
    if (boundary === "logout" || boundary === "session-invalid") {
      try {
        markSyntheticSignedOut(window.sessionStorage);
      } catch {
        // In-memory signed-out state still fails closed for this document.
      }
      setAuth({status: "signed-out"});
      if (isProtectedShell(window.location.pathname)) {
        navigate("/signed-out", true);
      }
      return;
    }
    setAuth({status: "loading"});
    void refreshPrincipal();
  }, [client, refreshPrincipal]);

  useEffect(() => {
    void refreshPrincipal();
  }, [refreshPrincipal]);

  useEffect(() => {
    channel.current = createSecurityBoundaryChannel(enforceBoundary);
    return () => channel.current?.close();
  }, [enforceBoundary]);

  useEffect(() => {
    const handlePageShow = (event: PageTransitionEvent) => {
      const kind = classifyPageRestore(event.persisted, currentNavigationType());
      setPageRestore(kind);
      if (kind !== "ordinary") {
        clearSensitiveClientState();
        client.clearSecurityState();
        setAuth({status: "loading"});
        void refreshPrincipal();
      }
    };
    window.addEventListener("pageshow", handlePageShow);
    return () => window.removeEventListener("pageshow", handlePageShow);
  }, [client, refreshPrincipal]);

  const logout = useCallback(async () => {
    try {
      if (auth.status === "active") {
        await client.logout();
      }
    } catch {
      // A response-loss logout is locally treated as signed out; the next login rotates the session.
    } finally {
      channel.current?.publish("logout");
      enforceBoundary("logout");
    }
  }, [auth.status, client, enforceBoundary]);

  const securityBoundary = useCallback((boundary: SecurityBoundary, principal?: Principal) => {
    clearSensitiveClientState();
    channel.current?.publish(boundary);
    if (principal) {
      setAuth({status: "active", principal});
      return;
    }
    setAuth({status: "loading"});
    void refreshPrincipal();
  }, [refreshPrincipal]);

  return (
    <Shell config={config} path={path} auth={auth} onLogout={logout}>
      {path === "/" && <Overview config={config} />}
      {path === "/signed-out" && <SignedOut config={config} pageRestore={pageRestore} />}
      {routePopulation[path] && (
        <ProtectedRoute
          path={path}
          expectedPopulation={routePopulation[path]}
          auth={auth}
          client={client}
          onSecurityBoundary={securityBoundary}
        />
      )}
      {path !== "/" && path !== "/signed-out" && !routePopulation[path] && <UnknownRoute />}
    </Shell>
  );
}

function Shell({
  config,
  path,
  auth,
  onLogout,
  children,
}: {
  config: RuntimeConfig;
  path: string;
  auth: AuthState;
  onLogout: () => void;
  children: ReactNode;
}): ReactNode {
  return (
    <>
      <div className="environment-banner" role="status" data-testid="environment-banner">
        {config.banner}{config.mockMode ? " — MOCK MODE" : ""}
      </div>
      <main className={`shell-frame ${path === "/workforce" ? "workforce-frame" : ""}`}>
        <header className="masthead">
          <div>
            <div className="phase-label">Phase 01 · identity control plane</div>
            <h1 className="brand">Atlas</h1>
          </div>
          <div className="status-stack">
            <div className="status-row">
              <span className={`status-icon ${auth.status}`} aria-hidden="true" />
              {auth.status === "active" ? `${auth.principal.display_name} · ${auth.principal.assurance}` : authLabel(auth.status)}
            </div>
            {auth.status === "active" && (
              <button className="text-button" type="button" onClick={onLogout}>Sign out securely</button>
            )}
          </div>
        </header>
        <nav className="route-nav" aria-label="Atlas identity surfaces">
          {[
            ["/", "Overview"],
            ["/customer", "Customer"],
            ["/merchant", "Merchant"],
            ["/workforce", "Workforce"],
          ].map(([route, label]) => (
            <a
              key={route}
              href={route}
              aria-current={path === route ? "page" : undefined}
              onClick={(event) => { event.preventDefault(); navigate(route); }}
            >
              {label}
            </a>
          ))}
        </nav>
        {children}
      </main>
    </>
  );
}

function ProtectedRoute({
  path,
  expectedPopulation,
  auth,
  client,
  onSecurityBoundary,
}: {
  path: string;
  expectedPopulation: "customer" | "merchant" | "workforce";
  auth: AuthState;
  client: APIClient;
  onSecurityBoundary: (boundary: SecurityBoundary, principal?: Principal) => void;
}): ReactNode {
  if (auth.status === "loading") {
    return <StatePanel title="Checking server session" kind="loading">No cached permission is being trusted.</StatePanel>;
  }
  if (auth.status === "failed") {
    return <StatePanel title="Identity service unavailable" kind="failed">No operation is enabled. Retry after readiness is restored.</StatePanel>;
  }
  if (auth.status === "signed-out") {
    return <SignIn client={client} population={expectedPopulation} returnTo={path} />;
  }
  const expectedPrincipalType = expectedPopulation === "merchant" ? "merchant_user" : expectedPopulation;
  if (auth.principal.type !== expectedPrincipalType) {
    return <StatePanel title="This route is isolated" kind="denied">Sign out before entering a different identity population. Staff impersonation is not available.</StatePanel>;
  }
  if (expectedPopulation === "customer") {
    return <CustomerConsole principal={auth.principal} client={client} onSecurityBoundary={onSecurityBoundary} />;
  }
  if (expectedPopulation === "merchant") {
    return <MerchantConsole principal={auth.principal} client={client} onSecurityBoundary={onSecurityBoundary} />;
  }
  return <WorkforceConsole principal={auth.principal} client={client} onSecurityBoundary={onSecurityBoundary} />;
}

function Overview({config}: {config: RuntimeConfig}): ReactNode {
  return (
    <section className="panel hero-panel">
      <div>
        <div className="eyebrow">Server-authoritative access</div>
        <h2>Identity operations without browser tokens</h2>
        <p>Choose an isolated population route. Atlas keeps the OIDC exchange and application session behind the BFF, rechecks PostgreSQL authorization on every protected request, and stores no access token in browser storage.</p>
      </div>
      <dl className="fact-grid">
        <div><dt>Environment</dt><dd>{config.environment}</dd></div>
        <div><dt>Data</dt><dd>Synthetic only</dd></div>
        <div><dt>Financial state</dt><dd>Absent</dd></div>
        <div><dt>API authority</dt><dd>PostgreSQL</dd></div>
      </dl>
    </section>
  );
}

function SignIn({client, population, returnTo}: {client: APIClient; population: "customer" | "merchant" | "workforce"; returnTo: string}): ReactNode {
  return (
    <section className={`panel sign-in-panel ${population === "workforce" ? "workforce-panel" : ""}`} data-testid="sign-in">
      <div className="eyebrow">{population} identity</div>
      <h2>Sign in through the synthetic identity provider</h2>
      <p>Atlas will rotate the application session at callback. The browser receives only an opaque, HttpOnly session cookie.</p>
      {population === "workforce" && <div className="warning-note">Workforce identity uses a separate realm, shorter session, and no customer impersonation.</div>}
      <a className="button primary" href={client.loginURL(population, returnTo)}>Continue to secure sign-in</a>
    </section>
  );
}

function CustomerConsole({principal, client, onSecurityBoundary}: ConsoleProps): ReactNode {
  return (
    <div className="console-grid">
      <PrincipalCard principal={principal} title="Customer identity" />
      <SessionCenter client={client} onSecurityBoundary={onSecurityBoundary} />
      <section className="panel span-two">
        <h2>Financial features are intentionally absent</h2>
        <p>Phase 01 proves identity and session boundaries only. No wallet, balance, beneficiary, payout, or other money-moving command exists on this route.</p>
      </section>
    </div>
  );
}

function MerchantConsole({principal, client, onSecurityBoundary}: ConsoleProps): ReactNode {
  return (
    <div className="console-grid">
      <PrincipalCard principal={principal} title="Merchant member" />
      <SessionCenter client={client} onSecurityBoundary={onSecurityBoundary} />
      <OrganizationCenter principal={principal} client={client} onSecurityBoundary={onSecurityBoundary} />
      <ApprovalCenter principal={principal} client={client} />
      <CredentialCenter principal={principal} client={client} />
    </div>
  );
}

function WorkforceConsole({principal, client, onSecurityBoundary}: ConsoleProps): ReactNode {
  return (
    <div className="console-grid workforce-console">
      <section className="panel span-two workforce-identity">
        <div className="eyebrow">Privileged workforce boundary</div>
        <h2>{principal.display_name}</h2>
        <p>Separate identity realm · {principal.assurance} assurance · authorization version {principal.authorization_version}</p>
        <div className="warning-note">No staff impersonation. Customer-context access is limited to an audited, masked server projection when an owning product phase supplies that resource.</div>
      </section>
      <SessionCenter client={client} onSecurityBoundary={onSecurityBoundary} />
      <ApprovalCenter principal={principal} client={client} />
      <section className="panel">
        <h2>Current authority</h2>
        <PermissionList permissions={principal.permissions} />
      </section>
    </div>
  );
}

type ConsoleProps = {
  principal: Principal;
  client: APIClient;
  onSecurityBoundary: (boundary: SecurityBoundary, principal?: Principal) => void;
};

function PrincipalCard({principal, title}: {principal: Principal; title: string}): ReactNode {
  return (
    <section className="panel">
      <div className="eyebrow">Current server context</div>
      <h2>{title}</h2>
      <dl className="fact-list">
        <div><dt>Principal</dt><dd>{principal.display_name}</dd></div>
        <div><dt>Tenant</dt><dd>{principal.active_tenant_id ?? "No active tenant"}</dd></div>
        <div><dt>Assurance</dt><dd>{principal.assurance}</dd></div>
        <div><dt>Authorization version</dt><dd>{principal.authorization_version}</dd></div>
      </dl>
    </section>
  );
}

function SessionCenter({client, onSecurityBoundary}: {client: APIClient; onSecurityBoundary: ConsoleProps["onSecurityBoundary"]}): ReactNode {
  const [sessions, setSessions] = useState<Session[] | undefined>();
  const [loadFailed, setLoadFailed] = useState(false);
  const [notice, setNotice] = useState<Notice | undefined>();

  const load = useCallback(async () => {
    setLoadFailed(false);
    try {
      setSessions(await client.sessions());
    } catch (error) {
      if (isAPIError(error) && error.status === 401) {
        onSecurityBoundary("session-invalid");
        return;
      }
      setLoadFailed(true);
    }
  }, [client, onSecurityBoundary]);

  useEffect(() => { void load(); }, [load]);

  const revoke = (session: Session) => {
    void runMutation(
      `Revoke ${session.current ? "this" : "selected"} session`,
      () => client.revokeSession(session.id),
      setNotice,
      async () => {
        if (session.current) {
          onSecurityBoundary("session-invalid");
        } else {
          await load();
        }
      },
    );
  };

  const revokeOthers = () => {
    const key = newIdempotencyKey("revoke-others");
    void runMutation("Revoke all other sessions", () => client.revokeAll(false, key), setNotice, load);
  };

  return (
    <section className="panel">
      <PanelHeading title="Sessions and devices" onRefresh={load} />
      <NoticeView notice={notice} />
      {loadFailed && <InlineState kind="failed">Sessions could not be loaded. No revocation control is assumed.</InlineState>}
      {!sessions && !loadFailed && <InlineState kind="loading">Loading from the server…</InlineState>}
      {sessions?.length === 0 && <InlineState kind="empty">No active sessions.</InlineState>}
      <div className="item-list">
        {sessions?.map((session) => (
          <article className="item" key={session.id}>
            <div><strong>{session.current ? "This device" : session.client_label || "Unlabelled device"}</strong><small>{session.assurance} · last seen {formatTime(session.last_seen_at)}</small></div>
            <button className="button compact danger" type="button" onClick={() => revoke(session)}>Revoke</button>
          </article>
        ))}
      </div>
      <button className="button compact" type="button" onClick={revokeOthers}>Log out other sessions</button>
    </section>
  );
}

function OrganizationCenter({principal, client, onSecurityBoundary}: ConsoleProps): ReactNode {
  const [organizations, setOrganizations] = useState<Organization[] | undefined>();
  const [members, setMembers] = useState<OrganizationMember[] | undefined>();
  const [notice, setNotice] = useState<Notice | undefined>();
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("merchant_viewer");
  const activeTenant = principal.active_tenant_id;

  const load = useCallback(async () => {
    try {
      const listedOrganizations = await client.organizations();
      setOrganizations(listedOrganizations);
      if (activeTenant) {
        setMembers(await client.members(activeTenant));
      } else {
        setMembers([]);
      }
    } catch (error) {
      setNotice(noticeFromError("Load organization context", error));
    }
  }, [activeTenant, client]);

  useEffect(() => { void load(); }, [load]);

  const switchTenant = (organizationID: string) => {
    void runMutation("Switch active organization", () => client.switchOrganization(organizationID), setNotice, async (updated) => {
      onSecurityBoundary("tenant-switch", updated);
    });
  };

  const invite = (event: FormEvent) => {
    event.preventDefault();
    if (!activeTenant) return;
    const key = newIdempotencyKey("invite");
    void runMutation(
      "Create invitation",
      () => client.invite(activeTenant, inviteEmail, inviteRole, key),
      setNotice,
      async (result) => {
        setInviteEmail("");
        setNotice({
          kind: "success",
          message: result.secret_disclosed
            ? "Invitation created. Its one-time acceptance token was returned only in this response; deliver it through the approved synthetic channel."
            : "Invitation replayed without disclosing the acceptance token.",
        });
        await load();
      },
      "identity.organization.invitation.create_admin",
      client,
    );
  };

  return (
    <section className="panel span-two">
      <PanelHeading title="Organization and membership" onRefresh={load} />
      <NoticeView notice={notice} />
      <div className="organization-switcher" aria-label="Active organization switcher">
        {organizations?.map((organization) => (
          <button
            className={`tenant-card ${organization.id === activeTenant ? "active" : ""}`}
            key={organization.id}
            type="button"
            aria-pressed={organization.id === activeTenant}
            onClick={() => switchTenant(organization.id)}
          >
            <strong>{organization.display_name}</strong>
            <small>{organization.principal_role}</small>
          </button>
        ))}
      </div>
      {!organizations && <InlineState kind="loading">Loading organizations from server truth…</InlineState>}
      {organizations?.length === 0 && <InlineState kind="empty">No organization membership is active.</InlineState>}
      {activeTenant && (
        <>
          <h3>Members</h3>
          <div className="item-list">
            {members?.map((member) => (
              <MemberRow
                key={member.id}
                member={member}
                principal={principal}
                onRole={(role) => {
                  const key = newIdempotencyKey("member-role");
                  void runMutation(
                    "Update member role",
                    () => client.updateMemberRole(activeTenant, member, role, key),
                    setNotice,
                    async () => { onSecurityBoundary("membership-change"); },
                    role === "merchant_admin" || member.role === "merchant_admin"
                      ? "identity.organization.membership.change_admin"
                      : undefined,
                    client,
                  );
                }}
                onRemove={() => {
                  const key = newIdempotencyKey("member-remove");
                  void runMutation(
                    "Remove member",
                    () => client.removeMember(activeTenant, member, key),
                    setNotice,
                    async () => { onSecurityBoundary("membership-change"); },
                    member.role === "merchant_admin" ? "identity.organization.membership.remove_admin" : undefined,
                    client,
                  );
                }}
              />
            ))}
          </div>
          {members?.length === 0 && <InlineState kind="empty">No authorized members are visible.</InlineState>}
          {hasPermission(principal, "organization.invitations.create") && (
            <form className="form-grid" onSubmit={invite}>
              <label>Email<input required type="email" value={inviteEmail} onChange={(event) => setInviteEmail(event.target.value)} /></label>
              <label>Delegated role<select value={inviteRole} onChange={(event) => setInviteRole(event.target.value)}><option value="merchant_viewer">Viewer</option><option value="merchant_operator">Operator</option></select></label>
              <button className="button primary" type="submit">Create single-use invitation</button>
              <p className="form-help">Atlas checks the inviter’s current delegation authority again at commit.</p>
            </form>
          )}
        </>
      )}
    </section>
  );
}

function MemberRow({
  member,
  principal,
  onRole,
  onRemove,
}: {
  member: OrganizationMember;
  principal: Principal;
  onRole: (role: OrganizationMember["role"]) => void;
  onRemove: () => void;
}): ReactNode {
  const [role, setRole] = useState(member.role);
  return (
    <article className="item member-item">
      <div>
        <strong>{member.email_hint || "Masked member"}</strong>
        <small>{member.role} · version {member.version} · {member.status}</small>
      </div>
      {hasPermission(principal, "organization.members.roles.update") && (
        <div className="inline-actions">
          <select
            aria-label={`Role for ${member.id}`}
            value={role}
            onChange={(event) => setRole(event.target.value as OrganizationMember["role"])}
          >
            <option value="merchant_viewer">Viewer</option>
            <option value="merchant_operator">Operator</option>
            <option value="merchant_admin">Administrator (approval)</option>
            <option value="merchant_security_admin">Security administrator (approval)</option>
          </select>
          <button className="button compact" type="button" disabled={role === member.role} onClick={() => onRole(role)}>Apply</button>
          {hasPermission(principal, "organization.members.remove") && member.role !== "merchant_admin" && (
            <button className="button compact danger" type="button" onClick={onRemove}>Remove</button>
          )}
          {hasPermission(principal, "organization.members.remove") && member.role === "merchant_admin" && (
            <button className="button compact" type="button" disabled title="Fresh step-up and last-administrator policy are not yet ratified">Removal unavailable</button>
          )}
        </div>
      )}
    </article>
  );
}

function CredentialCenter({principal, client}: {principal: Principal; client: APIClient}): ReactNode {
  const [credentials, setCredentials] = useState<Credential[] | undefined>();
  const [notice, setNotice] = useState<Notice | undefined>();
  const [name, setName] = useState("");
  const [oneTime, setOneTime] = useState<CredentialSecret | undefined>();
  const [copyConfirmed, setCopyConfirmed] = useState(false);

  const load = useCallback(async () => {
    try {
      setCredentials(await client.credentials());
    } catch (error) {
      setNotice(noticeFromError("Load API credentials", error));
    }
  }, [client]);
  useEffect(() => { if (hasPermission(principal, "api_credentials.read")) void load(); }, [load, principal]);

  const exposeSecret = async (result: CredentialSecret) => {
    setOneTime(result);
    setCopyConfirmed(false);
    await load();
  };

  const create = (event: FormEvent) => {
    event.preventDefault();
    const key = newIdempotencyKey("credential-create");
    void runMutation(
      "Create API credential",
      () => client.createCredential(name, 90, key),
      setNotice,
      async (result) => { setName(""); await exposeSecret(result); },
      "identity.api_credential.create",
      client,
    );
  };

  if (!hasPermission(principal, "api_credentials.read")) {
    return <section className="panel"><h2>API credentials</h2><InlineState kind="denied">This current server principal cannot read credential metadata.</InlineState></section>;
  }

  return (
    <section className="panel span-two">
      <PanelHeading title="Merchant API credentials" onRefresh={load} />
      <NoticeView notice={notice} />
      <div className="item-list">
        {credentials?.map((credential) => (
          <article className="item" key={credential.id}>
            <div><strong>{credential.name}</strong><small>{credential.status} · {credential.secret_hint} · expires {formatTime(credential.expires_at)}</small></div>
            <div className="inline-actions">
              {hasPermission(principal, "api_credentials.rotate") && credential.status !== "revoked" && (
                <button className="button compact" type="button" onClick={() => {
                  const key = newIdempotencyKey("credential-rotate");
                  void runMutation(
                    "Rotate API credential",
                    () => client.rotateCredential(credential.id, key),
                    setNotice,
                    exposeSecret,
                    "identity.api_credential.rotate",
                    client,
                  );
                }}>Rotate</button>
              )}
              {hasPermission(principal, "api_credentials.revoke") && credential.status !== "revoked" && (
                <button className="button compact danger" type="button" onClick={() => {
                  const key = newIdempotencyKey("credential-revoke");
                  void runMutation(
                    "Revoke API credential",
                    () => client.revokeCredential(credential.id, key),
                    setNotice,
                    load,
                    "identity.api_credential.revoke",
                    client,
                  );
                }}>Revoke</button>
              )}
            </div>
          </article>
        ))}
      </div>
      {credentials?.length === 0 && <InlineState kind="empty">No credential metadata exists for this organization.</InlineState>}
      {hasPermission(principal, "api_credentials.create") && (
        <form className="form-grid" onSubmit={create}>
          <label>Credential name<input required minLength={3} maxLength={100} value={name} onChange={(event) => setName(event.target.value)} /></label>
          <div><span className="field-label">Scope</span><div className="readonly-field">identity:read · local environment</div></div>
          <button className="button primary" type="submit">Create credential</button>
          <p className="form-help">Fresh action-bound verification is required. The 256-bit secret is shown only once.</p>
        </form>
      )}
      {oneTime && (
        <div className="secret-panel" role="dialog" aria-modal="true" aria-labelledby="one-time-secret-title">
          <div className="eyebrow">One-time disclosure</div>
          <h3 id="one-time-secret-title">Store this secret now</h3>
          {oneTime.secret_disclosed && oneTime.secret ? <output className="secret-value">{oneTime.secret}</output> : <InlineState kind="stale">This was an idempotent replay. Atlas will not disclose the secret again; rotate to replace it.</InlineState>}
          <label className="confirmation"><input type="checkbox" checked={copyConfirmed} onChange={(event) => setCopyConfirmed(event.target.checked)} /> I copied the secret into the approved synthetic client.</label>
          <button className="button" type="button" disabled={!copyConfirmed && oneTime.secret_disclosed} onClick={() => { setOneTime(undefined); setCopyConfirmed(false); }}>Close and clear from memory</button>
        </div>
      )}
    </section>
  );
}

function ApprovalCenter({principal, client}: {principal: Principal; client: APIClient}): ReactNode {
  const [approvals, setApprovals] = useState<Approval[] | undefined>();
  const [notice, setNotice] = useState<Notice | undefined>();
  const load = useCallback(async () => {
    try {
      setApprovals(await client.approvals());
    } catch (error) {
      setNotice(noticeFromError("Load approvals", error));
    }
  }, [client]);
  useEffect(() => { if (hasPermission(principal, "approvals.read")) void load(); }, [load, principal]);

  if (!hasPermission(principal, "approvals.read")) {
    return <section className="panel"><h2>Approvals</h2><InlineState kind="denied">Approval metadata is not available to this principal.</InlineState></section>;
  }

  return (
    <section className="panel">
      <PanelHeading title="Maker-checker approvals" onRefresh={load} />
      <NoticeView notice={notice} />
      <div className="item-list">
        {approvals?.map((approval) => (
          <article className="approval-card" key={approval.id}>
            <div className="item"><div><strong>{approval.status}</strong><small>{approval.action_type} · v{approval.version}</small></div></div>
            <code className="digest">{approval.payload_digest}</code>
            <div className="inline-actions">
              {approval.status === "pending" && hasPermission(principal, "approvals.decide") && (
                <>
                  <button className="button compact primary" type="button" onClick={() => approvalMutation("Approve", "identity.approval.decide", (key) => client.decideApproval(approval, "approve", key), client, setNotice, load)}>Approve</button>
                  <button className="button compact danger" type="button" onClick={() => approvalMutation("Reject", "identity.approval.decide", (key) => client.decideApproval(approval, "reject", key), client, setNotice, load)}>Reject</button>
                </>
              )}
              {approval.status === "approved" && hasPermission(principal, "approvals.execute") && (
                <button className="button compact primary" type="button" onClick={() => approvalMutation("Execute", "identity.approval.execute", (key) => client.executeApproval(approval, key), client, setNotice, load)}>Execute after recheck</button>
              )}
              {approval.status === "pending" && hasPermission(principal, "approvals.cancel") && (
                <button className="button compact" type="button" onClick={() => approvalMutation("Cancel", undefined, (key) => client.cancelApproval(approval, key), client, setNotice, load)}>Cancel</button>
              )}
            </div>
          </article>
        ))}
      </div>
      {!approvals && <InlineState kind="loading">Loading approvals from server truth…</InlineState>}
      {approvals?.length === 0 && <InlineState kind="empty">No approvals are visible to this principal.</InlineState>}
    </section>
  );
}

function approvalMutation(
  label: string,
  stepUpAction: string | undefined,
  action: (idempotencyKey: string) => Promise<unknown>,
  client: APIClient,
  setNotice: NoticeSetter,
  load: () => Promise<void>,
): void {
  const key = newIdempotencyKey(`approval-${label.toLowerCase()}`);
  void runMutation(label, () => action(key), setNotice, load, stepUpAction, client);
}

async function runMutation<T>(
  label: string,
  action: () => Promise<T>,
  setNotice: NoticeSetter,
  onSuccess: (result: T) => void | Promise<void>,
  stepUpAction?: string,
  client?: APIClient,
): Promise<void> {
  setNotice({kind: "pending", message: `${label} is being checked and committed…`});
  try {
    const result = await action();
    await onSuccess(result);
    setNotice((current) => current?.kind === "success" ? current : {kind: "success", message: `${label} completed from durable server truth.`});
  } catch (error) {
    if (isAPIError(error) && error.code === "STEP_UP_REQUIRED" && stepUpAction && client) {
      const verificationKey = newIdempotencyKey("step-up");
      setNotice({
        kind: "step-up",
        message: `${label} needs fresh action-bound verification. Atlas will not replay the pending mutation automatically.`,
        verify: () => {
          void client.stepUp(stepUpAction, verificationKey).then((challenge) => {
            window.location.assign(challenge.authorization_url);
          }).catch((stepUpError) => setNotice(noticeFromError("Start verification", stepUpError)));
        },
      });
      return;
    }
    const notice = noticeFromError(label, error);
    if (notice.kind === "ambiguous") {
      notice.retry = () => { void runMutation(label, action, setNotice, onSuccess, stepUpAction, client); };
    }
    setNotice(notice);
  }
}

function noticeFromError(label: string, error: unknown): Notice {
  if (!isAPIError(error)) {
    return {kind: "failed", message: `${label} failed safely. No browser-side authority is assumed.`};
  }
  if (error.responseLost) {
    return {kind: "ambiguous", message: `${label} may have committed, but the response was lost. Retry uses the same in-memory idempotency key.`};
  }
  if (error.status === 401) {
    return {kind: "denied", message: "The server session is no longer valid. Sign in again before any action."};
  }
  if (error.status === 403) {
    return {kind: "denied", message: `${label} is not authorized for the current principal, purpose, assurance, or tenant context.`};
  }
  if (error.status === 409 || error.status === 412) {
    return {kind: "stale", message: `${label} was blocked because server state changed. Refresh and make an explicit new decision.`};
  }
  return {kind: "failed", message: `${label} failed safely (${error.code}).${error.retryable ? " Retry after checking readiness." : ""}`};
}

function NoticeView({notice}: {notice: Notice | undefined}): ReactNode {
  if (!notice) return null;
  return (
    <div className={`inline-state ${notice.kind}`} role={notice.kind === "failed" || notice.kind === "denied" ? "alert" : "status"}>
      <span>{notice.message}</span>
      {notice.retry && <button className="text-button" type="button" onClick={notice.retry}>Retry same command</button>}
      {notice.verify && <button className="button compact primary" type="button" onClick={notice.verify}>Verify and return</button>}
    </div>
  );
}

function PanelHeading({title, onRefresh}: {title: string; onRefresh: () => void}): ReactNode {
  return <div className="panel-heading"><h2>{title}</h2><button className="text-button" type="button" onClick={onRefresh}>Refresh</button></div>;
}

function StatePanel({title, kind, children}: {title: string; kind: string; children: ReactNode}): ReactNode {
  return <section className="panel" data-state={kind}><h2>{title}</h2><p>{children}</p></section>;
}

function InlineState({kind, children}: {kind: string; children: ReactNode}): ReactNode {
  return <div className={`inline-state ${kind}`} role={kind === "failed" || kind === "denied" ? "alert" : "status"}>{children}</div>;
}

function PermissionList({permissions}: {permissions: readonly string[]}): ReactNode {
  if (permissions.length === 0) return <InlineState kind="empty">No standing permissions.</InlineState>;
  return <ul className="permission-list">{permissions.map((permission) => <li key={permission}>{permission}</li>)}</ul>;
}

function SignedOut({config, pageRestore}: {config: RuntimeConfig; pageRestore: PageRestoreKind}): ReactNode {
  const summary = summarizeBrowserStorage(window.localStorage, window.sessionStorage);
  return (
    <section className="panel" data-testid="signed-out">
      <div className="eyebrow">Security boundary complete</div>
      <h2>Client state cleared</h2>
      <p>Protected routes recheck the server session after reload, cross-tab security changes, and browser history restoration.</p>
      <dl className="fact-grid">
        <div><dt>Navigation restore</dt><dd data-testid="page-restore" data-page-restore={pageRestore}>{pageRestore}</dd></div>
        <div><dt>Atlas local storage</dt><dd>{summary.localAtlasEntries}</dd></div>
        <div><dt>Unexpected Atlas state</dt><dd>{summary.unexpectedAtlasEntries ? "detected — fail closed" : "absent"}</dd></div>
        <div><dt>Environment</dt><dd>{config.environment}</dd></div>
      </dl>
      <a className="button" href="/">Return to identity overview</a>
    </section>
  );
}

export function RouteFailure({config, path}: {config: RuntimeConfig; path: string}): ReactNode {
  const route = routePopulation[path] ? `${routePopulation[path]} route` : "Requested route";
  return (
    <>
      <div className="environment-banner" role="status">{config.banner}</div>
      <main className="shell-frame">
        <section className="panel" role="alert" data-testid="route-failure">
          <h2>Route unavailable</h2>
          <p>{route} could not render. Client state was cleared, no command was submitted, and no financial state exists.</p>
          <a className="button" href="/">Return to the identity overview</a>
        </section>
      </main>
    </>
  );
}

function UnknownRoute(): ReactNode {
  return <StatePanel title="Route unavailable" kind="denied">This route is not part of the Phase 01 identity surface.</StatePanel>;
}

function usePath(): string {
  const [path, setPath] = useState(window.location.pathname);
  useEffect(() => {
    const update = () => setPath(window.location.pathname);
    window.addEventListener("popstate", update);
    return () => window.removeEventListener("popstate", update);
  }, []);
  return path;
}

function navigate(path: string, replace = false): void {
  if (replace) window.history.replaceState({}, "", path);
  else window.history.pushState({}, "", path);
  window.dispatchEvent(new PopStateEvent("popstate"));
}

function currentNavigationType(): string | undefined {
  const [navigation] = window.performance.getEntriesByType("navigation");
  if (!navigation || !("type" in navigation)) return undefined;
  return String(navigation.type);
}

function hasPermission(principal: Principal, permission: string): boolean {
  return principal.permissions.includes(permission);
}

function formatTime(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? "invalid time" : date.toLocaleString();
}

function authLabel(status: AuthState["status"]): string {
  if (status === "loading") return "Checking session";
  if (status === "failed") return "Identity service unavailable";
  return "Signed out";
}
