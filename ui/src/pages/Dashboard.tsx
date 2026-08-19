import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { createScan, getScan, getScanIssues } from "../api/client";
import { useLang } from "../lib/lang";
import { getStrings } from "../lib/strings";

// Screen 1 — Dashboard (plans/05-ui-design.md). No aggregate "all scans"
// endpoint exists on the backend yet (history/* is still a 501 stub), so
// tiles reflect only the most recently triggered scan in this session --
// not a fabricated all-time total.
export function Dashboard() {
  const navigate = useNavigate();
  const { lang } = useLang();
  const t = getStrings(lang).dashboard;
  const [repoRef, setRepoRef] = useState(
    "/home/ubuntu/FrontiirProjects/RT/rt-external-api-with-auto-remote-resolved",
  );
  const [branch, setBranch] = useState("ticket-create-sla");
  const [scanId, setScanId] = useState<string | null>(null);

  const scanMutation = useMutation({
    mutationFn: () => createScan({ repo_ref: repoRef, branch }),
    onSuccess: (res) => setScanId(res.id),
  });

  const scanQuery = useQuery({
    queryKey: ["scan", scanId],
    queryFn: () => getScan(scanId!),
    enabled: !!scanId,
    refetchInterval: (query) => (query.state.data?.status === "running" ? 2000 : false),
  });

  const issuesQuery = useQuery({
    queryKey: ["scan-issues", scanId],
    queryFn: () => getScanIssues(scanId!, { limit: 200 }),
    enabled: !!scanId && scanQuery.data?.status === "success",
  });

  const total = issuesQuery.data?.length ?? null;
  const critical = issuesQuery.data?.filter((i) => i.severity === "critical").length ?? null;

  return (
    <div className="page">
      <h1>{t.title}</h1>

      <div className="tile-grid">
        <div className="tile">
          <div className="tile-label">{t.totalIssues}</div>
          <div className="tile-value">{total ?? "—"}</div>
        </div>
        <div className="tile tile-critical">
          <div className="tile-label">{t.critical}</div>
          <div className="tile-value">{critical ?? "—"}</div>
        </div>
        <div className="tile tile-ok">
          <div className="tile-label">{t.lastScanStatus}</div>
          <div className="tile-value">{scanQuery.data?.status ?? t.noneYet}</div>
        </div>
        <div className="tile">
          <div className="tile-label">{t.lastScanId}</div>
          <div className="tile-value">{scanId ?? "—"}</div>
        </div>
      </div>

      <form
        className="scan-form"
        onSubmit={(e) => {
          e.preventDefault();
          scanMutation.mutate();
        }}
      >
        <label>
          {t.repoPath}
          <input
            value={repoRef}
            onChange={(e) => setRepoRef(e.target.value)}
            placeholder="/absolute/path/to/a/git/repo"
            required
          />
        </label>
        <label>
          {t.branch}
          <input value={branch} onChange={(e) => setBranch(e.target.value)} required />
        </label>
        <div className="actions">
          <button type="submit" className="btn-primary" disabled={scanMutation.isPending}>
            {t.scanButton}
          </button>
          <button type="button" className="btn-ghost" onClick={() => navigate("/history")}>
            {t.viewRecentScans}
          </button>
        </div>
      </form>

      {scanMutation.isError && (
        <p className="error-text">{(scanMutation.error as Error).message}</p>
      )}
      {scanId && scanQuery.data?.status === "success" && (
        <button className="btn-ghost" onClick={() => navigate(`/scans/${scanId}/issues`)}>
          {t.viewIssues(total ?? "…")}
        </button>
      )}
    </div>
  );
}
