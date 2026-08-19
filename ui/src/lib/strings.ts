import type { Lang } from "./lang";

interface Strings {
  nav: { dashboard: string; history: string };
  dashboard: {
    title: string;
    totalIssues: string;
    critical: string;
    lastScanStatus: string;
    lastScanId: string;
    noneYet: string;
    repoPath: string;
    branch: string;
    scanButton: string;
    viewRecentScans: string;
    viewIssues: (n: string | number) => string;
  };
  issueList: {
    titlePrefix: string;
    noScanTitle: string;
    noScanBody: string;
    severityAll: string;
    severityLabel: (s: string) => string;
    linterAll: string;
    linterLabel: (l: string | undefined) => string;
    fileFilterPlaceholder: string;
    selected: (n: number) => string;
    viewPlan: string;
    loading: string;
    colFile: string;
    colLine: string;
    colLinter: string;
    colSeverity: string;
    colMessage: string;
    colSuggestedFix: string;
    colStatus: string;
    planRequested: string;
    needsAiPlan: string;
  };
  planViewer: {
    title: string;
    generating: string;
    issueSummary: string;
    issueLinter: string;
    issueFile: string;
    issueLine: string;
    issueColumn: string;
    issueMessage: string;
    codeContext: string;
    rootCause: string;
    currentBehavior: string;
    fixStrategy: string;
    recommendedFix: string;
    before: string;
    after: string;
    sideEffects: string;
    risk: string;
    breakingChange: string;
    yes: string;
    no: string;
    filesImpacted: string;
    impactAnalysis: string;
    affectedFile: string;
    affectedPackage: string;
    affectedSymbol: string;
    callers: string;
    recommendedTests: string;
    testPlan: string;
    acceptanceCriteria: string;
    status: string;
    reject: string;
    approveApply: string;
    confirmWarning: string;
    confirmApprove: string;
    applyFix: string;
    rejected: string;
    noPlanSelected: string;
    loadingPlan: string;
  };
  history: {
    title: string;
    tabs: { scans: string; fixes: string; rollbacks: string };
    notAvailable: (label: string, path: string) => string;
  };
  fixProgress: {
    title: string;
    noFixSelected: string;
    loading: string;
    branch: string;
    diff: string;
    applyingRescanPending: string;
    rescanPassed: string;
    fixFailed: string;
    rollbackThisFix: string;
    rollbackStatus: (r: string) => string;
    stepQueued: string;
    stepApplying: string;
    stepRescanning: string;
    stepResult: string;
  };
}

const STRINGS: Record<Lang, Strings> = {
  en: {
    nav: { dashboard: "Dashboard", history: "History" },
    dashboard: {
      title: "Dashboard",
      totalIssues: "Total issues",
      critical: "Critical",
      lastScanStatus: "Last scan status",
      lastScanId: "Last scan ID",
      noneYet: "none yet",
      repoPath: "Repo path",
      branch: "Branch",
      scanButton: "▶ Scan Lint",
      viewRecentScans: "View recent scans",
      viewIssues: (n) => `View ${n} issue(s) from this scan →`,
    },
    issueList: {
      titlePrefix: "Issue List — scan",
      noScanTitle: "Issue List",
      noScanBody: "No scan selected yet — trigger a scan from the Dashboard first.",
      severityAll: "Severity: All",
      severityLabel: (s) => `Severity: ${s}`,
      linterAll: "Linter: All",
      linterLabel: (l) => `Linter: ${l}`,
      fileFilterPlaceholder: "File contains…",
      selected: (n) => `${n} selected`,
      viewPlan: "View Plan",
      loading: "Loading…",
      colFile: "File",
      colLine: "Line:Col",
      colLinter: "Linter",
      colSeverity: "Severity",
      colMessage: "Message",
      colSuggestedFix: "Suggested fix",
      colStatus: "Status",
      planRequested: "Plan requested",
      needsAiPlan: "Needs AI plan",
    },
    planViewer: {
      title: "Plan Viewer",
      generating: "AI fix plan is still generating — polling…",
      issueSummary: "Issue summary",
      issueLinter: "Linter",
      issueFile: "File",
      issueLine: "Line",
      issueColumn: "Column",
      issueMessage: "Message",
      codeContext: "Original code context",
      rootCause: "Root cause",
      currentBehavior: "Current behavior",
      fixStrategy: "Fix strategy",
      recommendedFix: "Recommended fix",
      before: "Before",
      after: "After",
      sideEffects: "Possible side effects",
      risk: "Risk",
      breakingChange: "Breaking change",
      yes: "Yes",
      no: "No",
      filesImpacted: "Files impacted",
      impactAnalysis: "Impact analysis",
      affectedFile: "Affected file",
      affectedPackage: "Affected package",
      affectedSymbol: "Affected symbol",
      callers: "Callers",
      recommendedTests: "Recommended tests",
      testPlan: "Test plan",
      acceptanceCriteria: "Acceptance criteria",
      status: "Status",
      reject: "Reject",
      approveApply: "Approve & Apply Fix",
      confirmWarning:
        "This plan is high-risk / a breaking change — I confirm I want to approve it anyway.",
      confirmApprove: "Confirm Approve",
      applyFix: "Apply Fix",
      rejected: "This plan was rejected.",
      noPlanSelected: "No plan selected.",
      loadingPlan: "Loading plan…",
    },
    history: {
      title: "History",
      tabs: { scans: "Scan history", fixes: "Applied fixes", rollbacks: "Rollback history" },
      notAvailable: (label, path) =>
        `${label} isn't available yet — the backend's ${path} endpoint is still a placeholder.`,
    },
    fixProgress: {
      title: "Fix Progress",
      noFixSelected: "No fix selected.",
      loading: "Loading…",
      branch: "Branch",
      diff: "diff",
      applyingRescanPending: "applying + re-scanning pending",
      rescanPassed: "re-scan passed",
      fixFailed: "fix failed — see logs",
      rollbackThisFix: "Roll back this fix",
      rollbackStatus: (r) => `Rollback status: ${r}`,
      stepQueued: "Queued",
      stepApplying: "Applying",
      stepRescanning: "Re-scanning",
      stepResult: "Result",
    },
  },
  my: {
    nav: { dashboard: "ဒက်ရှ်ဘုတ်", history: "မှတ်တမ်း" },
    dashboard: {
      title: "ဒက်ရှ်ဘုတ်",
      totalIssues: "Issue စုစုပေါင်း",
      critical: "အရေးကြီးဆုံး",
      lastScanStatus: "နောက်ဆုံး Scan အခြေအနေ",
      lastScanId: "နောက်ဆုံး Scan ID",
      noneYet: "မရှိသေးပါ",
      repoPath: "Repo လမ်းကြောင်း",
      branch: "Branch",
      scanButton: "▶ Lint Scan စလုပ်ရန်",
      viewRecentScans: "လတ်တလော Scan များကြည့်ရန်",
      viewIssues: (n) => `ဒီ scan ကနေ issue ${n} ခုကို ကြည့်ရန် →`,
    },
    issueList: {
      titlePrefix: "Issue စာရင်း — scan",
      noScanTitle: "Issue စာရင်း",
      noScanBody: "Scan တစ်ခုမှ မရွေးထားသေးပါ — Dashboard ကနေ scan အရင်စလုပ်ပါ။",
      severityAll: "Severity: အားလုံး",
      severityLabel: (s) => `Severity: ${s}`,
      linterAll: "Linter: အားလုံး",
      linterLabel: (l) => `Linter: ${l}`,
      fileFilterPlaceholder: "File နာမည် ပါဝင်ရမည်…",
      selected: (n) => `${n} ခု ရွေးထားသည်`,
      viewPlan: "Plan ကြည့်ရန်",
      loading: "ဖွင့်နေသည်…",
      colFile: "File",
      colLine: "Line:Col",
      colLinter: "Linter",
      colSeverity: "Severity",
      colMessage: "ပြဿနာ",
      colSuggestedFix: "အကြံပြု Fix",
      colStatus: "အခြေအနေ",
      planRequested: "Plan တောင်းထားပြီး",
      needsAiPlan: "AI Plan လိုအပ်သည်",
    },
    planViewer: {
      title: "Plan ကြည့်ရှုရန်",
      generating: "AI fix plan ကို generate လုပ်နေဆဲ — စောင့်ပါ…",
      issueSummary: "Issue အကျဉ်းချုပ်",
      issueLinter: "Linter",
      issueFile: "File",
      issueLine: "Line",
      issueColumn: "Column",
      issueMessage: "ပြဿနာ",
      codeContext: "မူရင်း Code Context",
      rootCause: "အကြောင်းရင်း",
      currentBehavior: "လက်ရှိအခြေအနေ",
      fixStrategy: "Fix Strategy",
      recommendedFix: "အကြံပြု Fix",
      before: "မပြင်မီ",
      after: "ပြင်ပြီးနောက်",
      sideEffects: "ဖြစ်လာနိုင်သော Side Effects",
      risk: "အန္တရာယ်အဆင့်",
      breakingChange: "Breaking Change ဖြစ်နိုင်ခြေ",
      yes: "ဟုတ်ကဲ့",
      no: "မဟုတ်ပါ",
      filesImpacted: "သက်ရောက်သော Files",
      impactAnalysis: "Impact Analysis",
      affectedFile: "သက်ရောက်သော File",
      affectedPackage: "သက်ရောက်သော Package",
      affectedSymbol: "သက်ရောက်သော Symbol",
      callers: "ခေါ်သုံးသူများ (Callers)",
      recommendedTests: "အကြံပြု Test များ",
      testPlan: "Test Plan",
      acceptanceCriteria: "Acceptance Criteria",
      status: "အခြေအနေ",
      reject: "ငြင်းပယ်ရန်",
      approveApply: "အတည်ပြုပြီး Fix လုပ်ရန်",
      confirmWarning:
        "ဒီ Plan သည် အန္တရာယ်များသော / Breaking Change ဖြစ်နိုင်ခြေရှိပါသည် — ဒါကို သိပြီး အတည်ပြုလိုကြောင်း confirm ပါ။",
      confirmApprove: "အတည်ပြုရန် Confirm",
      applyFix: "Fix ကို အသုံးချရန်",
      rejected: "ဒီ Plan ကို ငြင်းပယ်ထားပါသည်။",
      noPlanSelected: "Plan တစ်ခုမှ မရွေးထားပါ။",
      loadingPlan: "Plan ဖွင့်နေသည်…",
    },
    history: {
      title: "မှတ်တမ်း",
      tabs: {
        scans: "Scan မှတ်တမ်း",
        fixes: "အသုံးချထားသော Fix များ",
        rollbacks: "Rollback မှတ်တမ်း",
      },
      notAvailable: (label, path) =>
        `${label} ကို အသင့်မဖြစ်သေးပါ — backend ရဲ့ ${path} endpoint က placeholder အဖြစ်သာ ရှိနေပါသေးသည်။`,
    },
    fixProgress: {
      title: "Fix Progress",
      noFixSelected: "Fix တစ်ခုမှ မရွေးထားပါ။",
      loading: "ဖွင့်နေသည်…",
      branch: "Branch",
      diff: "diff",
      applyingRescanPending: "fix အသုံးချနေဆဲ + re-scan ဆက်လက်စောင့်ဆိုင်းနေသည်",
      rescanPassed: "re-scan အောင်မြင်ပါသည်",
      fixFailed: "fix မအောင်မြင်ပါ — log ကြည့်ပါ",
      rollbackThisFix: "ဒီ Fix ကို Rollback လုပ်ရန်",
      rollbackStatus: (r) => `Rollback အခြေအနေ: ${r}`,
      stepQueued: "Queue ထဲရောက်ပြီး",
      stepApplying: "Fix အသုံးချနေဆဲ",
      stepRescanning: "ပြန်လည် Scan နေဆဲ",
      stepResult: "ရလဒ်",
    },
  },
};

export function getStrings(lang: Lang): Strings {
  return STRINGS[lang];
}
