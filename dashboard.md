golangci-lint အတွက် Web UI Dashboard တစ်ခု တည်ဆောက်ချင်ပါတယ်။

အခုအဆင့်မှာ Code ကို လုံးဝ မရေးပါနဲ့။
Implementation လည်း မလုပ်ပါနဲ့။

Senior Software Architect အနေနဲ့ Architecture Design နဲ့ Feature Flow ကိုပဲ ရှင်းပြပေးပါ။

## Feature Overview

CLI ကနေ golangci-lint run နေတဲ့အစား Web UI ကနေ Control လုပ်ချင်ပါတယ်။

Workflow က ဒီလိုဖြစ်ရမယ်။

1. User က Web UI ကို ဖွင့်မယ်။

2. "Scan Lint" Button ကို နှိပ်မယ်။

3. Backend က golangci-lint ကို Run ပြီး Result တွေကို Parse လုပ်မယ်။

4. UI မှာ Result List ကို Table အနေနဲ့ ပြမယ်။

Table Column ဥပမာ

- Checkbox
- File
- Line
- Column
- Linter Name
- Severity
- Rule
- Message
- Suggested Fix
- Status

5. User က Row တစ်ခု သို့မဟုတ် Multiple Rows ကို Checkbox နဲ့ Select လုပ်နိုင်ရမယ်။

6. "View Plan" Button ကို နှိပ်ရင်

Selected Lint Issue တွေအတွက်

AI Fix Plan

ကို Web UI ထဲမှာ Preview ပြပေးရမယ်။

Plan မှာ

- Root Cause
- Current Code Behavior
- Recommended Fix
- Risk
- Breaking Change
- Files Impact
- Test Plan (Given/When/Then)

တွေ ပါရမယ်။

7. User က Plan ကို Review လုပ်ပြီး

"Apply Fix"

Button ကို နှိပ်ရင်

AI က Code Fix လုပ်မယ်။

8. Fix ပြီးရင်

Automatic Re-run golangci-lint

လုပ်ပြီး

Passed / Failed

ကို ပြပေးရမယ်။

9. History Page မှာ

- Scan History
- Applied Fixes
- Rollback History

ကို သိမ်းထားချင်ပါတယ်။

---

## အခုအဆင့်မှာ လိုချင်တာ

Code မရေးပါနဲ့။

အောက်ပါအချက်တွေကိုပဲ Design Review လုပ်ပေးပါ။

### 1. Overall Architecture

- Frontend
- Backend
- AI Layer
- golangci-lint Runner
- Storage

---

### 2. User Workflow

Step by Step Sequence Diagram

---

### 3. Backend Components

ဘယ် Service တွေ ခွဲသင့်လဲ

ဥပမာ

- Scan Service
- Parser Service
- Plan Service
- Fix Service
- History Service
- Rollback Service

---

### 4. API Design

လိုအပ်မယ့် REST APIs

ဥပမာ

POST /scan

GET /issues

POST /plan

POST /fix

POST /rollback

GET /history

တို့ကိုသာ List ထုတ်ပေးပါ။

Implementation မရေးပါနဲ့။

---

### 5. Database Design

ဘာ Collection/Table တွေလိုမလဲ

ဥပမာ

LintScan

LintIssue

FixPlan

FixHistory

Configuration

တို့ကို Design ပေးပါ။

---

### 6. UI Design

Screen Flow

Dashboard

Issue List

Plan Viewer

Fix Progress

History

တို့ကို Wireframe Level နဲ့ ရှင်းပြပေးပါ။

---

### 7. State Management

Issue Selection

Plan Loading

Fix Progress

Rollback

တို့ကို ဘယ်လို State Flow နဲ့ စီမံသင့်လဲ။

---

### 8. Security

AI Fix မလုပ်ခင်

Confirmation

Approval

Permission

Audit Log

တွေ ဘယ်လို ထည့်သင့်လဲ။

---

### 9. Scalability

Repository အကြီးကြီးတွေ

Microservice

Monorepo

Multiple Projects

တွေကို Support လုပ်နိုင်ဖို့ ဘာတွေ ထည့်စဉ်းစားသင့်လဲ။

---

### 10. Risks

Architecture Risks

Performance Risks

Concurrency Risks

AI Wrong Fix Risks

Rollback Risks

---

### 11. Recommended Folder Structure

golangci/

├── api/

├── scanner/

├── parser/

├── planner/

├── fixer/

├── history/

├── storage/

├── ui/

├── config/

စတဲ့ Folder Structure ကို အကြံပြုပေးပါ။

---

Output Format

## System Architecture

## Workflow

## API Design

## Database Design

## UI Design

## Component Responsibilities

## Risks

## Future Enhancements

Code မရေးပါနဲ့။
Architecture Design နဲ့ Recommendation ပဲ ပေးပါ။