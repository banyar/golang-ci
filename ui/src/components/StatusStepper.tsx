export type StepState = "done" | "active" | "pending" | "failed";

export interface Step {
  label: string;
  state: StepState;
}

const ICON: Record<StepState, string> = {
  done: "✓",
  active: "●",
  pending: "○",
  failed: "✕",
};

export function StatusStepper({ steps }: { steps: Step[] }) {
  return (
    <ol className="status-stepper">
      {steps.map((step) => (
        <li key={step.label} className={`stepper-step stepper-${step.state}`}>
          <span className="stepper-icon">{ICON[step.state]}</span>
          <span className="stepper-label">{step.label}</span>
        </li>
      ))}
    </ol>
  );
}
