"""Collects ROUTES from every zone (plus any listed here) and walks them."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[3]))
from tools.checks.walk import run
from tools.mapkit import zone_routes

ROUTES = zone_routes(Path(__file__).resolve().parents[1])

if __name__ == "__main__":
    run(ROUTES)
