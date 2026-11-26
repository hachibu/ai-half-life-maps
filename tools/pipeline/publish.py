"""Publish an agent's finished action deterministically (used by the Codex worker; any agent may use it).

    python -m tools.pipeline.publish AGENT ACTION NUMBER BRANCH

Reads the files the agent left in .codex-out/ (see .github/codex/worker.md) and does the git/GitHub part:
commit + push + open the PR, post the review comment, or open the proposed task issues.
"""
import subprocess
import sys
from pathlib import Path

from tools.pipeline import gh
from tools.pipeline import model as M

OUT = Path(".codex-out")


def sh(*cmd):
    print("+", " ".join(cmd)); subprocess.run(cmd, check=True)


def code(agent, action, n, branch):
    sh("git", "add", "-A", "--", ".", ":!.codex-out", ":!build", ":!next.json")
    if subprocess.run(["git", "diff", "--cached", "--quiet"]).returncode == 0:
        print("no changes to publish"); return
    title = (OUT / "pr-title.txt").read_text().strip() if (OUT / "pr-title.txt").exists() else "%s: work on #%s" % (agent, n)
    sh("git", "commit", "-qm", title)
    if action == "fix-pr":
        pr = gh.api("repos/{repo}/pulls/%s" % n)
        sh("git", "push", "-q", "origin", "HEAD:refs/heads/%s" % pr["head"]["ref"]); return
    sh("git", "push", "-q", "origin", "HEAD:refs/heads/%s" % branch)
    owner = gh.repo().split("/")[0]
    if gh.api("repos/{repo}/pulls?state=open&head=%s:%s" % (owner, branch)):
        return
    body = (OUT / "pr-body.md").read_text() if (OUT / "pr-body.md").exists() else "Closes #%s\n\n(no description written)" % n
    if "#%s" % n not in body:
        body = "Closes #%s\n\n" % n + body
    pr = gh.api("repos/{repo}/pulls", "POST", {"title": title, "head": branch, "base": "main", "body": body})
    print("opened PR #%d" % pr["number"])


def review(agent, n):
    text = (OUT / "review.md").read_text()
    rv = M.parse_agent_review(text)
    head = gh.api("repos/{repo}/pulls/%s" % n)["head"]["sha"]
    if not rv or rv["reviewer"] != agent or not head.startswith(rv["sha"][:7]):
        raise SystemExit("review.md has no valid agent-review block for %s at %s" % (agent, head[:7]))
    gh.api("repos/{repo}/issues/%s/comments" % n, "POST", {"body": text})


def propose():
    for p in sorted((OUT / "issues").glob("*.md")):
        title, _, body = p.read_text().partition("\n")
        i = gh.api("repos/{repo}/issues", "POST", {"title": title.strip("# ").strip(), "body": body.strip(), "labels": ["task"]})
        print("opened task #%d" % i["number"])


if __name__ == "__main__":
    agent, action, n, branch = sys.argv[1:5]
    if action == "review": review(agent, n)
    elif action == "propose": propose()
    elif action != "idle": code(agent, action, n, branch)
