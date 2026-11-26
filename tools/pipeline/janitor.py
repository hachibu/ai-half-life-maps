"""Hourly housekeeping (run by the pipeline workflow's schedule):

  - expire claims older than STALE_CLAIM_HOURS that have no open PR (the task goes back to the pool)
  - rebuild ledger.sqlite and LEDGER.md on the ledger branch
"""
import datetime as dt
import os
from pathlib import Path

from tools.pipeline import gh, ledger
from tools.pipeline import model as M
from tools.pipeline.gates import STALE_CLAIM_HOURS


def expire_claims(now=None):
    now = now or dt.datetime.now(dt.timezone.utc)
    prs = gh.pulls("open")
    linked = {n for p in prs for n in M.closes(p.get("body"))}
    for i in gh.issues("open", "task"):
        t = M.parse_task(i)
        if not t["claimed"] or t["number"] in linked:
            continue
        ev = [e for e in gh.label_events(t["number"]) if e["event"] == "labeled" and e["label"]["name"].startswith("claimed:")]
        if not ev:
            continue
        when = dt.datetime.fromisoformat(ev[-1]["created_at"].replace("Z", "+00:00"))
        hours = (now - when).total_seconds() / 3600
        if hours > STALE_CLAIM_HOURS:
            for a in t["claimed"]:
                gh.api("repos/{repo}/issues/%d/labels/claimed:%s" % (t["number"], a), "DELETE")
            gh.api("repos/{repo}/issues/%d/comments" % t["number"], "POST", {"body":
                   "Claim expired after %.0f h with no pull request; the task is free again." % hours})
            ledger.record("claim-expired", "issue#%d" % t["number"], ",".join(t["claimed"]), False, "", {"hours": round(hours)})
            print("expired claim on #%d" % t["number"])


def render():
    out = os.environ.get("LEDGER_OUT")
    if not out:
        return
    con = ledger.build(out, Path(out) / "ledger.sqlite")
    con.close()
    ledger.render(Path(out) / "ledger.sqlite", Path(out) / "LEDGER.md")


if __name__ == "__main__":
    expire_claims()
    render()
