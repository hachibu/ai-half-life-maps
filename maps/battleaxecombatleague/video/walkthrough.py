"""BattleAxe walkthrough video: fight night from the red corner's locker room, down the walkout tunnel and the flaming
runway into the cage, then up an aisle through the crowd, along the concourse, up the balcony stairs to the upper
deck, and a high shot over the octagon.

    python maps/battleaxecombatleague/video/walkthrough.py check     # camera never inside solid
    python maps/battleaxecombatleague/video/walkthrough.py render    # build/battleaxecombatleague/video/chunk_*.mp4
    python maps/battleaxecombatleague/video/walkthrough.py audio     # build/battleaxecombatleague/video/audio.wav
    (make video MAP=battleaxecombatleague runs all three, joins them and makes a Discord-size copy)
"""
import os, sys, subprocess, wave
from pathlib import Path
ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))
from tools.mapkit.paths import build_dir, FONT_BOLD
import numpy as np
from PIL import Image, ImageDraw, ImageFont, ImageFilter

NAME = "battleaxecombatleague"
BD = build_dir(NAME); VID = BD / "video"; VID.mkdir(exist_ok=True)
BSP = str(BD / (NAME + ".bsp")); VALVE = str(BD / "valve")
FPS, W_, H_ = 30, 854, 480

KEYS = [  # (time, eye, target)
    (0.0, (0, -1990, 100), (0, -1700, 92)), (3.5, (0, -1840, 96), (0, -1300, 92)),
    (6.5, (0, -1480, 84), (0, -400, 92)), (9.5, (0, -760, 84), (0, 0, 80)),
    (12.5, (0, -380, 92), (0, 0, 72)), (15.0, (110, -130, 96), (-200, 220, 60)),
    (16.8, (220, -220, 230), (0, 0, 60)), (18.8, (430, -340, 190), (0, 0, 60)), (21.0, (760, -60, 72), (900, 0, 44)),
    (24.0, (980, 0, 160), (1900, 0, 420)), (27.0, (1420, 0, 300), (1900, 0, 440)),
    (30.0, (1760, 260, 520), (1760, 1100, 560)), (33.5, (1420, 760, 600), (1760, 1120, 560)),
    (36.5, (1480, 1060, 660), (1700, 1180, 620)), (39.0, (1640, 1245, 676), (0, 0, 200)),
    (43.0, (1640, 1640, 880), (0, 0, 120)), (47.0, (700, -1100, 980), (0, 0, 300)),
    (51.0, (-600, -1300, 900), (0, 0, 250)),
]
DUR = KEYS[-1][0]
CAPTIONS = [(0.6, 3.2, "BATTLEAXE COMBAT LEAGUE"), (7.0, 2.4, "The walkout"), (13.0, 2.4, "The cage"),
            (24.5, 2.4, "Through the crowd"), (33.5, 2.4, "Balcony stairs"), (43.5, 2.6, "Upper deck")]


def catmull(p0, p1, p2, p3, u):
    p0, p1, p2, p3 = map(np.array, (p0, p1, p2, p3))
    return 0.5 * ((2 * p1) + (-p0 + p2) * u + (2 * p0 - 5 * p1 + 4 * p2 - p3) * u ** 2 + (-p0 + 3 * p1 - 3 * p2 + p3) * u ** 3)


def cam(t):
    K = KEYS
    t = min(max(t, K[0][0]), K[-1][0] - 1e-6)
    i = max(j for j in range(len(K) - 1) if K[j][0] <= t)
    u = (t - K[i][0]) / (K[i + 1][0] - K[i][0]); u = u * u * (3 - 2 * u) * 0.35 + u * 0.65
    g = lambda k, idx: K[min(max(k, 0), len(K) - 1)][idx]
    return catmull(g(i - 1, 1), g(i, 1), g(i + 1, 1), g(i + 2, 1), u), catmull(g(i - 1, 2), g(i, 2), g(i + 1, 2), g(i + 2, 2), u)


def fade(t, t0, hold, fin=0.3, fout=0.6):
    if t < t0 or t > t0 + fin + hold + fout: return 0.0
    if t < t0 + fin: return (t - t0) / fin
    if t < t0 + fin + hold: return 1.0
    return 1 - (t - t0 - fin - hold) / fout


def caption(img, msg, alpha, big):
    if alpha <= 0: return img
    im = Image.fromarray(img); f = ImageFont.truetype(FONT_BOLD, 44 if big else 24)
    lay = Image.new("RGB", im.size); d = ImageDraw.Draw(lay)
    xy = (W_ // 2, int(H_ * 0.18)) if big else (36, H_ - 44)
    d.text(xy, msg, fill=(255, 238, 214), font=f, anchor="mm" if big else "lm")
    sh = Image.new("RGB", im.size); ImageDraw.Draw(sh).text((xy[0] + 2, xy[1] + 2), msg, fill=(255, 255, 255), font=f, anchor="mm" if big else "lm")
    a = np.asarray(im).astype(float) * (1 - 0.45 * alpha * (np.asarray(sh.filter(ImageFilter.GaussianBlur(3)))[..., :1] / 255.0))
    a = a + np.asarray(lay).astype(float) * alpha
    return np.clip(a, 0, 255).astype(np.uint8)


def render_chunk(args):
    ci, f0, f1, path = args
    from tools.mapkit.render import Renderer
    R = Renderer(BSP, asset_root=VALVE)
    tmp = path[:-4] + ".part.mp4"
    p = subprocess.Popen(["ffmpeg", "-y", "-loglevel", "error", "-f", "rawvideo", "-pix_fmt", "rgb24", "-s", "%dx%d" % (W_, H_),
                          "-r", str(FPS), "-i", "-", "-c:v", "libx264", "-crf", "19", "-preset", "veryfast", "-pix_fmt", "yuv420p", tmp],
                         stdin=subprocess.PIPE)
    for fr in range(f0, f1):
        t = fr / FPS
        eye, tgt = cam(t)
        img = R.frame(eye, tgt, t, W_, H_, 80, {})
        for t0, hold, msg in CAPTIONS:
            img = caption(img, msg, fade(t, t0, hold), msg.isupper())
        fo = min(1.0, t / 0.6, (DUR - t) / 0.8)                  # fade in from / out to black
        if fo < 1: img = (img.astype(float) * max(fo, 0)).astype(np.uint8)
        p.stdin.write(img.tobytes())
        if fr % 30 == 0: print("chunk %d frame %d/%d" % (ci, fr - f0, f1 - f0), flush=True)
    p.stdin.close()
    R.close()
    if p.wait() == 0: os.replace(tmp, path)
    return path


def audio(path, sr=22050):
    """Synthesised arena: crowd roar that swells and cheers, muffled in the locker room and the walkout tunnel,
    a pulse of bass from the PA, and big cheers at the cage, in the crowd and on the upper deck."""
    n = int(DUR * sr); r = np.random.default_rng(9); tt = np.arange(n) / sr
    def lowpass(x, k): return np.convolve(x, np.ones(k) / k, "same")
    roar = lowpass(r.standard_normal(n), 6) * 0.6 + lowpass(r.standard_normal(n), 30) * 1.6
    roar *= 0.75 + 0.25 * np.sin(tt * 0.9) * np.sin(tt * 0.37 + 1)
    for t0 in (12.8, 24.0, 39.5):                               # cheers: louder, brighter swells
        roar += lowpass(r.standard_normal(n), 3) * np.exp(-((tt - t0) / 1.2) ** 2) * 1.2
    muffled = lowpass(roar, 60) * 2.5
    bass = np.sin(2 * np.pi * 55 * tt) * (np.sin(np.pi * tt * 2) > 0.6) * np.exp(-((tt * 2) % 1) * 6) * 0.5
    ts = np.arange(0, DUR + 0.1, 0.1); gin, gm = [], []
    for t in ts:
        e, _ = cam(t)
        m = float(np.clip((-e[1] - 1300) / 300, 0, 1)) if abs(e[0]) < 400 else 0.0   # locker room + tunnel
        gm.append(m); gin.append(1 - m)
    g = lambda v: np.interp(tt, ts, v)
    mix = roar * g(gin) + muffled * g(gm) + bass * (0.4 + 0.6 * g(gm))
    mix *= np.clip(np.minimum(tt / 0.6, (DUR - tt) / 0.8), 0, 1)
    mix = np.clip(mix / (np.abs(mix).max() + 1e-9) * 0.8, -1, 1)
    with wave.open(path, "wb") as w_:
        w_.setnchannels(1); w_.setsampwidth(2); w_.setframerate(sr); w_.writeframes((mix * 32000).astype("<i2").tobytes())


if __name__ == "__main__":
    mode = sys.argv[1]; args = sys.argv[2:]
    if mode == "check":
        from tools.mapkit.render import Renderer
        r = Renderer(BSP, asset_root=VALVE)
        bad = [round(f / FPS, 2) for f in range(int(DUR * FPS)) if r.solid(tuple(cam(f / FPS)[0]))]
        r.close()
        print("camera-in-solid frames:", len(bad), bad[:40])
    elif mode == "stills":
        for t in map(float, args): render_chunk((0, int(t * FPS), int(t * FPS) + 1, str(VID / ("still_%05.1f.mp4" % t))))
    elif mode == "render":
        from multiprocessing import Pool
        nfr = int(DUR * FPS); size = 220
        jobs = [(i, s, min(s + size, nfr), str(VID / ("chunk_%02d.mp4" % i))) for i, s in enumerate(range(0, nfr, size))]
        jobs = [j for j in jobs if not os.path.exists(j[3])]
        with Pool(2) as pool:
            for r in pool.imap_unordered(render_chunk, jobs): print("done", r, flush=True)
    elif mode == "audio":
        audio(str(VID / "audio.wav"))
