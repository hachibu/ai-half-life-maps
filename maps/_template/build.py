#!/usr/bin/env python3
"""Assembles every file in zones/ into the map. You normally never edit this file: add or change zones instead."""
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from tools.mapkit import build_map

build_map(__file__, worldspawn={"message": "Template map"})
