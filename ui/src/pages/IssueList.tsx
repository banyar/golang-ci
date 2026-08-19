import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { createPlan, getScanIssues } from "../api/client";
import { SeverityBadge } from "../components/SeverityBadge";
import { useLang } from "../lib/lang";
import { getStrings } from "../lib/strings";

// Screen 2 — Issue List (plans/05-ui-design.md). Issue selection is
// sessionStorage-backed and keyed by scan, per the state-ownership table:
// "Survives filter/pagination changes; cleared on scan change."
function selectionKey(scanId: string) {
  return `golangci_selection_${scanId}`;
}

function loadSelection(scanId: string): Set<string> {
  const raw = sessionStorage.getItem(selectionKey(scanId));
  return raw ? new Set(JSON.parse(raw)) : new Set();
}

function saveSelection(scanId: string, ids: Set<string>) {
  sessionStorage.setItem(selectionKey(scanId), JSON.stringify([...ids]));
}

export function IssueList() {
  const { scanId } = useParams<{ scanId: string }>();
  const navigate = useNavigate();
  const { lang } = useLang();
  const t = getStrings(lang).issueList;
  const [severityFilter, setSeverityFilter] = useState("all");
  const [linterFilter, setLinterFilter] = useState("all");
  const [fileFilter, setFileFilter] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (scanId) setSelected(loadSelection(scanId));
  }, [scanId]);

  const issuesQuery = useQuery({
    queryKey: ["scan-issues", scanId],
    queryFn: () => getScanIssues(scanId!, { limit: 200 }),
    enabled: !!scanId,
  });

  const planMutation = useMutation({
    mutationFn: (issueIds: string[]) => createPlan({ issue_ids: issueIds }),
    onSuccess: (res) => navigate(`/plans/${res.id}`),
  });

  const linters = useMemo(
    () => [...new Set((issuesQuery.data ?? []).map((i) => i.linter).filter(Boolean))],
    [issuesQuery.data],
  );

  const filtered = (issuesQuery.data ?? []).filter((issue) => {
    if (severityFilter !== "all" && issue.severity !== severityFilter) return false;
    if (linterFilter !== "all" && issue.linter !== linterFilter) return false;
    if (fileFilter && !issue.file_path?.includes(fileFilter)) return false;
    return true;
  });

  function toggle(id: string) {
    if (!scanId) return;
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setSelected(next);
    saveSelection(scanId, next);
  }

  if (!scanId) {
    return (
      <div className="page">
        <h1>{t.noScanTitle}</h1>
        <p>{t.noScanBody}</p>
      </div>
    );
  }

  return (
    <div className="page">
      <h1>
        {t.titlePrefix} {scanId}
      </h1>

      <div className="toolbar">
        <select value={severityFilter} onChange={(e) => setSeverityFilter(e.target.value)}>
          <option value="all">{t.severityAll}</option>
          {["critical", "high", "medium", "low", "info"].map((s) => (
            <option key={s} value={s}>
              {t.severityLabel(s)}
            </option>
          ))}
        </select>
        <select value={linterFilter} onChange={(e) => setLinterFilter(e.target.value)}>
          <option value="all">{t.linterAll}</option>
          {linters.map((l) => (
            <option key={l} value={l}>
              {t.linterLabel(l)}
            </option>
          ))}
        </select>
        <input
          placeholder={t.fileFilterPlaceholder}
          value={fileFilter}
          onChange={(e) => setFileFilter(e.target.value)}
        />
      </div>

      {selected.size > 0 && (
        <div className="bulk-bar">
          {t.selected(selected.size)} —{" "}
          <button
            className="btn-primary"
            disabled={planMutation.isPending}
            onClick={() => planMutation.mutate([...selected])}
          >
            {t.viewPlan}
          </button>
        </div>
      )}

      {issuesQuery.isLoading && <p>{t.loading}</p>}
      {issuesQuery.isError && <p className="error-text">{(issuesQuery.error as Error).message}</p>}

      {issuesQuery.data && (
        <table className="issue-table">
          <thead>
            <tr>
              <th></th>
              <th>{t.colFile}</th>
              <th>{t.colLine}</th>
              <th>{t.colLinter}</th>
              <th>{t.colSeverity}</th>
              <th>{t.colMessage}</th>
              <th>{t.colSuggestedFix}</th>
              <th>{t.colStatus}</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((issue) => (
              <tr key={issue.id}>
                <td>
                  <input
                    type="checkbox"
                    checked={selected.has(issue.id!)}
                    onChange={() => toggle(issue.id!)}
                  />
                </td>
                <td>{issue.file_path}</td>
                <td>
                  {issue.line}:{issue.column}
                </td>
                <td>{issue.linter}</td>
                <td>
                  <SeverityBadge severity={issue.severity} />
                </td>
                <td>
                  {issue.message}
                  {lang === "my" && issue.reason_my && (
                    <div className="issue-reason-my">{issue.reason_my}</div>
                  )}
                </td>
                <td>{issue.plans?.length ? t.planRequested : t.needsAiPlan}</td>
                <td>{issue.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
