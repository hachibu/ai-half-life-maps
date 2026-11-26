"""Parse tasks (issues), pull requests, reviews and lessons into plain data. No network here."""
import re
from pathlib import PurePosixPath

from tools.pipeline import AGENTS

PLACEHOLDER = {"", "_no response_", "none", "n/a", "-", "tbd", "todo"}
ALLOWED_ROOTS = ("maps/", "tools/", "docs/", ".github/", "Makefile", "AGENTS.md", "README.md", "requirements.txt")
PROTECTED = (".github/", "tools/pipeline/", "AGENTS.md", "Makefile", "CLAUDE.md")   # rules of the system itself
PRIORITY = {"P0": 0, "P1": 1, "P2": 2, "P3": 3}


def strip_comments(text):
    return re.sub(r"<!--.*?-->", "", text or "", flags=re.S)


def sections(body, level="###"):
    """Markdown body -> {heading lowercased: content} for headings of the given level (## or ###)."""
    out, cur = {}, None
    for line in strip_comments(body).splitlines():
        m = re.match(r"^%s\s+(.+?)\s*$" % re.escape(level), line)
        if m and not line.startswith(level + "#"):
            cur = m.group(1).strip().lower(); out[cur] = []
        elif cur is not None:
            out[cur].append(line)
    return {k: "\n".join(v).strip() for k, v in out.items()}


def filled(text):
    t = (text or "").strip()
    return t.lower() not in PLACEHOLDER


def labels(item):
    return {l["name"] if isinstance(l, dict) else l for l in item.get("labels", [])}


# ------------------------------------------------------------------ tasks
def parse_task(issue):
    s = sections(issue.get("body") or "")
    paths = [p.strip().strip("`") for p in re.split(r"[\n,]+", s.get("paths", "")) if filled(p.strip().strip("`-* "))]
    paths = [p.lstrip("-* ").strip("`") for p in paths]
    deps = [int(x) for x in re.findall(r"#(\d+)", s.get("depends on", ""))]
    prio = (re.search(r"P[0-3]", s.get("priority", "")) or re.search(r"P[0-3]", "P2")).group(0)
    lab = labels(issue)
    claimed = [a for a in AGENTS if "claimed:%s" % a in lab]
    return dict(number=issue["number"], title=issue.get("title", ""), map=s.get("map", "").strip("` "),
                paths=paths, goal=s.get("goal", ""), criteria=s.get("acceptance criteria", ""),
                deps=deps, priority=prio, labels=lab, claimed=claimed,
                incident="incident" in lab, hold="hold" in lab, state=issue.get("state", "open"),
                created=issue.get("created_at", ""))


def path_overlap(a, b):
    a, b = a.rstrip("/"), b.rstrip("/")
    return a == b or a.startswith(b + "/") or b.startswith(a + "/")


def covered(path, allowed):
    return any(path_overlap(path, p) and (path == p.rstrip("/") or path.startswith(p.rstrip("/") + "/"))
               for p in allowed)


# ------------------------------------------------------------------ pull requests
def pr_agent(pr):
    ref = pr["head"]["ref"]
    for a in AGENTS:
        if ref.startswith(a + "/"):
            return a
    return "human"


def closes(body):
    return [int(n) for n in re.findall(r"(?i)\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s+#(\d+)", body or "")]


def changed_maps(files):
    return sorted({f["filename"].split("/")[1] for f in files
                   if f["filename"].startswith("maps/") and f["filename"].count("/") >= 2
                   and not f["filename"].split("/")[1].startswith("_")})


REVIEW_RE = re.compile(r"<!--\s*agent-review\s*-->(.*)", re.S)


def parse_agent_review(body):
    """Structured review block: reviewer / verdict (approve|changes) / sha."""
    m = REVIEW_RE.search(body or "")
    if not m:
        return None
    kv = dict((k.strip().lower(), v.strip()) for k, v in re.findall(r"(?m)^\s*([A-Za-z]+)\s*:\s*(\S+)", m.group(1)))
    if not {"reviewer", "verdict", "sha"} <= kv.keys():
        return None
    return dict(reviewer=kv["reviewer"].lower(), verdict=kv["verdict"].lower(), sha=kv["sha"].lower())


def is_codex_login(login):
    return "codex" in (login or "").lower()


# ------------------------------------------------------------------ lessons
LESSON_SECTIONS = ("what happened", "why", "rule")


def parse_lesson(text):
    m = re.match(r"^---\n(.*?)\n---\n(.*)$", text or "", re.S)
    if not m:
        return None
    meta = {}
    for line in m.group(1).splitlines():
        if ":" in line:
            k, v = line.split(":", 1); meta[k.strip()] = v.strip()
    tags = [t.strip().strip("'\"") for t in meta.get("tags", "").strip("[]").split(",") if t.strip()]
    return dict(title=meta.get("title", ""), tags=tags, gate=meta.get("gate", ""), source=meta.get("source", ""),
                sections=sections(m.group(2), "##"))


def lesson_problems(text):
    L = parse_lesson(text)
    if L is None:
        return ["missing front matter (--- title/tags/gate/source ---)"]
    p = []
    if not L["title"]: p.append("no title")
    if not L["tags"]: p.append("no tags")
    for s in LESSON_SECTIONS:
        if not filled(L["sections"].get(s, "")): p.append("section '## %s' missing or empty" % s.title())
    return p


def is_lesson_path(path):
    pp = PurePosixPath(path)
    return len(pp.parts) == 3 and pp.parts[:2] == ("docs", "lessons") and pp.suffix == ".md" and pp.name != "README.md"
