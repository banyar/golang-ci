import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { ApiError, approvePlan, createFix, getPlan } from "../api/client";
import { pick, useLang } from "../lib/lang";
import { getStrings } from "../lib/strings";

// Screen 3 — Plan Viewer (plans/05-ui-design.md), extended 2026-08-19 to
// match before-fixed/*.md's section depth (see
// plans/2026-08-19-enrich-fixplan-generation.md). `json.RawMessage`
// columns (side_effects, impact_analysis, recommended_test_commands,
// acceptance_criteria, files_impacted, test_plan) come back from swag's
// codegen typed as `number[]` -- a cosmetic artifact of how it infers
// json.RawMessage, not the real runtime shape -- so every one of them is
// cast through `unknown` and rendered defensively here, same as this file
// already did for files_impacted/test_plan before this change.
function renderList(value: unknown) {
  if (value == null) return <span className="muted">—</span>;
  if (Array.isArray(value)) {
    if (value.length === 0) return <span className="muted">—</span>;
    return (
      <ul>
        {value.map((v, i) => (
          <li key={i}>{typeof v === "object" ? JSON.stringify(v) : String(v)}</li>
        ))}
      </ul>
    );
  }
  return <pre>{JSON.stringify(value, null, 2)}</pre>;
}

interface ImpactInfo {
  affected_file?: string;
  affected_package?: string;
  affected_symbol?: string;
  callers?: string[];
}

function parseImpact(value: unknown): ImpactInfo | null {
  if (value == null) return null;
  if (typeof value === "object" && !Array.isArray(value)) return value as ImpactInfo;
  return null;
}

export function PlanViewer() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { lang } = useLang();
  const t = getStrings(lang).planViewer;
  const [needsConfirm, setNeedsConfirm] = useState(false);
  const [confirmed, setConfirmed] = useState(false);

  const planQuery = useQuery({
    queryKey: ["plan", id],
    queryFn: () => getPlan(id!),
    enabled: !!id,
    refetchInterval: (query) => (query.state.data?.status === "generating" ? 2000 : false),
  });

  const approveMutation = useMutation({
    mutationFn: (approve: boolean) => approvePlan(id!, { approve, confirmed }),
    onError: (err) => {
      if (err instanceof ApiError && err.restErr.code === "CONFIRMATION_REQUIRED") {
        setNeedsConfirm(true);
      }
    },
  });

  const fixMutation = useMutation({
    mutationFn: () => createFix({ plan_id: id! }),
    onSuccess: (res) => navigate(`/fixes/${res.id}`),
  });

  if (!id) return <div className="page">{t.noPlanSelected}</div>;
  if (planQuery.isLoading) return <div className="page">{t.loadingPlan}</div>;
  if (planQuery.isError)
    return <div className="page error-text">{(planQuery.error as Error).message}</div>;

  const plan = planQuery.data!;

  if (plan.status === "generating") {
    return (
      <div className="page">
        <h1>{t.title}</h1>
        <p>{t.generating}</p>
      </div>
    );
  }

  const issue = plan.issues?.[0];
  const impact = parseImpact(plan.impact_analysis);

  return (
    <div className="page">
      <h1>{t.title}</h1>

      {issue && (
        <>
          <h2>{t.issueSummary}</h2>
          <table className="issue-summary-table">
            <tbody>
              <tr>
                <th>{t.issueLinter}</th>
                <td>{issue.linter}</td>
              </tr>
              <tr>
                <th>{t.issueFile}</th>
                <td>
                  <code>{issue.file_path}</code>
                </td>
              </tr>
              <tr>
                <th>{t.issueLine}</th>
                <td>{issue.line}</td>
              </tr>
              <tr>
                <th>{t.issueColumn}</th>
                <td>{issue.column}</td>
              </tr>
              <tr>
                <th>{t.issueMessage}</th>
                <td>{issue.message}</td>
              </tr>
            </tbody>
          </table>
        </>
      )}

      {plan.code_context && (
        <>
          <h2>{t.codeContext}</h2>
          <pre className="code-block code-context">{plan.code_context}</pre>
        </>
      )}

      <dl className="plan-detail">
        <dt>{t.rootCause}</dt>
        <dd>{pick(lang, plan.root_cause, plan.root_cause_my)}</dd>
        <dt>{t.currentBehavior}</dt>
        <dd>{pick(lang, plan.current_behavior, plan.current_behavior_my)}</dd>
      </dl>

      {plan.fix_strategy_code && (
        <>
          <h2>{t.fixStrategy}</h2>
          <pre className="code-block">{plan.fix_strategy_code}</pre>
        </>
      )}

      <dl className="plan-detail">
        <dt>{t.recommendedFix}</dt>
        <dd>{pick(lang, plan.recommended_fix, plan.recommended_fix_my)}</dd>
      </dl>

      {(plan.before_snippet || plan.after_snippet) && (
        <div className="before-after">
          <div>
            <h3>{t.before}</h3>
            <pre className="code-block">{plan.before_snippet || "—"}</pre>
          </div>
          <div>
            <h3>{t.after}</h3>
            <pre className="code-block">{plan.after_snippet || "—"}</pre>
          </div>
        </div>
      )}

      {plan.side_effects != null && (
        <>
          <h2>{t.sideEffects}</h2>
          {renderList(plan.side_effects)}
        </>
      )}

      <dl className="plan-detail">
        <dt>{t.risk}</dt>
        <dd className={`risk-${plan.risk_level}`}>{plan.risk_level}</dd>
        <dt>{t.breakingChange}</dt>
        <dd>{plan.breaking_change ? t.yes : t.no}</dd>
        <dt>{t.filesImpacted}</dt>
        <dd>{renderList(plan.files_impacted)}</dd>
      </dl>

      {impact && (
        <>
          <h2>{t.impactAnalysis}</h2>
          <dl className="plan-detail">
            <dt>{t.affectedFile}</dt>
            <dd>{impact.affected_file || "—"}</dd>
            <dt>{t.affectedPackage}</dt>
            <dd>{impact.affected_package || "—"}</dd>
            <dt>{t.affectedSymbol}</dt>
            <dd>{impact.affected_symbol || "—"}</dd>
            <dt>{t.callers}</dt>
            <dd>{renderList(impact.callers)}</dd>
          </dl>
        </>
      )}

      {plan.recommended_test_commands != null && (
        <>
          <h2>{t.recommendedTests}</h2>
          <pre className="code-block bash-block">
            {Array.isArray(plan.recommended_test_commands)
              ? plan.recommended_test_commands.join("\n")
              : JSON.stringify(plan.recommended_test_commands)}
          </pre>
        </>
      )}

      <dl className="plan-detail">
        <dt>{t.testPlan}</dt>
        <dd>{renderList(plan.test_plan)}</dd>
      </dl>

      {plan.acceptance_criteria != null && (
        <>
          <h2>{t.acceptanceCriteria}</h2>
          {renderList(plan.acceptance_criteria)}
        </>
      )}

      <dl className="plan-detail">
        <dt>{t.status}</dt>
        <dd>{plan.status}</dd>
      </dl>

      {plan.status === "pending" && (
        <div className="actions">
          <button className="btn-ghost" onClick={() => approveMutation.mutate(false)}>
            {t.reject}
          </button>
          <button className="btn-primary" onClick={() => approveMutation.mutate(true)}>
            {t.approveApply}
          </button>
        </div>
      )}

      {needsConfirm && (
        <div className="confirm-box">
          <label>
            <input
              type="checkbox"
              checked={confirmed}
              onChange={(e) => setConfirmed(e.target.checked)}
            />
            {t.confirmWarning}
          </label>
          <button
            className="btn-primary"
            disabled={!confirmed}
            onClick={() => approveMutation.mutate(true)}
          >
            {t.confirmApprove}
          </button>
        </div>
      )}

      {approveMutation.isError && !needsConfirm && (
        <p className="error-text">{(approveMutation.error as Error).message}</p>
      )}

      {plan.status === "approved" && (
        <button className="btn-primary" disabled={fixMutation.isPending} onClick={() => fixMutation.mutate()}>
          {t.applyFix}
        </button>
      )}
      {plan.status === "rejected" && <p>{t.rejected}</p>}
    </div>
  );
}
