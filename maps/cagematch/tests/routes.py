"""Walk routes that must stay passable.  Run: make test MAP=cagematch"""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[3]))
from tools.checks.walk import run

ROUTES = [
    ("balcony walk around the pit (east -> north)", 192, [(590, 0), (590, 300), (420, 420), (0, 590)]),
    ("pit floor crossing", 0, [(220, 0), (0, 0), (-220, 0)]),
]

if __name__ == "__main__":
    run(ROUTES)
