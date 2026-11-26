"""Casa Roble walkthrough video: street -> porch -> fireplace -> skylit kitchen -> hall -> primary bedroom -> deck
under the oaks -> garage office -> down the driveway -> crane shot over the house.

    python maps/casaroble/video/walkthrough.py check     # camera never inside solid
    python maps/casaroble/video/walkthrough.py render    # build/casaroble/video/chunk_*.mp4 (2 processes)
    python maps/casaroble/video/walkthrough.py audio     # build/casaroble/video/audio.wav
    (make video MAP=casaroble runs all three, joins them and makes a Discord-size copy)
"""
import os, sys, subprocess, wave
from pathlib import Path
ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT))
from tools.mapkit.paths import build_dir, FONT_BOLD
import numpy as np
from PIL import Image, ImageDraw, ImageFont, ImageFilter

NAME = "casaroble"
BD = build_dir(NAME); VID = BD / "video"; VID.mkdir(exist_ok=True)
BSP = str(BD / (NAME + ".bsp")); VALVE = str(BD / "valve")
FPS, W_, H_ = 30, 854, 480

KEYS = [  # (time, eye, target)
    (0.0, (80, -340, 110), (-80, 400, 110)), (3.5, (-60, -230, 90), (-130, 400, 100)),
    (6.0, (-144, -40, 66), (-144, 300, 90)), (8.5, (-144, 190, 78), (-150, 420, 76)),
    (10.5, (-140, 320, 78), (-372, 416, 46)), (13.5, (-110, 470, 78), (-372, 416, 40)),
    (16.0, (-10, 500, 78), (150, 760, 140)), (18.5, (20, 690, 78), (150, 730, 230)),
    (21.0, (0, 845, 78), (-384, 850, 70)), (23.5, (-48, 850, 78), (-48, 1000, 72)),
    (26.0, (-84, 1000, 78), (-170, 1200, 60)), (28.5, (-160, 1150, 78), (-150, 1400, 60)),
    (31.0, (-130, 1320, 80), (200, 1330, 140)), (34.0, (-40, 1500, 72), (240, 1600, 60)),
    (37.0, (130, 1630, 66), (360, 1632, 60)), (39.5, (330, 1630, 66), (390, 1760, 40)),
    (42.0, (360, 1560, 66), (360, 1000, 66)), (45.0, (360, 1100, 70), (360, 200, 70)),
    (48.0, (390, 320, 84), (0, -220, 90)), (51.5, (160, -200, 300), (-80, 700, 60)),
    (55.0, (0, -330, 540), (-60, 900, 40)),
]
DUR = KEYS[-1][0]
CAPTIONS = [(0.6, 3.2, "CASA ROBLE"), (10.8, 2.6, "Living room"), (16.5, 2.6, "Kitchen"),
            (24.0, 2.4, "Primary bedroom"), (31.3, 2.6, "Deck under the oaks"), (39.6, 2.4, "Garage office")]
HOUSE = (-400, 256, 272, 1216)          # interior: quieter breeze, fire crackle near the hearth
GARAGE = (272, 1472, 448, 1792)
FIRE = np.array([-372, 416, 40])


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


def inside(box, e):
    return box[0] <= e[0] <= box[2] and box[1] <= e[1] <= box[3] and e[2] < 200


def audio(path, sr=22050):
    """Synthesised: a soft breeze outdoors, birdsong in the yards, room tone indoors and the fire crackling."""
    n = int(DUR * sr); r = np.random.default_rng(5); tt = np.arange(n) / sr
    def lowpass(x, k): return np.convolve(x, np.ones(k) / k, "same")
    breeze = lowpass(r.standard_normal(n), 40) * (0.6 + 0.4 * np.sin(tt * 0.7) ** 2) * 2.2
    birds = np.zeros(n)
    for t0 in np.sort(r.uniform(0, DUR, 70)):                    # short chirps, a few notes each
        f0 = r.uniform(2600, 4200)
        for k in range(r.integers(2, 5)):
            s = int((t0 + k * 0.11) * sr); L = int(0.07 * sr)
            if s + L >= n: break
            ts = np.arange(L) / sr
            birds[s:s + L] += np.sin(2 * np.pi * (f0 + 900 * np.sin(ts * 40)) * ts) * np.hanning(L) * 0.12
    crackle = np.zeros(n)
    for s in r.integers(0, n - 200, int(DUR * 25)):
        crackle[s:s + 120] += r.standard_normal(120) * np.exp(-np.arange(120) / 18) * r.uniform(0.1, 0.5)
    crackle = crackle + lowpass(r.standard_normal(n), 200) * 1.5
    room = lowpass(r.standard_normal(n), 400) * 2.0
    ts = np.arange(0, DUR + 0.1, 0.1); go, gi, gf = [], [], []
    for t in ts:
        e, _ = cam(t)
        ins = inside(HOUSE, e) or inside(GARAGE, e)
        go.append(0.25 if ins else 1.0); gi.append(1.0 if ins else 0.0)
        gf.append(np.clip(1 - np.linalg.norm(e - FIRE) / 420, 0, 1) if inside(HOUSE, e) else 0.0)
    g = lambda v: np.interp(tt, ts, v)
    mix = (breeze * 0.35 + birds) * g(go) + room * 0.2 * g(gi) + crackle * 0.5 * g(gf)
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
