# Project AI Rules

- Before starting work, read `docs/AI_PROJECT_STARTER/PROJECT_AI_PLAYBOOK.md`.
- Treat that file as the default AI collaboration rules for this repository.
- Then read `docs/AI_PROJECT_STARTER/CODEX_RULES.md` as the Codex-specific compact rule layer.
- Then read `docs/PROJECT_LOCAL_CONTEXT.md` as the WeLink-specific supplement.
- Before modifying, adding, renaming, or deleting files in Git, first run `$git-remote-sync-guard` or an equivalent remote sync check.
- If the repo is `behind`, `diverged`, `no-upstream`, `no-remote`, or remote fetch fails, stop and tell the user before editing.
- Do not auto `pull`, `rebase`, `merge`, or `push` as part of the preflight check.
- Prefer non-invasive exploration first, then implement the minimum compatible change.
- Treat the root `README.md` as the product entry and `docs/README.md` as the docs index.
- Do not create a parallel docs / rules system if the current docs can absorb the change.
- When changing startup, path, indexing, MCP, or relation-analysis behavior, update the linked docs in the same task.
