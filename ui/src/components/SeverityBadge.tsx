const SEVERITY_CLASS: Record<string, string> = {
  critical: "severity-critical",
  high: "severity-high",
  medium: "severity-medium",
  low: "severity-low",
  info: "severity-info",
};

export function SeverityBadge({ severity }: { severity?: string }) {
  const key = (severity ?? "info").toLowerCase();
  const className = SEVERITY_CLASS[key] ?? "severity-info";
  return <span className={`severity-badge ${className}`}>{severity ?? "info"}</span>;
}
