import { useParams } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { createRollback, getFix, getRollback } from "../api/client";
import { StatusStepper, type Step } from "../components/StatusStepper";
import { useState } from "react";
import { useLang } from "../lib/lang";
import { getStrings } from "../lib/strings";

// Screen 4 — Fix Progress (plans/05-ui-design.md). FixHistory.result only
// has 3 real states (applying|passed|failed) -- Applying and Re-scanning
// are shown as one combined worker-queued phase (see worker/fix.go: apply
// + rescan run as a single job), not fabricated separate progress.
function stepsFor(result: string | undefined, t: ReturnType<typeof getStrings>["fixProgress"]): Step[] {
  const applyDone = result === "passed" || result === "failed";
  return [
    { label: t.stepQueued, state: "done" },
    { label: t.stepApplying, state: applyDone ? "done" : "active" },
    { label: t.stepRescanning, state: applyDone ? "done" : "pending" },
    {
      label: t.stepResult,
      state: result === "passed" ? "done" : result === "failed" ? "failed" : "pending",
    },
  ];
}

export function FixProgress() {
  const { id } = useParams<{ id: string }>();
  const { lang } = useLang();
  const t = getStrings(lang).fixProgress;
  const [rollbackId, setRollbackId] = useState<string | null>(null);

  const fixQuery = useQuery({
    queryKey: ["fix", id],
    queryFn: () => getFix(id!),
    enabled: !!id,
    refetchInterval: (query) => (query.state.data?.result === "applying" ? 2000 : false),
  });

  const rollbackMutation = useMutation({
    mutationFn: () => createRollback({ fix_history_id: id! }),
    onSuccess: (res) => setRollbackId(res.id),
  });

  const rollbackQuery = useQuery({
    queryKey: ["rollback", rollbackId],
    queryFn: () => getRollback(rollbackId!),
    enabled: !!rollbackId,
    refetchInterval: (query) => (query.state.data?.result === "reverting" ? 2000 : false),
  });

  if (!id) return <div className="page">{t.noFixSelected}</div>;
  if (fixQuery.isLoading) return <div className="page">{t.loading}</div>;
  if (fixQuery.isError)
    return <div className="page error-text">{(fixQuery.error as Error).message}</div>;

  const fix = fixQuery.data!;

  return (
    <div className="page">
      <h1>{t.title}</h1>
      <StatusStepper steps={stepsFor(fix.result, t)} />
      <p className="status-line">
        {t.branch} <code>{fix.branch_name ?? "—"}</code> · {t.diff}{" "}
        <code>{fix.diff_ref ?? "pending"}</code> ·{" "}
        {fix.result === "applying" && t.applyingRescanPending}
        {fix.result === "passed" && t.rescanPassed}
        {fix.result === "failed" && t.fixFailed}
      </p>

      {fix.result === "passed" && !rollbackId && (
        <button className="btn-ghost" disabled={rollbackMutation.isPending} onClick={() => rollbackMutation.mutate()}>
          {t.rollbackThisFix}
        </button>
      )}

      {rollbackQuery.data && (
        <p className="status-line">{t.rollbackStatus(rollbackQuery.data.result ?? "")}</p>
      )}
    </div>
  );
}
