"""Repository paths shared by every tool and map."""
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
FONTS = Path(__file__).resolve().parent / "fonts"
FONT_BOLD = str(FONTS / "Poppins-Bold.ttf")
FONT_MONO = str(FONTS / "DejaVuSansMono.ttf")


def build_dir(map_name: str) -> Path:
    """build/<map>/ - every generated file goes here (git-ignored)."""
    d = REPO / "build" / map_name
    d.mkdir(parents=True, exist_ok=True)
    return d
