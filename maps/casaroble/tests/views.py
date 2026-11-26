"""Collects VIEWS from every zone."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[3]))
from tools.mapkit import zone_views

VIEWS = zone_views(Path(__file__).resolve().parents[1])
