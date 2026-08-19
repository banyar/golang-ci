# Git init + push to existing GitHub repo (banyar/golang-ci)

## Context
The local project directory (`/home/ubuntu/FrontiirProjects/RT/golangci`) is not yet a git
repository. The target remote, `https://github.com/banyar/golang-ci`, already has a `main`
branch with one prior commit (`890d9a1 first golandci development`) — an earlier snapshot of
this same project (matching `rule-my.md`, older `Makefile`/`cmd/golangci-report.sh`, plus two
files not present locally: `bak/golangci_old.yml`, `bak/.golangci.yml`, and `lint-configs.txt`).
The local directory is a newer, expanded version (adds `backend/`, `ui/`, `docs/`, `plans/`,
etc.). The goal is to connect local work to the remote **without discarding that existing
history** — user confirmed via AskUserQuestion: "Preserve history, add local as new commit."

SSH auth to GitHub as user `banyar` already verified working (`ssh -T git@github.com`
succeeded). No `gh` CLI auth and no HTTPS credential helper found, so the remote will use the
SSH URL (`git@github.com:banyar/golang-ci.git`) rather than HTTPS.

Secrets check done: `.env` is already listed in `.gitignore`; grep across `backend/`/`cmd/` for
password/secret/token/apikey patterns found only code identifiers (env var names, struct
fields), no literal credential values. `dump.rdb` (a Redis RDB dump — runtime artifact, not
source) is **not** currently in `.gitignore` and will be added before the first commit.

## Steps

1. **Init repo on `main`, matching remote's branch name**
   `git init -b main` in the project root. (git 2.43.0 installed — supports `-b`.)

2. **Add remote via SSH** (SSH already verified; no HTTPS token available)
   `git remote add origin git@github.com:banyar/golang-ci.git`

3. **Fetch existing remote history**
   `git fetch origin`

4. **Attach local `main` to the remote's history without touching the working tree**
   `git reset origin/main` (mixed reset — moves the `main` ref and index to match
   `origin/main`'s commit; working tree files are left untouched). This makes the upcoming
   commit a normal child of the existing remote commit — no force push required later.

   After this, `git status` is expected to show:
   - `bak/golangci_old.yml`, `bak/.golangci.yml`, `lint-configs.txt` as **deleted** (present in
     the old remote commit, absent locally)
   - `Makefile`, `.gitignore`, `cmd/golangci-report.sh` as **modified** (local versions differ)
   - Everything else new locally (`backend/`, `ui/`, `docs/`, `plans/`, etc.) as **untracked**

5. **Update `.gitignore`**: add a `dump.rdb` line (Redis runtime dump, not source). Leave the
   rest of the existing `.gitignore` as-is — it already excludes `.env`, the lint binary, and
   lint cache; `ui/` has its own nested `.gitignore` that already excludes `node_modules`/`dist`.

6. **Stage everything**: `git add -A` (this will include the deletions of the three
   remote-only files from step 4 — flagged to user for explicit confirmation in step 7, not
   silently dropped).

7. **Present the draft for review before committing** (required — no commit finalizes without
   this): show `git status --short` (full file list: additions/modifications/deletions) and
   the exact `.gitignore` diff, and the proposed commit message. Confirm with the user,
   specifically confirming they're OK with `bak/*` and `lint-configs.txt` being removed from
   the repo (they're absent from the current working tree, so keeping them would mean
   restoring files the user doesn't have locally).

8. **Commit** (conventional commit format, subject ≤50 chars, verified with
   `echo -n "..." | wc -c`), e.g.:
   `chore(golangci): sync updated local workspace`

9. **Push**: `git push -u origin main`. Expected to be a plain fast-forward push (local
   `main`'s parent is `origin/main`'s current tip) — no force needed.

10. **Verify**: `git log --oneline` (expect 2 commits: the original + the new one),
    `git status` (clean), and `git ls-remote git@github.com:banyar/golang-ci.git` to confirm
    the remote tip now matches the new local commit hash.

## Explanation to user
Throughout execution, explain each step in Burmese in the chat (not in this plan file), since
that's how the user asked to communicate about this task.
