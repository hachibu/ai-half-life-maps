"""Skybox helpers (face directions, TGA writer) and an animated additive fire sprite (.spr v2)."""
import math, struct
import numpy as np

# GoldSrc/Quake2 skybox face -> direction for image coords s,t in [-1,1] (t=+1 is the top row)
def face_dirs(face, s, t):
    one = np.ones_like(s)
    return {
        "rt": np.stack([one, -s, t], -1),
        "lf": np.stack([-one, s, t], -1),
        "bk": np.stack([s, one, t], -1),
        "ft": np.stack([-s, -one, t], -1),
        "up": np.stack([-t, -s, one], -1),
        "dn": np.stack([t, -s, -one], -1),
    }[face]

def write_tga(path, rgb):
    h, w, _ = rgb.shape
    hdr = struct.pack("<BBBHHBHHHHBB", 0, 0, 2, 0, 0, 0, 0, 0, w, h, 24, 0)
    data = np.ascontiguousarray(rgb[::-1, :, ::-1]).tobytes()   # bottom-up, BGR
    open(path, "wb").write(hdr + data)

# ------------------------------------------------------------------ fire sprite
def fire_frames(w=64, h=128, n=12, seed=7):
    r = np.random.default_rng(seed)
    heat = np.zeros((h + 2, w))
    frames = []
    xs = np.linspace(-1, 1, w)
    for step in range(n * 3 + 40):
        heat[-2:, :] = r.random((2, w)) ** 0.5 * 255 * (np.abs(xs) < 0.75)
        up = np.roll(heat, -1, 0)
        avg = (up + np.roll(up, 1, 1) + np.roll(up, -1, 1) + np.roll(heat, -2, 0)) / 4.0
        heat = np.maximum(avg - r.random(heat.shape) * 3.2, 0)
        if step >= 40 and (step - 40) % 3 == 0:
            f = heat[:h].copy()
            yy = np.linspace(0, 1, h)[:, None]
            env = np.clip(1 - np.abs(xs)[None, :] / (0.25 + 0.75 * yy), 0, 1)   # taper to a tip
            frames.append(np.clip(f * env * 1.3, 0, 255))
    return frames[:n]

def fire_palette():
    pal = []
    for i in range(256):
        t = i / 255
        r = min(255, int(t * 3 * 255)); g = min(255, int(max(0, t * 3 - 1) * 255)); b = min(255, int(max(0, t * 3 - 2) * 255))
        pal.append((r, int(g * 0.85), int(b * 0.6)))
    pal[0] = (0, 0, 0)
    return pal

def write_sprite(path):
    frames = fire_frames()
    h, w = frames[0].shape
    pal = fire_palette()
    out = struct.pack("<4si iifiiifi".replace(" ", ""), b"IDSP", 2, 0, 1, math.hypot(w, h) / 2, w, h, len(frames), 0.0, 1)
    out += struct.pack("<h", 256) + bytes(c for p in pal for c in p)
    for f in frames:
        idx = np.clip(f, 0, 255).astype(np.uint8)
        out += struct.pack("<i", 0) + struct.pack("<iiii", -w // 2, h, w, h) + idx.tobytes()
    open(path, "wb").write(out)
    return frames, pal
