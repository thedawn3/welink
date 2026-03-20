# Project AI Rules

- Project-level `AGENTS.md` is the primary entry for AI collaboration in this repository and is the first rule layer to read.
- Read in order: `AGENTS.md` -> root `README.md` (plus `docs/README.md` as docs index) -> `docs/PROJECT_LOCAL_CONTEXT.md` -> `docs/AI_PROJECT_STARTER/PROJECT_AI_PLAYBOOK.md` -> `docs/AI_PROJECT_STARTER/CODEX_RULES.md`.
- Treat starter files as baseline supplements; repository-level rules and local context remain authoritative.
- If `.trae/rules/` exists, treat it as an extra enforcement layer rather than the entry layer, and keep it aligned with project-level rules.
- Before modifying, adding, renaming, or deleting files in Git, first run `$git-remote-sync-guard` or an equivalent remote sync check.
- If the repo is `behind`, `diverged`, `no-upstream`, `no-remote`, or remote fetch fails, stop and tell the user before editing.
- Do not auto `pull`, `rebase`, `merge`, or `push` as part of the preflight check.
- Prefer non-invasive exploration first, then implement the minimum compatible change.
- Treat the root `README.md` as the product entry and `docs/README.md` as the docs index.
- Do not create a parallel docs / rules system if the current docs can absorb the change.
- When changing startup, path, indexing, MCP, or relation-analysis behavior, update the linked docs in the same task.
