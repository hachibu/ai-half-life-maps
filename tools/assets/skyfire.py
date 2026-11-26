"""Custom night skybox (6 TGA faces) and an animated additive fire sprite (.spr v2)."""
import math, struct
import numpy as np

rng = np.random.default_rng(1234)

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

def dir_to_face(d):
    """Inverse mapping for a single direction -> (face, s, t)."""
    x, y, z = d
    ax = np.abs(d); k = int(np.argmax(ax))
    if k == 0:
        return ("rt", -y / x, z / x) if x > 0 else ("lf", y / -x, z / -x)
    if k == 1:
        return ("bk", x / y, z / y) if y > 0 else ("ft", -x / -y, z / -y)
    return ("up", -y / z, -x / z) if z > 0 else ("dn", -y / -z, x / -z)

MOON = np.array([0.35, 1.0, 0.5]); MOON /= np.linalg.norm(MOON)
N_STARS = 3500
_sd = rng.normal(size=(N_STARS, 3)); _sd /= np.linalg.norm(_sd, axis=1, keepdims=True)
_sd[:, 2] = np.abs(_sd[:, 2])
STARS = _sd
STAR_B = rng.random(N_STARS) ** 4 * 255 + 25

def sky_color(D):
    D = D / np.linalg.norm(D, axis=-1, keepdims=True)
    z = D[..., 2]
    e = np.arcsin(np.clip(z, -1, 1))
    az = np.arctan2(D[..., 1], D[..., 0])
    zen = np.array([4, 6, 18.0]); hor = np.array([30, 28, 48.0])
    k = np.clip(e / (math.pi / 2), 0, 1) ** 0.6
    col = hor * (1 - k[..., None]) + zen * k[..., None]
    glow = np.exp(-np.maximum(e, 0) / 0.10)
    col += np.array([70, 40, 22.0]) * glow[..., None]            # suburban light pollution
    nb = np.array([0.3, -0.5, 0.8]); nb /= np.linalg.norm(nb)
    band = np.exp(-((D @ nb) ** 2) / 0.03)
    wob = 0.6 + 0.4 * np.sin(D @ np.array([9.0, 4, 7]) + 1) * np.sin(D @ np.array([-5.0, 11, 3]))
    col += np.array([22, 22, 34.0]) * (band * wob * np.clip(e / 0.3, 0, 1))[..., None]
    ang = np.arccos(np.clip(D @ MOON, -1, 1))
    col += np.array([60, 65, 80.0]) * np.exp(-ang / 0.09)[..., None]
    disk = ang < 0.04
    crater = 0.85 + 0.15 * np.sin(D @ np.array([90.0, 40, 70])) * np.sin(D @ np.array([-60.0, 80, 30]))
    col[disk] = (np.array([238, 234, 215.0]) * crater[disk][..., None])
    # tree line + ground
    tree = 0.035 + 0.018 * np.sin(3 * az) + 0.012 * np.sin(7 * az + 1) + 0.008 * np.sin(17 * az + 2)
    ground = e < tree
    col[ground] = np.array([6, 7, 9.0])
    return col, ground

def make_face(face, N=256):
    j, i = np.mgrid[0:N, 0:N]
    s = (i + 0.5) / N * 2 - 1
    t = 1 - (j + 0.5) / N * 2
    D = face_dirs(face, s, t)
    col, ground = sky_color(D)
    # stars projected into this face
    for d, b in zip(STARS, STAR_B):
        f, ss, tt = dir_to_face(d)
        if f != face or d[2] < 0.06: continue
        x = int((ss + 1) / 2 * N); y = int((1 - tt) / 2 * N)
        if 0 <= x < N and 0 <= y < N and not ground[y, x]:
            fade = min(1.0, d[2] / 0.25)
            col[y, x] += b * fade
            if b > 180:
                for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                    if 0 <= x + dx < N and 0 <= y + dy < N: col[y + dy, x + dx] += b * 0.35 * fade
    return np.clip(col, 0, 255).astype(np.uint8)

def write_tga(path, rgb):
    h, w, _ = rgb.shape
    hdr = struct.pack("<BBBHHBHHHHBB", 0, 0, 2, 0, 0, 0, 0, 0, w, h, 24, 0)
    data = np.ascontiguousarray(rgb[::-1, :, ::-1]).tobytes()   # bottom-up, BGR
    open(path, "wb").write(hdr + data)

def write_skybox(prefix):
    faces = {}
    for f in ("rt", "bk", "lf", "ft", "up", "dn"):
        faces[f] = make_face(f)
        write_tga(f"{prefix}{f}.tga", faces[f])
    return faces

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
