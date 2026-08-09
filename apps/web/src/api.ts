import type {components} from "./generated/openapi";

export type Principal = components["schemas"]["Principal"];
export type Session = components["schemas"]["Session"];
export type Organization = components["schemas"]["Organization"];
export type OrganizationMember = components["schemas"]["OrganizationMember"];
export type Invitation = components["schemas"]["OrganizationInvitation"];
export type Credential = components["schemas"]["APICredential"];
export type CredentialSecret = components["schemas"]["APICredentialCreated"];
export type Approval = components["schemas"]["Approval"];
export type Problem = components["schemas"]["Problem"];

export type APIError = Error & {
  status: number;
  code: string;
  retryable: boolean;
  responseLost: boolean;
};

type Fetcher = typeof fetch;

type RequestOptions = {
  body?: unknown;
  idempotencyKey?: string;
  ifMatch?: string;
  purpose?: string;
};

export type APIClient = ReturnType<typeof createAPIClient>;

const csrfHeader = "X-Atlas-CSRF-Token";

export function createAPIClient(apiOrigin: string, fetcher: Fetcher = fetch) {
  const normalizedOrigin = apiOrigin.replace(/\/$/, "");
  let csrfToken = "";

  async function request<T>(method: string, path: string, options: RequestOptions = {}): Promise<T> {
    const headers = new Headers({Accept: "application/json"});
    if (options.body !== undefined) {
      headers.set("Content-Type", "application/json");
    }
    if (method !== "GET" && method !== "HEAD") {
      if (!csrfToken) {
        throw clientError("CSRF token unavailable", "CSRF_TOKEN_UNAVAILABLE");
      }
      headers.set(csrfHeader, csrfToken);
    }
    if (options.idempotencyKey) {
      headers.set("Idempotency-Key", options.idempotencyKey);
    }
    if (options.ifMatch) {
      headers.set("If-Match", options.ifMatch);
    }
    if (options.purpose) {
      headers.set("X-Atlas-Purpose", options.purpose);
    }

    let response: Response;
    try {
      response = await fetcher(normalizedOrigin + path, {
        method,
        headers,
        credentials: "include",
        cache: "no-store",
        body: options.body === undefined ? undefined : JSON.stringify(options.body),
      });
    } catch {
      throw clientError("The server outcome is unknown. Retry the same action safely.", "RESPONSE_LOST", true);
    }

    const rotatedCSRF = response.headers.get(csrfHeader);
    if (rotatedCSRF) {
      csrfToken = rotatedCSRF;
    }
    if (response.status === 401) {
      csrfToken = "";
    }
    if (!response.ok) {
      let problem: Partial<Problem> = {};
      try {
        problem = await response.json() as Problem;
      } catch {
        // A bounded generic error is safer than surfacing an untrusted response body.
      }
      const error = new Error(problem.title || "Request could not be completed") as APIError;
      error.status = response.status;
      error.code = problem.code || "REQUEST_FAILED";
      error.retryable = problem.retryable === true;
      error.responseLost = false;
      throw error;
    }
    if (response.status === 204) {
      return undefined as T;
    }
    return await response.json() as T;
  }

  function loginURL(population: "customer" | "merchant" | "workforce", returnTo: string): string {
    const query = new URLSearchParams({population, return_to: returnTo});
    return `${normalizedOrigin}/v1/auth/login?${query.toString()}`;
  }

  return {
    loginURL,
    clearSecurityState() {
      csrfToken = "";
    },
    current: () => request<Principal>("GET", "/v1/me"),
    sessions: async () => (await request<{data: Session[]}>("GET", "/v1/sessions")).data,
    revokeSession: (sessionID: string) => request<void>("DELETE", `/v1/sessions/${encodeURIComponent(sessionID)}`),
    revokeAll: (includeCurrent: boolean, idempotencyKey: string) => request<void>("POST", "/v1/sessions/revoke-all", {
      body: {include_current: includeCurrent}, idempotencyKey,
    }),
    logout: () => request<void>("POST", "/v1/logout"),
    stepUp: (action: string, idempotencyKey: string) => request<{authorization_url: string}>("POST", "/v1/step-up/challenges", {
      body: {action}, idempotencyKey,
    }),
    organizations: async () => (await request<{data: Organization[]}>("GET", "/v1/organizations")).data,
    switchOrganization: (organizationID: string) => request<Principal>("PUT", "/v1/me/active-organization", {
      body: {organization_id: organizationID},
    }),
    members: async (organizationID: string) => (await request<{data: OrganizationMember[]}>(
      "GET", `/v1/organizations/${encodeURIComponent(organizationID)}/members`,
      {purpose: "organization_administration"},
    )).data,
    invite: (organizationID: string, email: string, role: string, idempotencyKey: string) =>
      request<{invitation: Invitation; acceptance_token: string | null; secret_disclosed: boolean}>(
        "POST", `/v1/organizations/${encodeURIComponent(organizationID)}/invitations`,
        {body: {email, role}, idempotencyKey},
      ),
    updateMemberRole: (
      organizationID: string,
      member: OrganizationMember,
      role: string,
      idempotencyKey: string,
    ) => request<OrganizationMember | Approval>(
      "PATCH",
      `/v1/organizations/${encodeURIComponent(organizationID)}/members/${encodeURIComponent(member.id)}`,
      {
        body: {role, purpose: "organization_administration"},
        idempotencyKey,
        ifMatch: `"membership-v${member.version}"`,
      },
    ),
    removeMember: (organizationID: string, member: OrganizationMember, idempotencyKey: string) =>
      request<void>(
        "DELETE",
        `/v1/organizations/${encodeURIComponent(organizationID)}/members/${encodeURIComponent(member.id)}`,
        {idempotencyKey, ifMatch: `"membership-v${member.version}"`},
      ),
    credentials: async () => (await request<{data: Credential[]}>("GET", "/v1/api-credentials")).data,
    createCredential: (name: string, expiresInDays: number, idempotencyKey: string) =>
      request<CredentialSecret>("POST", "/v1/api-credentials", {
        body: {name, scopes: ["identity:read"], expires_in_days: expiresInDays, purpose: "credential_management"},
        idempotencyKey,
      }),
    rotateCredential: (credentialID: string, idempotencyKey: string) =>
      request<CredentialSecret>("POST", `/v1/api-credentials/${encodeURIComponent(credentialID)}/rotate`, {idempotencyKey}),
    revokeCredential: (credentialID: string, idempotencyKey: string) =>
      request<void>("DELETE", `/v1/api-credentials/${encodeURIComponent(credentialID)}`, {idempotencyKey}),
    approvals: async () => (await request<{data: Approval[]}>("GET", "/v1/approvals")).data,
    decideApproval: (approval: Approval, decision: "approve" | "reject", idempotencyKey: string) =>
      request<Approval>("POST", `/v1/approvals/${encodeURIComponent(approval.id)}/decisions`, {
        body: {decision, purpose: "approval_review", reason: null},
        idempotencyKey,
        ifMatch: `"${approval.version}"`,
      }),
    executeApproval: (approval: Approval, idempotencyKey: string) =>
      request<Approval>("POST", `/v1/approvals/${encodeURIComponent(approval.id)}/executions`, {
        idempotencyKey,
        ifMatch: `"${approval.version}"`,
      }),
    cancelApproval: (approval: Approval, idempotencyKey: string) =>
      request<Approval>("POST", `/v1/approvals/${encodeURIComponent(approval.id)}/cancellations`, {
        body: {reason: "cancelled from the Atlas synthetic console"},
        idempotencyKey,
        ifMatch: `"${approval.version}"`,
      }),
  };
}

export function newIdempotencyKey(prefix: string): string {
  return `${prefix}-${crypto.randomUUID()}`;
}

export function isAPIError(error: unknown): error is APIError {
  return error instanceof Error && "status" in error && "code" in error;
}

function clientError(message: string, code: string, responseLost = false): APIError {
  const error = new Error(message) as APIError;
  error.status = 0;
  error.code = code;
  error.retryable = responseLost;
  error.responseLost = responseLost;
  return error;
}
