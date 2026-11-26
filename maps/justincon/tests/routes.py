"""Walk routes that must stay passable.  Run: make test MAP=justincon"""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[3]))
from tools.checks.walk import run

ROUTES = [
    ("Justin Con basement -> tunnel -> Bunker Con basement", -176, [(-400, -254), (-450, -254), (-850, -254), (-880, -254)]),
    ("Justin Con basement -> tunnel -> Respawn Con basement", -176, [(400, -254), (450, -254), (850, -254), (880, -254)]),
    ("Justin Con: front door -> living room", 0, [(0, -420), (0, -330), (0, -250), (-200, -200)]),
]

if __name__ == "__main__":
    run(ROUTES)
