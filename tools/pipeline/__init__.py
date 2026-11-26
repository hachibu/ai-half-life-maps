"""The pipeline: deterministic gates and an append-only ledger. Agents work when a human gives them a task.

Every unit of work is a GitHub issue (a task) and moves along one graph:

    idea -> ready -> claimed -> pr -> built -> shown -> reviewed -> learned -> merged -> healthy

Each arrow is a gate: a pure function of GitHub data (no model judgement, no PR code executed).
See docs/pipeline.md for the full description.
"""
AGENTS = ("claude", "codex")
