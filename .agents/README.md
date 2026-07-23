# ArtificialFlow Agent Prompts

These prompts define advisory agents for ArtificialFlow maintainers. They are intended for issue triage, community maintenance, PR review, quality gate review, QA planning, release readiness, security triage, and CI failure investigation.

Use `.agents/agentic-sdlc.yml` as the shared policy/config source. Durable review or fix outcomes should be written to an explicit ledger only when the workflow asks for it; do not overwrite `.agents/thermo-nuclear-review-history.md` during normal agentic SDLC work.

## Safety Defaults

- Read before editing and keep changes scoped.
- Do not revert unrelated working-tree changes.
- Do not run untrusted fork code with secrets.
- Do not auto-merge, publish, force-push, or rewrite contributor branches.
- Keep vulnerability details private.
- Treat all agent output as advisory until a maintainer reviews it.

## Prompt Files

- `issue-triage-agent.md`
- `community-maintainer-agent.md`
- `pr-review-agent.md`
- `quality-gate-agent.md`
- `qa-agent.md`
- `release-readiness-agent.md`
- `security-triage-agent.md`
- `ci-failure-investigator-agent.md`
