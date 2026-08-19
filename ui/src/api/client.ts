import type { components } from "./schema";

export type LintScan = components["schemas"]["storage.LintScan"];
export type LintIssue = components["schemas"]["storage.LintIssue"];
export type FixPlan = components["schemas"]["storage.FixPlan"];
export type FixHistory = components["schemas"]["storage.FixHistory"];
export type RollbackHistory = components["schemas"]["storage.RollbackHistory"];
export type RestErr = components["schemas"]["golangci_backend_frontiir_utils.RestErr"];

const API_BASE = import.meta.env.VITE_API_BASE ?? "/api/v1";
const API_KEY_STORAGE_KEY = "golangci_api_key";

export function getApiKey(): string {
  return localStorage.getItem(API_KEY_STORAGE_KEY) ?? "";
}

export function setApiKey(key: string): void {
  localStorage.setItem(API_KEY_STORAGE_KEY, key);
}

export class ApiError extends Error {
  status: number;
  restErr: RestErr;

  constructor(status: number, restErr: RestErr) {
    super(restErr.message ?? `request failed with status ${status}`);
    this.status = status;
    this.restErr = restErr;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": getApiKey(),
      ...init?.headers,
    },
  });

  if (!res.ok) {
    const restErr: RestErr = await res.json().catch(() => ({ message: res.statusText }));
    throw new ApiError(res.status, restErr);
  }

  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

// -- Scans --------------------------------------------------------------

export function createScan(body: { repo_ref: string; branch: string }) {
  return request<{ id: string; status: string }>("/scans", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function getScan(id: string) {
  return request<LintScan>(`/scans/${id}`);
}

export function getScanIssues(id: string, params?: { limit?: number; offset?: number }) {
  const qs = new URLSearchParams();
  if (params?.limit) qs.set("limit", String(params.limit));
  if (params?.offset) qs.set("offset", String(params.offset));
  const suffix = qs.toString() ? `?${qs}` : "";
  return request<LintIssue[]>(`/scans/${id}/issues${suffix}`);
}

export function getIssue(id: string) {
  return request<LintIssue>(`/issues/${id}`);
}

// -- Plans ----------------------------------------------------------------

export function createPlan(body: { issue_ids: string[] }) {
  return request<FixPlan | { id: string; status: string }>("/plans", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function getPlan(id: string) {
  return request<FixPlan>(`/plans/${id}`);
}

export function approvePlan(id: string, body: { approve?: boolean; confirmed?: boolean }) {
  return request<FixPlan>(`/plans/${id}/approve`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

// -- Fixes ------------------------------------------------------------------

export function createFix(body: { plan_id: string }) {
  return request<{ id: string; result: string }>("/fixes", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function getFix(id: string) {
  return request<FixHistory>(`/fixes/${id}`);
}

// -- Rollbacks -----------------------------------------------------------

export function createRollback(body: { fix_history_id: string }) {
  return request<{ id: string; result: string }>("/rollbacks", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function getRollback(id: string) {
  return request<RollbackHistory>(`/rollbacks/${id}`);
}
