import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { useLang } from "../lib/lang";
import { getStrings } from "../lib/strings";

// Screen 5 — History (plans/05-ui-design.md). /history/* is still a 501
// stub on the backend (golangci/backend/api/router.go's shared `stub`
// handler) -- this renders that as an explicit "not implemented yet"
// state per tab, not a generic error or fabricated data.
const TAB_KEYS = ["scans", "fixes", "rollbacks"] as const;
const TAB_PATHS: Record<(typeof TAB_KEYS)[number], string> = {
  scans: "/history/scans",
  fixes: "/history/fixes",
  rollbacks: "/history/rollbacks",
};

export function History() {
  const { lang } = useLang();
  const t = getStrings(lang).history;
  const TABS = TAB_KEYS.map((key) => ({ key, label: t.tabs[key], path: TAB_PATHS[key] }));
  const [tab, setTab] = useState<(typeof TAB_KEYS)[number]>("scans");
  const active = TABS.find((tb) => tb.key === tab)!;

  const query = useQuery({
    queryKey: ["history", tab],
    queryFn: async () => {
      const res = await fetch(`/api/v1${active.path}`, {
        headers: { "X-API-Key": localStorage.getItem("golangci_api_key") ?? "" },
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new ApiError(res.status, body);
      }
      return res.json();
    },
  });

  return (
    <div className="page">
      <h1>{t.title}</h1>
      <div className="tabs">
        {TABS.map((tb) => (
          <button
            key={tb.key}
            className={tb.key === tab ? "tab tab-active" : "tab"}
            onClick={() => setTab(tb.key)}
          >
            {tb.label}
          </button>
        ))}
      </div>

      {query.isError && query.error instanceof ApiError && query.error.status === 501 ? (
        <p className="muted">
          {t.notAvailable(active.label, active.path)} (see plans/16-review-checklist.md item H1)
        </p>
      ) : query.isError ? (
        <p className="error-text">{(query.error as Error).message}</p>
      ) : null}
    </div>
  );
}
