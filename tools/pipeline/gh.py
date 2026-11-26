"""Tiny GitHub REST client on top of the `gh` CLI (REST only: no GraphQL)."""
import json
import os
import subprocess


def repo():
    if os.environ.get("GITHUB_REPOSITORY"):
        return os.environ["GITHUB_REPOSITORY"]
    url = subprocess.run(["git", "remote", "get-url", "origin"], capture_output=True, text=True).stdout.strip()
    return url.rstrip("/").removesuffix(".git").split("github.com")[-1].lstrip(":/")


def api(path, method="GET", data=None, paginate=False):
    """Call the REST API. `path` may contain {repo}. Returns parsed JSON (list for paginated calls)."""
    path = path.replace("{repo}", repo())
    cmd = ["gh", "api", "-X", method, path, "-H", "Accept: application/vnd.github+json"]
    if paginate:
        cmd += ["--paginate", "--slurp"]
    if data is not None:
        cmd += ["--input", "-"]
    out = subprocess.run(cmd, input=json.dumps(data) if data is not None else None,
                         capture_output=True, text=True)
    if out.returncode != 0:
        raise RuntimeError("gh api %s %s failed: %s" % (method, path, out.stderr.strip() or out.stdout.strip()))
    if not out.stdout.strip():
        return None
    res = json.loads(out.stdout)
    if paginate:                       # --slurp gives a list of pages
        return [x for page in res for x in (page if isinstance(page, list) else [page])]
    return res


def issues(state="open", labels=None):
    q = "repos/{repo}/issues?state=%s&per_page=100" % state + ("&labels=%s" % labels if labels else "")
    return [i for i in api(q, paginate=True) if "pull_request" not in i]


def pulls(state="open"):
    return api("repos/{repo}/pulls?state=%s&per_page=100" % state, paginate=True)


def pr_files(n):
    return api("repos/{repo}/pulls/%d/files?per_page=100" % n, paginate=True)


def comments(n):
    return api("repos/{repo}/issues/%d/comments?per_page=100" % n, paginate=True)


def reviews(n):
    return api("repos/{repo}/pulls/%d/reviews?per_page=100" % n, paginate=True)


def review_comments(n):
    return api("repos/{repo}/pulls/%d/comments?per_page=100" % n, paginate=True)


def runs_for_branch(branch, workflow="map-check.yml"):
    r = api("repos/{repo}/actions/workflows/%s/runs?branch=%s&per_page=100" % (workflow, branch))
    return r.get("workflow_runs", []) if r else []


def runs_for_sha(sha, workflow="map-check.yml"):
    r = api("repos/{repo}/actions/workflows/%s/runs?head_sha=%s&per_page=50" % (workflow, sha))
    return r.get("workflow_runs", []) if r else []


def run_jobs(run_id):
    r = api("repos/{repo}/actions/runs/%d/jobs?per_page=100" % run_id)
    return r.get("jobs", []) if r else []


def commit(sha):
    return api("repos/{repo}/commits/%s" % sha)


def label_events(n):
    return [e for e in api("repos/{repo}/issues/%d/events?per_page=100" % n, paginate=True)
            if e.get("event") in ("labeled", "unlabeled")]
