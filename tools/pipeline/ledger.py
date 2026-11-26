"""The ledger: an append-only record of everything the pipeline decided.

Storage is conflict-free by design: each event is its own small JSON file under events/ on the `ledger`
branch (only CI writes there), so concurrent writers never touch the same file. SQLite is the query
engine, rebuilt from the events whenever you need it:

    python -m tools.pipeline.ledger fetch            # pull the ledger branch into build/ledger/
    python -m tools.pipeline.ledger build            # events -> build/ledger/ledger.sqlite
    python -m tools.pipeline.ledger query "SELECT * FROM latest WHERE state != 'ok'"
    python -m tools.pipeline.ledger render           # -> build/ledger/LEDGER.md

In CI, set LEDGER_OUT to a checkout of the ledger branch and record()/record_gates() write event files there.
"""
import datetime as dt
import json
import os
import re
import secrets
import sqlite3
import subprocess
import sys
from pathlib import Path

LOCAL = Path("build/ledger")

SCHEMA = """
CREATE TABLE events (ts TEXT, kind TEXT, ref TEXT, agent TEXT, gate TEXT, state TEXT, sha TEXT, detail TEXT);
CREATE INDEX ev_ref ON events(ref, gate, ts);
CREATE VIEW latest AS
  SELECT e.* FROM events e
  JOIN (SELECT ref, gate, MAX(ts) ts FROM events WHERE kind = 'gate' GROUP BY ref, gate) m
    ON e.ref = m.ref AND e.gate = m.gate AND e.ts = m.ts WHERE e.kind = 'gate';
CREATE VIEW merges AS SELECT ts, ref, agent, sha, json_extract(detail, '$.title') title FROM events WHERE kind = 'merge';
CREATE VIEW failures AS
  SELECT agent, gate, COUNT(*) n FROM events WHERE kind = 'gate' AND state = 'fail' GROUP BY agent, gate;
CREATE VIEW incidents AS SELECT * FROM events WHERE kind IN ('incident', 'health');
"""


def _now():
    return dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%fZ")


def _out():
    o = os.environ.get("LEDGER_OUT")
    return Path(o) if o else None


def _event(kind, ref, agent, gate, state, sha, detail):
    return dict(ts=_now(), kind=kind, ref=ref, agent=agent or "-", gate=gate or "", state=state or "",
                sha=sha or "", detail=detail or {}, run=os.environ.get("GITHUB_RUN_ID", "local"))


def _write(ev):
    out = _out()
    if out is None:
        print("ledger (dry):", json.dumps(ev)); return
    d = out / "events"; d.mkdir(parents=True, exist_ok=True)
    name = "%s-%s-%s-%s-%s.json" % (ev["ts"].replace(":", ""), re.sub(r"\W", "", ev["ref"]), ev["kind"],
                                    re.sub(r"\W", "", ev["gate"]) or "x", secrets.token_hex(3))
    (d / name).write_text(json.dumps(ev, sort_keys=True) + "\n")


def _latest_states(ref):
    out = _out()
    if out is None or not (out / "events").is_dir():
        return {}
    best = {}
    for p in sorted((out / "events").glob("*-%s-gate-*.json" % re.sub(r"\W", "", ref))):
        ev = json.loads(p.read_text())
        best[ev["gate"]] = (ev["state"], ev["sha"], ev["detail"].get("reasons"))
    return best


def record(kind, ref, agent, ok, sha="", detail=None):
    _write(_event(kind, ref, agent, kind, "ok" if ok else "fail", sha, detail))


def record_gates(ref, agent, sha, results):
    """One event per gate whose (state, sha, reasons) changed since the last recorded event."""
    last = _latest_states(ref)
    for r in results:
        cur = (r.state, sha, r.reasons)
        if last.get(r.gate) != cur:
            _write(_event("gate", ref, agent, r.gate, r.state, sha, {"reasons": r.reasons}))


# ------------------------------------------------------------------ local tools
def fetch():
    LOCAL.mkdir(parents=True, exist_ok=True)
    if subprocess.run(["git", "fetch", "-q", "origin", "ledger"], capture_output=True).returncode != 0:
        print("no ledger branch yet (it appears after the first pipeline run)"); return False
    tar = subprocess.run(["git", "archive", "FETCH_HEAD"], capture_output=True).stdout
    subprocess.run(["tar", "-x", "-C", str(LOCAL)], input=tar)
    print("ledger: %d events" % len(list((LOCAL / "events").glob("*.json"))))
    return True


def build(src=None, db=None):
    src = Path(src or LOCAL); db = Path(db or LOCAL / "ledger.sqlite")
    db.parent.mkdir(parents=True, exist_ok=True)
    if db.exists(): db.unlink()
    con = sqlite3.connect(db); con.executescript(SCHEMA)
    rows = []
    for p in sorted((src / "events").glob("*.json")) if (src / "events").is_dir() else []:
        e = json.loads(p.read_text())
        rows.append((e["ts"], e["kind"], e["ref"], e["agent"], e["gate"], e["state"], e["sha"], json.dumps(e["detail"])))
    con.executemany("INSERT INTO events VALUES (?,?,?,?,?,?,?,?)", rows); con.commit()
    print("built %s (%d events)" % (db, len(rows)))
    return con


def connect(db=None):
    db = Path(db or LOCAL / "ledger.sqlite")
    return sqlite3.connect(db) if db.exists() else build()


def query(sql, db=None):
    con = connect(db); cur = con.execute(sql)
    cols = [c[0] for c in cur.description]
    print("\t".join(cols))
    for row in cur.fetchall():
        print("\t".join(str(x) for x in row))


def render(db=None, out=None):
    con = connect(db)
    q = lambda s: con.execute(s).fetchall()
    L = ["# Ledger", "", "Generated from the event files on this branch by `tools/pipeline/ledger.py render`. "
         "Query it yourself: `make ledger` then `python -m tools.pipeline.ledger query \"...\"`.", ""]
    L += ["## Open gate problems", "", "| item | gate | state | sha | since | reasons |", "|---|---|---|---|---|---|"]
    merged = {r[0] for r in q("SELECT ref FROM merges")}
    for ref, gate, state, sha, ts, detail in q("SELECT ref, gate, state, sha, ts, detail FROM latest WHERE state != 'ok' ORDER BY ts DESC"):
        if ref in merged: continue
        reasons = "; ".join(json.loads(detail).get("reasons") or [])[:160].replace("|", "/")
        L.append("| %s | %s | %s | %s | %s | %s |" % (ref, gate, state, sha[:7], ts[:16], reasons))
    L += ["", "## Recent merges", "", "| when | item | agent | title |", "|---|---|---|---|"]
    for ts, ref, agent, sha, title in q("SELECT * FROM merges ORDER BY ts DESC LIMIT 30"):
        L.append("| %s | %s | %s | %s |" % (ts[:16], ref, agent, (title or "").replace("|", "/")))
    L += ["", "## Gate failures by agent", "", "| agent | gate | failures |", "|---|---|---|"]
    for a, g, n in q("SELECT * FROM failures ORDER BY n DESC"):
        L.append("| %s | %s | %d |" % (a, g, n))
    L += ["", "## Health of main", "", "| when | result | sha | detail |", "|---|---|---|---|"]
    for ts, kind, ref, agent, gate, state, sha, detail in q("SELECT * FROM incidents ORDER BY ts DESC LIMIT 20"):
        L.append("| %s | %s | %s | %s |" % (ts[:16], state, sha[:7], detail[:120].replace("|", "/")))
    out = Path(out or LOCAL / "LEDGER.md"); out.write_text("\n".join(L) + "\n")
    print("wrote", out)


if __name__ == "__main__":
    cmd, *rest = sys.argv[1:] or ["build"]
    {"fetch": fetch, "build": build, "query": query, "render": render}[cmd](*rest)
