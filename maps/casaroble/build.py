#!/usr/bin/env python3
"""Assembles every file in zones/ into the map, then writes the sky and fire sprite. Add or change zones instead."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
sys.path.insert(0, str(Path(__file__).resolve().parent))
from tools.mapkit import build_map
import assets

out = build_map(__file__, worldspawn={"message": "Casa Roble", "skyname": "casaroble"})
assets.write(out / "valve", "casaroble")
