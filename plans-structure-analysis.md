# Phase 1 — Document Analysis (Artifact Reference Study)

## 1.1 Artifact Structure

Artifact (`golangci-lint Web Dashboard — Architecture Design`) ရဲ့ Layout ကို layer အဆင့်လိုက် ခွဲကြည့်ရင်:

```
Header    → eyebrow label ("System Design Review · Draft · No Implementation")
            + H1 title
            + subtitle paragraph (one-paragraph elevator pitch)
            + meta-row (Scope / AI Layer / Status — key-value strip)
Nav Rail  → sticky in-page anchor links, sections 8 ခုကို တစ်ချက်ချင်း jump လုပ်နိုင်
Body      → <section id="..."> ၈ ခု, တစ်ခုချင်းစီ:
              - eyebrow number label ("01 — Layers", "02 — End to end", ...)
              - H2 section title
              - intro paragraph (ဘာကြောင့် ဒီ Decision လုပ်ရလဲ ရှင်းသော "Why" framing)
              - H3 subsection(s) — necessary ရင်သာ
              - "panel" box (diagram / table / wireframe mockup)
Footer    → one-line document identity strip
```

**အဓိက Insight**: Section အားလုံးက **"Decision + Why" → "Artifact ဖြင့် Operationalize"** ဆိုတဲ့ pattern တစ်ခုတည်းကို ထပ်ခါထပ်ခါ သုံးထားသည် — ဥပမာ "System Architecture" section မှာ prose paragraph က "ဘာကြောင့် Job Queue ခွဲထားရလဲ" ရှင်းပြီး၊ ပြီးမှ Component Diagram ဖြင့် ၎င်းဆုံးဖြတ်ချက်ကို visual proof အဖြစ် ပြသသည်။

## 1.2 Documentation Style

| ဂုဏ်သတ္တိ | Convention |
|---|---|
| Language mixing | Prose = Professional Business Myanmar; Technical identifiers (`Queue`, `FixPlan.status`, `git revert`) = English inline-code |
| Emphasis | Decision-critical noun/phrase ကို `<b>` bold — reader စကန်ဖတ်ရင် Decision point တွေကို ချက်ချင်း မြင်နိုင် |
| Diagram-first | Concept ရှုပ်ထွေးလေ (Sequence, State Machine, ER) Diagram နဲ့ ဖော်ပြမှု များလေ — prose တစ်ခုတည်းနဲ့ မရှင်းပါ |
| Tables for structured facts | Endpoint list, DB fields, Risk register, Component responsibility — **fact-dense content အားလုံး Table ဖြင့်** (paragraph ဖြင့် ရေးမထား) |
| Acceptance/Test format | **Given / When / Then ၃ တိုင် table** (Plan Viewer → Test Plan အပိုင်း) |
| Cross-referencing | Section တစ်ခုက နောက်ပိုင်း/ရှေ့ပိုင်း Section ကို anchor link ဖြင့် ညွှန်း (`→ Component Responsibilities → Security`) — content ကို **duplicate မလုပ်ဘဲ point ပြရုံ**

## 1.3 Folder Organization

Artifact ရဲ့ "Recommended folder structure" က **Single Responsibility ↔ Folder တစ်ခုချင်းစီ** ဆိုတဲ့ 1:1 mapping ကို သုံးထားသည်:

```
golangci/
├── api/       ← Gin handlers, validation, RBAC middleware
├── scanner/   ← golangci-lint process orchestration
├── parser/    ← JSON → LintIssue normalization
├── planner/   ← AI Layer client, prompt templates
├── fixer/     ← autofix + AI patch application
├── history/   ← audit read models
├── worker/    ← job queue consumer (Senior recommendation — user list မှာ မပါခဲ့)
├── storage/   ← GORM models + migrations
├── ui/        ← SPA screens
└── config/    ← severity-mapping.json, permissions.json
```

**သတိပြုစရာ**: Artifact က User ရဲ့ မူလ folder list ကို **တိုက်ရိုက် Copy မလုပ်ဘဲ** `worker/` ကို "Senior-level Recommendation" အဖြစ် ထပ်ဖြည့်ပြီး၊ ဘာကြောင့်ဖြည့်ရလဲ (Scan/Fix job ကို API process ထဲက ခွဲထုတ်ဖို့) ကိုပါ prose ထဲမှာ ရှင်းပြထားသည် — ဒါက **"Template ကို Blindly မလိုက်ဘဲ Justify ပြီးမှ Diverge"** ဆိုတဲ့ pattern ဖြစ်သည်။

## 1.4 Planning Flow

Section အစီအစဉ်ကို "abstraction level" အလိုက် ခွဲကြည့်ရင်:

```
01 System Architecture   ← WHY (system-level decisions, scalability posture)
02 Workflow               ← HOW (sequence + state machine)
03 API Design             ← WHAT interface (contract)
04 Database Design        ← WHAT data
05 UI Design               ← WHAT screen (most stakeholder-visible)
06 Component Responsibility ← WHO owns what (synthesis of 01-05)
07 Risks                   ← WHAT could go wrong (reviews everything before it)
08 Future Enhancements     ← WHAT's next (explicitly out of v1 scope)
```

ဒါက "Abstract → Concrete → Cross-cutting → Forward-looking" ဆိုတဲ့ 4-phase flow ဖြစ်ပြီး — **Risk section က နောက်ဆုံးအနီးမှာ ရှိနေတာ မတော်တဆ မဟုတ်ပါ**: အရာအားလုံး Design ပြီးမှသာ Risk ကို comprehensive စွာ ဖော်ထုတ်နိုင်ပါတယ်။

## 1.5 File Dependency (Artifact ကိုယ်တိုင်ရဲ့ Internal Reference Pattern)

Artifact ထဲက cross-reference အားလုံးကို ခြေရာခံကြည့်ရင်:

```
Workflow          → references → Component Responsibilities (§Security, for approval gate detail)
API Design        → references → Component Responsibilities (§Security, for role enforcement)
System Architecture → references → Risks (§Concurrency, for per-repo lock detail)
Component Resp.   → references → System Architecture (§Scalability, justifying worker/ addition)
```

**Pattern**: Dependency က တစ်ဖက်သတ် (strictly forward) မဟုတ်ဘဲ **bidirectional cross-reference** ဖြစ်သည် — နောက်ပိုင်း Section က ရှေ့ပိုင်းကို confirm ရန် ညွှန်း၊ ရှေ့ပိုင်း Section ကလည်း detail အပြည့်ကို နောက်ပိုင်းသို့ defer လုပ်ရန် ညွှန်း။ ဒါကို `plans/` files တွေမှာ replicate လုပ်ရန် — file တစ်ခုချင်းစီရဲ့ "Dependency on previous files" ကို **တစ်ဖက်သတ် (reads-from) attribute** အဖြစ်သာ Phase 2 table မှာ သတ်မှတ်ပေးမည် (documentation suite အသစ်တစ်ခုအတွက် forward-reference အများကြီး စဉ်းစားရင် ရေးရခက်၊ ဖတ်ရခက်လာနိုင်လို့)။

---

# Phase 2 — Project Planning Structure (`plans/`)

| # | File | Purpose | Why this file exists | What should be documented | Expected Sections | Dependency (reads from) | Output of the document | Maintainer |
|---|------|---------|----------------------|---------------------------|--------------------|--------------------------|--------------------------|------------|
| 00 | `00-project-overview.md` | Whole planning suite ရဲ့ entry point / index | Docs 16 ခုကို အသစ်ဝင်လာသူ (BA/Dev/QA) တစ်ယောက်က တစ်ခါတည်း orient ဖြစ်နိုင်ရန် — Artifact ရဲ့ Header + Nav Rail ရဲ့ .md equivalent | Feature name, elevator pitch, stakeholder list, current phase/status, document index (links) | Objective, Stakeholders, Status, Document Index, How to read this suite | မရှိ (root document) | Reader ရှေ့ဆက်ဖတ်ရမည့် Map | PM (owner) + BA (scope blurb) |
| 01 | `01-business-requirement.md` | Raw stakeholder ask ကို solutioning မလုပ်ခင် capture | Downstream doc (Workflow, PRD, API) အားလုံးက ဒီ file ကနေ trace ဖြစ်ရမည် — မဟုတ်ရင် Technical doc တွေ Business Intent ကနေ drift ဖြစ်တတ် | Problem statement, business goal, requestor, success metric, high-level constraint | Background, Problem Statement, Business Goal, Requestor, Success Criteria, Constraints | 00 | "ဘာကြောင့် ဒီ Project လုပ်ရလဲ" ရဲ့ Single Source of Truth | BA (primary), PM review |
| 02 | `02-current-workflow.md` | As-is state ကို Proposed Workflow မရေးခင် document | 03 ကို design ဖို့ 02 (today ဘာဖြစ်နေလဲ) အရင်သိရမည် — dashboard.md ရဲ့ "CLI ကနေ Run နေတဲ့အစား" framing ဟာ ဒီ file အနေနဲ့ formalize ဖြစ်သင့် | Current step-by-step process, tools involved, manual pain points, time/effort cost | Current Steps, Actors Involved, Pain Points, Cost of Current Process | 01 | Baseline — Proposed Workflow ကို ဒီနှင့် Compare လုပ်ရန် | BA + Developer (technical accuracy confirm) |
| 03 | `03-proposed-workflow.md` | User journey အသစ် (Scan→Plan→Fix→Rescan→History) | Artifact ရဲ့ "Workflow" Section ရဲ့ တိုက်ရိုက် precedent — API/DB/UI Design အားလုံးက ဒီ Flow ကို implement လုပ်ခြင်းသာ ဖြစ်သင့် | User journey steps, sequence diagram, state machine (Issue lifecycle), actor/role list | User Journey, Sequence Diagram, State Machine, Actors & Roles | 02 (delta), 01 (goal alignment) | Master reference — API/UI/DB Decision အားလုံး ဒီကို trace ရမည် | BA (flow owner) + PM approve, Architect feasibility review |
| 04 | `04-prd.md` | Business Requirement + Workflow ကို Formal Enterprise PRD (FR-1.., NFR, GWT-AC) အဖြစ် consolidate | Business/BA world နှင့် Technical/Dev/QA world ကြား Contract Bridge — FR numbering + GWT AC က QA testable, Dev unambiguous ဖြစ်စေ | (21-section PRD structure အပြည့် — Overview...Future Enhancements) | *(User သတ်မှတ်ထားသော 21 Section structure အတိုင်း)* | 01, 02, 03 | Dev/QA/PM sign-off ရမည့် Single Enterprise Document | BA (author) + PM (approver) + Dev/QA (reviewer) |
| 05 | `05-ui-design.md` | Screen-level Wireframe (Dashboard, Issue List, Plan Viewer, Fix Progress, History) | Screen က Stakeholder အများဆုံး မြင်ရမည့် Artifact — Backend တည်ဆောက်ပြီးမှ ပြင်ရင် Cost ကြီးလို့ Sign-off ဒီအဆင့်မှာ လိုအပ် | Screen inventory, per-screen wireframe + field list, state-ownership table | Screen Flow, Per-Screen Wireframe, State Management Table | 03 (journey→screen), 04 (FR→field/action) | Frontend Dev Build Spec + QA UI Test Checklist source | Developer (Frontend) + BA review + PM visual sign-off |
| 06 | `06-api-design.md` | REST Endpoint Contract (Method/Path/Purpose/Role table) | Backend/Frontend Team parallel work လုပ်ဖို့ Frozen Contract လိုအပ်; QA API Test Surface ဖြစ် | Endpoint summary table, role/permission per endpoint, versioning note | Endpoint Summary, Auth/Role Notes, Versioning | 03 (steps→endpoints), 07 (entity exposure), 05 (UI needs) | Backend+Frontend Contract + Integration Test input | Developer (Backend, primary) + BA/QA review |
| 07 | `07-database-design.md` | Entity/Schema Design (ER Diagram + Field table) | Schema Decision က Post-launch reverse လုပ်ရခက်ဆုံး (Migration/Data-loss risk) — Code မရေးခင် Review ရမည် | ER diagram, entity field table, design rationale (e.g. fingerprint field, junction table) | ER Diagram, Entity Field Tables, Design Rationale | 04 (FR→data need), 03 (state→status field) | Dev Migration/Schema Spec + DBA Review Artifact | Developer (Backend) + DBA/Architect review |
| 08 | `08-validation-rules.md` | Field-level + Business-rule Validation Spec | Validation Logic ကို API/DB/UI Doc သုံးခုမှာ ကွဲကွဲနေရင် "ဘယ် Layer က Validate လုပ်ရမလဲ" Ambiguity ဖြစ် — Centralize လုပ်ခြင်းက ဒီ Repo ရဲ့ Config-driven Validation Convention (`ticket_create_validataion_required_fields.json` Pattern) နှင့် ကိုက်ညီ | Required/optional field, format rule, cross-field rule, per-endpoint validation table | Field Validation Table, Cross-field/Business Validation, Config vs Code-driven note | 06, 07 | QA Edge-case Test Input + Dev Implementation Checklist | BA (rule owner) + QA (test-case) + Developer (implement) |
| 09 | `09-error-handling.md` | Error Taxonomy + HTTP Status Mapping | Consistent Error Contract — API consumer + Audit/Logging (`utils.RestErr` Pattern) နှစ်ဖက်စလုံးအတွက် အရေးကြီး | Error code table, HTTP status per scenario, user-facing vs log-only distinction | Error Catalog Table, Logging Requirements | 06 (endpoints), 08 (validation failure) | Dev Implementation Reference + QA Negative-Test Source | Developer (primary) + QA (completeness review) |
| 10 | `10-business-rules.md` | Field-validation မဟုတ်သော Domain/Business Logic Rule (e.g. "Branch တစ်ခုမှာ Active Fix တစ်ခုတည်း") | ဒီလို Rule တွေက FR List ထဲမှာ မှတ်မိအောင် သီးခြားမရေးရင် Implementation အချိန်မှာ Silent Drop ဖြစ်တတ် | Rule list + rationale + exception + owner | Business Rule Catalog (ID/Rule/Rationale/Exception), Rule-to-FR Traceability | 04 (FR elaborate), 01 (goal trace) | Dev "Must-not-violate" Checklist + QA Business-scenario Test source | BA (owner) + PM (exception approve) |
| 11 | `11-sequence-diagram.md` | Flow အသီးသီးအတွက် Technical-level Sequence Diagram (03 ထက် Fine-grained) | Service-to-service Call Order, Sync/Async Boundary (202 Accepted + Poll) ကို Component Design မလုပ်ခင် သတ်မှတ်ရမည် | Per-flow sequence diagram, sync/async boundary note | Per-Flow Sequence Diagram + Notes | 03 (journey), 06 (endpoints), 07 (DB writes) | Architecture Reference + Integration Test Scenario source | Developer/Architect |
| 12 | `12-component-design.md` | Service/Component Responsibility + Folder Structure | Developer အများ Parallel စ Code မရေးခင် Single-Responsibility Boundary သဘောတူထားရမည် — Ownership Overlap ကို ကာကွယ် | Service breakdown table, security-control-per-component table, folder tree | Service Breakdown, Security Controls Mapping, Folder Structure | 11, 07, 06 | Dev Module/Package Boundary Spec (Folder Structure ကိုယ်တိုင်) | Developer/Architect (primary) + PM ownership review |
| 13 | `13-test-plan.md` | QA Test Strategy — Happy/Error/Edge/Boundary (GWT format) | Code မရေးခင် QA Bar ကို သတ်မှတ်ထား (TDD-adjacent) — Post-hoc Improvise မဖြစ်စေရန် | Per-FR test scenario matrix (GWT), regression checklist, NFR test note | Test Scenario Matrix, Regression Checklist, NFR Test Notes | 04 (FR/AC), 08 (edge case), 09 (error case) | QA Execution Checklist + 04's AC Coverage Completeness Check | QA (primary) + BA/Dev review |
| 14 | `14-risk-analysis.md` | Risk Register (Category/Risk/Mitigation) | Risk ကို Prose ထဲ implicit ထားမည့်အစား Explicit Table ဖြင့် Reviewable/Trackable ဖြစ်စေရန် | Risk table (category, risk, mitigation), Open Risks needing decision | Risk Register Table, Open Risks | **ALL prior files (01-13)** | PM Go/No-go Sign-off Input + Ongoing Risk-tracking Artifact | Architect/Developer (identify) + PM (accept/track) |
| 15 | `15-implementation-plan.md` | Coding Roadmap — Task Breakdown/Sequencing/Milestone | "Design" ဆုံးပြီး "Build" စတဲ့ Boundary — ဒါကမှ Actual Ticket/Code Reference ခွင့်ပြု | Task list per component, milestones, task dependency, effort estimate | Task Breakdown (by Component), Milestones, Sequencing, Definition of Done | 12 (task boundary), 13 (DoD), 14 (risk-mitigating task) | Sprint/Ticket Backlog Seed (Jira/Redmine တင်ရန်) | Developer (primary) + PM (milestone owner) |
| 16 | `16-review-checklist.md` | Final Sign-off Gate — Design Phase ရဲ့ Commitment အားလုံး Honor ဖြစ်မဖြစ် Checklist | Enterprise Governance — Artifact ကိုယ်တိုင် "System Design Review · Draft" လို့ label တပ်ထားသလိုပဲ, ဒီ file ကတော့ Review ကို Repeatable/Formal ဖြစ်စေခြင်း | Checklist item per 01-15 (e.g. "04 ရဲ့ FR အားလုံး 13 မှာ Test Coverage ရှိ/မရှိ"), Sign-off table | Pre-Implementation Checklist, Post-Implementation Checklist, Sign-off Table | **ALL (00-15)** | Governance Record / Design-phase Audit Trail | PM (gate-keeper) + BA/Dev/QA (sign-off contributor) |

---

# Phase 3 — Execution Order (Step-by-Step Flow)

## 3.1 Sequential vs Parallel Groups

```
GROUP A (Strictly Sequential — တစ်ခုပြီးမှ တစ်ခု)
  00 (skeleton) → 01 → 02 → 03 → 04
       ↑                              │
       └──── (content ပြီးမှ update) ──┘
       00 ကို "skeleton" အနေနဲ့ ပထမဆုံး ဖန်တီး (index placeholder),
       ဒါပေမယ့် content အပြည့် ရေးသားခြင်းကို GROUP D ပြီးမှသာ Finalize

GROUP B (04 ပြီးမှ Parallel လုပ်နိုင် — Design-detail level)
  04 ──┬──→ 05-ui-design.md          ┐
       ├──→ 06-api-design.md         ├─ (3 files ကို Developer/BA သီးခြား
       └──→ 07-database-design.md    ┘   Parallel ရေးနိုင်, တစ်ယောက်ချင်း Owner)

GROUP C (06,07 ပြီးမှ Parallel — Boundary-detail level)
  06,07 ──┬──→ 08-validation-rules.md   ┐
          ├──→ 09-error-handling.md     ├─ (Parallel)
          └──→ 10-business-rules.md     ┘

GROUP D (Technical synthesis — Sequential, Architect-led)
  05,06,07,08,09,10 → 11-sequence-diagram.md → 12-component-design.md

GROUP E (12 ပြီးမှ Parallel)
  12 ──┬──→ 13-test-plan.md         ┐
       └──→ 14-risk-analysis.md      ┘ (13 ပြီးမှ 14 review လုပ်တာ သင့်လျော်—
                                        14 က ALL prior ကို review ဖြစ်လို့)

GROUP F (Terminal — Sequential)
  13,14 → 15-implementation-plan.md → 16-review-checklist.md (Pre-Impl gate)
                                            │
                                       ★ DEVELOPER CODE စတင် ★
                                            │
                                       16-review-checklist.md (Post-Impl gate — ပြန်ဖွင့်ပြီး update)
```

## 3.2 "Developer ဘယ်အချိန်မှ Code စရေးသင့်လဲ" — Explicit Answer

| အဆင့် | Gate | Developer လုပ်ဆောင်ချက် |
|---|---|---|
| 04 (PRD) approve မဖြစ်ခင် | 🔴 **Code လုံးဝ မစရ** | Design doc ဖတ်ရုံ/Feasibility feedback ပေးရုံသာ |
| 05-12 (Design docs) ရေးနေစဉ် | 🔴 **Code မစရ** | Architect/Senior Developer အနေနဲ့ 11-12 ကို Author/Review လုပ်ရန်သာ |
| 15 (`implementation-plan.md`) ပြီးပြီး၊ 16 (`review-checklist.md`) ရဲ့ **Pre-Implementation Checklist** ✅ ဖြစ်ပြီးမှ | 🟢 **Code စတင်ခွင့်ရ** | Task breakdown (15) ကို Ticket/Sprint ထဲ ထည့်ပြီး Coding စတင် |
| Code ပြီးနောက် | ⬜ | 16 ကို ပြန်ဖွင့်ပြီး **Post-Implementation Checklist** ဖြည့် (Sign-off ပြည့်စုံမှ Feature ကို Done လို့ ယူဆ) |

**အကျဉ်းချုပ်**: Developer ရဲ့ **ပထမဆုံး Code Line** ကို `15-implementation-plan.md` ပြီးပြီး AND `16-review-checklist.md`'s Pre-Implementation section ✅ ဖြစ်မှသာ ရေးသင့်သည် — ဒီအလျင် ရေးရင် Requirement/Design ပြောင်းလဲမှုကြောင့် Code ကို ပြန်ရေးရမည့် Risk (Rework Cost) မြင့်တက်နိုင်ပါသည်။

---

**Rule Compliance confirm**: Code မရေးထားပါ။ Architecture ပြောင်းမထားပါ (Artifact ရဲ့ Component/Folder Structure ကို Reference အနေနဲ့သာ ကိုးကား)။ Requirement Content (FR/AC value) မရေးထားပါ — File တစ်ခုချင်းစီရဲ့ **Meta-description** ကိုသာ ဖော်ပြထားပါသည်။ PRD Content မရေးထားပါ။
