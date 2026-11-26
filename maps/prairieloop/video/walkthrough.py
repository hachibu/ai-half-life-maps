"""Prairie Loop walkthrough video: shows the 'Restore the Power' event in motion (fans, door, messages, alarm, teleport).

    python maps/prairieloop/video/walkthrough.py check     # camera never inside solid
    python maps/prairieloop/video/walkthrough.py render    # build/prairieloop/video/chunk_*.mp4 (2 processes)
    python maps/prairieloop/video/walkthrough.py audio     # build/prairieloop/video/audio.wav
    (make video MAP=prairieloop runs all three, joins them and makes a Discord-size copy)
"""
import os, sys, math, subprocess, wave
from pathlib import Path
ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT)); sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from tools.paths import build_dir, FONT_BOLD
BD = build_dir("prairieloop"); VID = BD / "video"; VID.mkdir(exist_ok=True)
BSP = str(BD / "prairieloop.bsp"); VALVE = str(BD / "valve")
import numpy as np
from PIL import Image, ImageDraw, ImageFont, ImageFilter

FPS, W_, H_ = 30, 854, 480
SEG_A = [  # street -> breaker 3 -> transformer -> blue house -> reveal -> hall -> server room -> teleport pad
    (0.0, (-2600, 40, 95), (-2950, 200, 80)), (2.5, (-2780, 120, 85), (-2950, 200, 75)),
    (4.5, (-2860, 150, 80), (-2944, 200, 70)), (6.5, (-2700, 60, 100), (-1500, 100, 100)),
    (10.0, (-1700, 0, 110), (-900, 300, 80)), (13.0, (-1100, 320, 120), (-835, 410, 40)),
    (15.5, (-700, 600, 110), (0, 640, 90)), (18.0, (250, 650, 80), (384, 900, 75)),
    (20.0, (384, 760, 64), (384, 1000, 64)), (21.3, (370, 900, 64), (380, 1100, 64)),
    (22.5, (380, 1080, 64), (600, 1150, 60)), (23.7, (570, 1190, 64), (300, 1224, -60)),
    (25.0, (440, 1224, -60), (200, 1224, -150)), (26.3, (330, 1190, -116), (144, 1000, -140)),
    (27.6, (300, 960, -116), (512, 880, -140)), (28.8, (512, 900, -116), (512, 700, -200)),
    (30.0, (512, 760, -170), (512, 500, -260)), (31.2, (512, 640, -250), (512, 400, -265)),
    (32.4, (512, 470, -258), (300, 0, -330)), (34.0, (512, 345, -258), (-100, -250, -340)),
    (36.0, (500, 330, -256), (-400, -100, -380)), (37.5, (380, 336, -300), (0, 200, -400)),
    (39.0, (220, 300, -398), (-320, 16, -110)), (41.5, (-100, 100, -380), (-560, -384, -400)),
    (43.0, (-450, -150, -400), (-560, -384, -410)), (46.5, (-560, -300, -400), (-560, -560, -420)),
    (48.5, (-560, -405, -400), (-560, -560, -405)), (50.0, (-590, -440, -400), (-690, -500, -420)),
    (51.5, (-560, -420, -395), (-560, -250, -400)), (52.5, (-560, -320, -395), (-200, -200, -400)), (53.6, (-250, -200, -390), (255, -330, -400)),
    (55.5, (180, -230, -390), (255, -330, -420)), (56.4, (250, -300, -366), (200, 100, -380)), (57.0, (252, -310, -366), (150, 200, -380)),
]
SEG_B = [  # arrive on the blue house roof -> roof loop -> steam vent -> wide shot
    (57.0, (384, 1150, 205), (384, 700, 150)), (59.0, (384, 1100, 205), (0, 0, 60)),
    (61.0, (700, 1056, 205), (1344, 1056, 200)), (64.0, (1300, 1056, 205), (1344, 700, 200)),
    (66.0, (1344, 760, 210), (1440, 576, 205)), (68.0, (1440, 500, 210), (1440, 0, 205)),
    (70.5, (1440, 0, 210), (1440, -500, 205)), (73.0, (1440, -560, 210), (1344, -900, 205)),
    (75.0, (1344, -900, 210), (1100, -1056, 205)), (77.5, (900, -1056, 210), (500, -1100, 190)),
    (80.0, (600, -1250, 260), (384, -1316, 30)), (82.5, (430, -1400, 240), (384, -1310, 20)),
    (85.0, (0, -900, 420), (0, 0, 0)), (88.0, (-300, -700, 560), (0, 100, 0)),
]
CUT = 57.0
DUR = SEG_B[-1][0]
T_BREAKER, T_POWER = 5.0, 43.5
T_DOOR0, DOOR_RISE, DOOR_TIME, DOOR_WAIT = T_POWER + 1.5, 104.0, 104.0 / 110.0, 14.0
T_ALARM0, T_ALARM1 = T_POWER + 0.6, T_POWER + 16.0

def catmull(p0, p1, p2, p3, u):
    p0, p1, p2, p3 = map(np.array, (p0, p1, p2, p3))
    return 0.5 * ((2 * p1) + (-p0 + p2) * u + (2 * p0 - 5 * p1 + 4 * p2 - p3) * u ** 2 + (-p0 + 3 * p1 - 3 * p2 + p3) * u ** 3)

def cam(t):
    K = SEG_A if t < CUT else SEG_B
    t = min(max(t, K[0][0]), K[-1][0] - 1e-6)
    i = max(j for j in range(len(K) - 1) if K[j][0] <= t)
    u = (t - K[i][0]) / (K[i + 1][0] - K[i][0]); u = u * u * (3 - 2 * u) * 0.35 + u * 0.65
    g = lambda k, idx: K[min(max(k, 0), len(K) - 1)][idx]
    return catmull(g(i - 1, 1), g(i, 1), g(i + 1, 1), g(i + 2, 1), u), catmull(g(i - 1, 2), g(i, 2), g(i + 1, 2), g(i + 2, 2), u)

def door_h(t):
    if t < T_DOOR0: return 0.0
    if t < T_DOOR0 + DOOR_TIME: return DOOR_RISE * (t - T_DOOR0) / DOOR_TIME
    t2 = t - (T_DOOR0 + DOOR_TIME + DOOR_WAIT)
    return DOOR_RISE if t2 < 0 else max(0.0, DOOR_RISE - DOOR_RISE * t2 / DOOR_TIME)

def rz(a):
    c, s_ = math.cos(a), math.sin(a); return np.array([[c, -s_, 0], [s_, c, 0], [0, 0, 1.0]])

FB = FONT_BOLD
def game_text(img, msg, y, col, alpha):
    if alpha <= 0: return img
    im = Image.fromarray(img); f = ImageFont.truetype(FB, 22)
    lay = Image.new("RGB", im.size); d = ImageDraw.Draw(lay)
    d.text((W_ // 2, int(H_ * y)), msg, fill=col, font=f, anchor="mm")
    glow = lay.filter(ImageFilter.GaussianBlur(3))
    a = np.asarray(im).astype(float) + (np.asarray(glow).astype(float) * 0.8 + np.asarray(lay).astype(float)) * alpha
    return np.clip(a, 0, 255).astype(np.uint8)

def fade(t, t0, hold, fin=0.15, fout=0.6):
    if t < t0 or t > t0 + fin + hold + fout: return 0.0
    if t < t0 + fin: return (t - t0) / fin
    if t < t0 + fin + hold: return 1.0
    return 1 - (t - t0 - fin - hold) / fout

def render_chunk(args):
    ci, f0, f1, path = args
    from tools.render.client import Renderer
    R = Renderer(BSP, asset_root=VALVE)
    door = [mi for mi, kv in R.minfo.items() if kv.get("targetname") == "server_door"][0]
    fans = [(mi, float(kv.get("speed", 0))) for mi, kv in R.minfo.items() if kv["classname"] == "func_rotating"]
    tmp = path[:-4] + ".part.mp4"
    p = subprocess.Popen(["ffmpeg", "-y", "-loglevel", "error", "-f", "rawvideo", "-pix_fmt", "rgb24", "-s", "%dx%d" % (W_, H_),
                          "-r", str(FPS), "-i", "-", "-c:v", "libx264", "-crf", "19", "-preset", "veryfast", "-pix_fmt", "yuv420p", tmp],
                         stdin=subprocess.PIPE)
    for fr in range(f0, f1):
        t = fr / FPS
        eye, tgt = cam(t)
        dyn = {door: (np.eye(3), np.array([0, 0, door_h(t)]))}
        for mi, sp in fans: dyn[mi] = (rz(math.radians(sp * t)), R.mtrans[mi][1])
        img = R.frame(eye, tgt, t, W_, H_, 80, dyn)
        img = game_text(img, "A breaker was switched on.  (Flip all 3 to restore power)", 0.7, (255, 220, 90), fade(t, T_BREAKER, 2.6))
        img = game_text(img, "POWER RESTORED!  The JustinCon server room is open.", 0.25, (255, 90, 200), fade(t, T_POWER, 4.5))
        fl = max(0.0, 1 - abs(t - CUT) / 0.35)                    # teleport flash
        if fl > 0: img = np.clip(img.astype(float) * (1 - fl) + 255 * fl, 0, 255).astype(np.uint8)
        p.stdin.write(img.tobytes())
        if fr % 30 == 0: print("chunk %d frame %d/%d" % (ci, fr - f0, f1 - f0), flush=True)
    p.stdin.close()
    R.close()
    if p.wait() == 0: os.replace(tmp, path)
    return path

def audio(path, house_boxes, sr=22050):
    def readwav(p_):
        with wave.open(p_) as w_: return np.frombuffer(w_.readframes(w_.getnframes()), "<i2").astype(float) / 32768
    n = int(DUR * sr)
    tile = lambda x: np.tile(x, n // len(x) + 1)[:n]
    wind, fans, alarm = tile(readwav(VALVE + "/sound/justincon/wind_loop.wav")), tile(readwav(VALVE + "/sound/justincon/fans_loop.wav")), tile(readwav(VALVE + "/sound/justincon/alarm.wav"))
    ts = np.arange(0, DUR + 0.1, 0.1); gw, gf, ga = [], [], []
    for t in ts:
        eye, _ = cam(t); z = eye[2]
        indoor = any(x0 <= eye[0] <= x1 and y0 <= eye[1] <= y1 and 0 <= z <= 144 for x0, y0, x1, y1 in house_boxes)
        gw.append((0.45 if indoor else 1.0) * np.clip(1 + min(z, 0) / 140, 0, 1))
        gf.append(np.clip((-z - 90) / 170, 0, 1) + (0.25 if (z < 0 and eye[1] > 850) else 0))
        on = T_ALARM0 <= t <= T_ALARM1
        ga.append((0.55 if z < -200 else 0.15) if on else 0.0)
    tt = np.arange(n) / sr
    mix = wind * np.interp(tt, ts, gw) * 0.8 + fans * np.interp(tt, ts, gf) * 0.9 + alarm * np.interp(tt, ts, ga)
    k0 = int((T_POWER + 0.2) * sr); L = int(1.2 * sr)              # distant transformer explosion: low thump
    r = np.random.default_rng(3); burst = r.standard_normal(L) * np.exp(-np.arange(L) / (0.25 * sr))
    burst = np.convolve(burst, np.ones(60) / 60, "same") * 2.5
    mix[k0:k0 + L] += burst * 0.5
    mix = np.clip(mix, -1, 1)
    with wave.open(path, "wb") as w_:
        w_.setnchannels(1); w_.setsampwidth(2); w_.setframerate(sr); w_.writeframes((mix * 32000).astype("<i2").tobytes())

if __name__ == "__main__":
    mode = sys.argv[1]; args = sys.argv[2:]
    if mode == "check":
        from tools.render.client import Renderer
        r = Renderer(BSP, asset_root=VALVE)
        bad = [round(f / FPS, 2) for f in range(int(DUR * FPS)) if r.solid(tuple(cam(f / FPS)[0]))]
        r.close()
        print("camera-in-solid frames:", len(bad), bad[:40])
    elif mode == "stills":
        times = list(map(float, args))
        for t in times: render_chunk((0, int(t * FPS), int(t * FPS) + 1, str(VID / ("still_%05.1f.mp4" % t))))
    elif mode == "render":
        from multiprocessing import Pool
        nfr = int(DUR * FPS); size = 220
        jobs = [(i, s, min(s + size, nfr), str(VID / ("chunk_%02d.mp4" % i))) for i, s in enumerate(range(0, nfr, size))]
        jobs = [j for j in jobs if not os.path.exists(j[3])]
        with Pool(2) as pool:
            for r in pool.imap_unordered(render_chunk, jobs): print("done", r, flush=True)
    elif mode == "audio":
        import io, contextlib
        with contextlib.redirect_stdout(io.StringIO()):
            import build
        audio(str(VID / "audio.wav"), [build.footprint(h) for h in build.HOUSES])
