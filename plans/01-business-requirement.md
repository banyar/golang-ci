# 01 — Business Requirement

**Date:** 2026-08-03
**Status:** Draft — needs BA/PM confirmation on marked items
**Source:** `golangci/dashboard.md` (original ask), `golangci/cmd/` (current tooling, for problem framing)
**Dependency:** [00-project-overview.md](00-project-overview.md)

## Background

ဒီ Repo ထဲမှာ golangci-lint ကို run ဖို့ CLI Tooling (`make lint`, `make lint-fixed-plan`, `make lint-fixed-plan-result`) ရှိပြီးသားပါတယ် — Scan → rule-based Fix Plan → Apply → Verify ဆိုတဲ့ Flow ကို Bash Script (`golangci/cmd/golangci-report.sh`, `lint-fixed-plan.sh`, `lint-fixed-plan-result.sh`) သုံးပြီး လုပ်ဆောင်ထားပါတယ်။ အသေးစိတ်ကို [02-current-workflow.md](02-current-workflow.md) မှာ ဖော်ပြပါမည်။

## Problem Statement

Lint Issue Triage/Fix Workflow အားလုံးက CLI-only ဖြစ်ပြီး, အောက်ပါ Limitation ရှိပါတယ် (`dashboard.md:10` ရဲ့ မူလ Framing အတိုင်း):

- Issue List ကို Table အနေနဲ့ Interactive ကြည့်၊ Filter၊ Multi-select လုပ်လို့ မရပါ (Terminal/Markdown Report ကိုသာ ဖတ်ရ).
- Fix Plan ဟာ Rule-based Template (`lint-fixed-plan.sh` ထဲက Hardcoded `case`/`switch` per Linter) ဖြစ်ပြီး, LLM-generated Contextual Reasoning မဟုတ်ပါ.
- Issue တစ်ခုချင်းစီကိုသာ (`N=<num>`) ကိုင်တွယ်ရပါတယ် — Batch Selection မရှိပါ.
- Scan History ကို Default အနေနဲ့ မသိမ်းပါ (`KEEP_HISTORY=0` — Run အသစ်တိုင်း အဟောင်းဖျက်) — Applied Fix/Rollback ကို Browse ကြည့်နိုင်တဲ့ UI လုံးဝမရှိပါ.
- Approval Gate (Human Sign-off မတိုင်ခင် Fix Apply မလုပ်ရ ဆိုတဲ့ Explicit Control) မရှိပါ — Developer တစ်ယောက်တည်း Local Editor ထဲမှာ Manual ပြင်ရပါတယ်.

## Business Goal

CLI-based Lint Triage/Fix ကို Web UI Dashboard အဖြစ် Upgrade လုပ်ပြီး: Scan → Interactive Issue List → AI-generated Fix Plan (Claude) → Human Approval → Apply → Auto Re-scan → Persistent History/Rollback ဆိုတဲ့ Loop တစ်ခုလုံးကို Browser ထဲကနေ Control လုပ်နိုင်အောင် လုပ်ဆောင်ခြင်း (`dashboard.md:1-9` အတိုင်း).

## Requestor

| Field | Value |
|---|---|
| Name | Banyar Sithu |
| Email | banyar.sithu@frontiir.net |
| Role | **Assumption (default, 2026-08-04)** — Requestor (Banyar Sithu) acts as the de facto BA/PM/Approver for this build cycle (single-person internal-tooling project). Reopen if a separate BA/PM is later assigned. |

## Success Criteria

**Assumption (default, 2026-08-04)** — no numeric target was ever stated in any source material, so a qualitative goal is adopted instead of inventing a false-precision number: (1) lint-issue triage/fix no longer requires the CLI for the common case, (2) a majority of newly-introduced lint issues get a plan+fix without a manual editor pass. Revisit with a real metric once the dashboard has real usage data.

## Constraints

| Constraint | Source | Type |
|---|---|---|
| GORM + MySQL stack ကိုသာ Reuse ရမည် (`git.frontiir.net/sa-dev/rtdatacore` Pattern) | Architecture Artifact §Database Design rationale | Technical — confirmed |
| Fix Service သည် `lint-fix/*` Branch Prefix အောက်မှာသာ Commit ခွင့်ရှိရမည်; `main`/`master` သို့ တိုက်ရိုက် Write လုံးဝ ခွင့်မပြု | Architecture Artifact §System Architecture, Security posture | Technical/Security — confirmed |
| Deadline / Budget / Team Size | — | **Assumption (default)**: none — internal tooling, no fixed deadline or budget ceiling stated or needed for an M1-scale build |
