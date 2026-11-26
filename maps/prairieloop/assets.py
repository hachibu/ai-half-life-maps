"""Assets for Prairie Loop Winter: JustinCon After Hours. All artwork/audio generated here, original."""
import math, struct, os, sys
import numpy as np
from PIL import Image, ImageDraw, ImageFilter
from tools.mapkit.textures import noise, colorize, img, font, FB, MONO, siding_f, planks, T as JT
from tools.mapkit import sky

rng = np.random.default_rng(2026)
T = {}
for k in ("WALLPAPER", "WOODFLOOR", "CEILING", "BASEWALL", "CEILTILE", "COUCH", "COUNTER", "FRIDGE", "TVBLACK",
          "STEP", "TRIM", "METAL", "BRICK", "CARPET", "LAMP", "GLASS", "CARGLASS", "TIRE", "CARRED", "CARBLUE", "CARSILVER", "WOODFENCE", "sky"):
    T[k] = JT[k]

# ------------------------------------------------------------------ outdoor
def snow(seed=1):
    def f():
        r = np.random.default_rng(seed)
        a = noise(128, 128, 3) * 0.5 + r.random((128, 128)) * 0.5
        return img(colorize(a, (196, 204, 220), (240, 244, 252)))
    return f
T["SNOW"] = snow(1)
T["ROOFSNOW"] = snow(2)
def snowroad():
    a = noise(128, 128, 3) * 0.5 + rng.random((128, 128)) * 0.5
    c = colorize(a, (170, 176, 190), (225, 230, 240))
    for x0 in (22, 40, 86, 104):                       # tyre ruts
        c[:, x0:x0 + 8] = c[:, x0:x0 + 8] * 0.72 + np.array([40, 40, 44]) * 0.28
    return img(c)
T["SNOWROAD"] = snowroad
def ice():
    a = colorize(noise(128, 128, 6) * 0.6 + 0.4, (150, 185, 210), (205, 228, 242))
    im = img(a); d = ImageDraw.Draw(im)
    for _ in range(40):
        x, y = rng.integers(0, 128, 2); L = rng.integers(10, 40); ang = rng.random() * math.pi
        d.line([x, y, x + L * math.cos(ang), y + L * math.sin(ang)], fill=(235, 245, 252))
    return im
T["ICE"] = ice
def rinkboard():
    a = colorize(noise(64, 32, 1) * 0.2 + 0.8, (220, 222, 225), (245, 246, 248))
    a[20:26] = (40, 70, 170); a[0:3] = (230, 60, 60)
    return img(a)
T["RINKBOARD"] = rinkboard
for nm, c0, c1 in (("SIDING_BLUE", (50, 85, 150), (85, 125, 195)), ("SIDING_CREAM", (190, 175, 140), (230, 218, 185)),
                   ("SIDING_RED", (120, 40, 35), (165, 65, 55)), ("SIDING_GREEN", (70, 100, 75), (110, 145, 115)),
                   ("SIDING_GRAY", (110, 112, 118), (155, 158, 165)), ("SIDING_YELLOW", (185, 160, 80), (225, 200, 120)),
                   ("SIDING_WHITE", (200, 202, 205), (238, 240, 242)), ("SIDING_BROWN", (100, 75, 55), (145, 110, 80)),
                   ("SIDING_SAGE", (120, 135, 110), (160, 175, 150))):
    T[nm] = siding_f(c0, c1)
def garagedoor():
    a = colorize(noise(64, 64, 1) * 0.2 + 0.8, (190, 192, 195), (225, 226, 228))
    for y in range(0, 64, 16): a[y:y + 2] *= 0.6
    return img(a)
T["GARAGEDOOR"] = garagedoor
T["BARK"] = lambda: img(colorize(noise(32, 64, 0.7), (45, 32, 22), (90, 65, 45)))
def pine():
    a = colorize(rng.random((64, 64)) * 0.6 + noise(64, 64, 1) * 0.4, (14, 38, 24), (40, 80, 50))
    m = rng.random((64, 64)) > 0.86
    a[m] = (215, 222, 235)                         # snow on the needles
    return img(a)
T["PINE"] = pine
T["SNOWMAN"] = lambda: img(colorize(noise(32, 32, 2) * 0.3 + 0.7, (215, 220, 232), (245, 247, 252)))
T["COAL"] = lambda: img(colorize(noise(16, 16, 1), (10, 10, 12), (40, 40, 44)))
T["CARROT"] = lambda: img(colorize(noise(16, 16, 1), (200, 90, 20), (240, 130, 40)))
T["SCARF"] = lambda: img(colorize(noise(16, 16, 1), (150, 20, 30), (200, 40, 50)))
T["PLANK"] = planks((95, 70, 45), (150, 115, 80))
def plate(text, fg, bg, size=14, w=128, h=32, fnt=None):
    def f():
        im = Image.new("RGB", (w, h), bg)
        ImageDraw.Draw(im).text((w // 2, h // 2), text, fill=fg, font=font(fnt or FB, size), anchor="mm")
        return im
    return f
T["STREETSIGN"] = plate("PRAIRIE LOOP", (255, 255, 255), (30, 110, 60))
T["DEADEND"] = plate("DEAD END", (20, 20, 20), (240, 200, 40), size=16)

# ------------------------------------------------------------------ tunnels
T["TUNNELWALL"] = lambda: img(colorize(rng.random((64, 64)) * 0.4 + noise(64, 64, 2) * 0.6, (60, 60, 66), (100, 100, 108)))
T["TUNNELFLOOR"] = lambda: img(colorize(rng.random((64, 64)) * 0.5 + noise(64, 64, 1) * 0.5, (45, 45, 50), (80, 80, 88)))
for nm, c in (("STRIP_PINK", (255, 60, 190)), ("STRIP_CORAL", (255, 120, 90)), ("STRIP_CYAN", (40, 230, 255)),
              ("STRIP_VIOLET", (150, 80, 255)), ("STRIP_WHITE", (235, 235, 255))):
    T[nm] = (lambda c=c: img(np.tile(np.array(c, float), (16, 16, 1))))

def axe_emblem(d, cx, cy, s, col, edge):
    """Original angular axe: diagonal haft + faceted double-bit head."""
    d.line([cx - 0.9 * s, cy + 0.9 * s, cx + 0.55 * s, cy - 0.55 * s], fill=(70, 40, 30), width=max(2, int(s * 0.16)))
    hx, hy = cx + 0.45 * s, cy - 0.45 * s
    head = [(hx - 0.15 * s, hy - 0.62 * s), (hx + 0.5 * s, hy - 0.35 * s), (hx + 0.62 * s, hy + 0.15 * s),
            (hx + 0.15 * s, hy + 0.05 * s), (hx - 0.05 * s, hy - 0.15 * s)]
    d.polygon(head, fill=col, outline=edge)
    back = [(hx - 0.05 * s, hy - 0.15 * s), (hx - 0.45 * s, hy - 0.2 * s), (hx - 0.3 * s, hy + 0.05 * s), (hx - 0.1 * s, hy + 0.12 * s)]
    d.polygon(back, fill=col, outline=edge)

def moon_emblem(d, cx, cy, s, col, bg):
    d.ellipse([cx - s, cy - s, cx + s, cy + s], fill=col)
    d.ellipse([cx - s * 0.55, cy - s * 1.05, cx + s * 1.25, cy + s * 0.75], fill=bg)
    pass

def emblem_plate(kind):
    def f():
        bg = (25, 8, 22) if kind == "axe" else (8, 14, 34)
        im = Image.new("RGB", (64, 64), bg); d = ImageDraw.Draw(im)
        if kind == "axe":
            d.rectangle([1, 1, 62, 62], outline=(255, 110, 90), width=2)
            axe_emblem(d, 32, 34, 26, (255, 70, 180), (255, 170, 150))
        else:
            d.rectangle([1, 1, 62, 62], outline=(150, 90, 255), width=2)
            moon_emblem(d, 30, 32, 20, (60, 230, 255), bg)
        return im
    return f
T["EMB_AXE"] = emblem_plate("axe")
T["EMB_MOON"] = emblem_plate("moon")
T["TS_AXE"] = plate("BATTLEAXE  >>", (255, 120, 200), (25, 8, 22), size=13)
T["TS_MOON"] = plate("<<  CRESCENT", (90, 235, 255), (8, 14, 34), size=13)
T["TS_JC"] = plate("JUSTINCON  v", (255, 240, 255), (30, 10, 45), size=13)

# ------------------------------------------------------------------ the hall
def arcade_carpet():
    r = np.random.default_rng(98)
    im = Image.new("RGB", (128, 128), (14, 10, 40)); d = ImageDraw.Draw(im)
    cols = [(255, 60, 190), (40, 230, 255), (255, 210, 60), (150, 80, 255), (90, 255, 140)]
    for _ in range(26):
        x, y = r.integers(0, 128, 2); c = cols[int(r.integers(len(cols)))]; k = r.integers(0, 4)
        if k == 0: d.polygon([(x, y), (x + 10, y + 3), (x + 3, y + 11)], outline=c, width=2)
        elif k == 1: d.arc([x, y, x + 12, y + 12], 0, 270, fill=c, width=2)
        elif k == 2:
            pts = [(x + i * 4, y + (i % 2) * 5) for i in range(5)]; d.line(pts, fill=c, width=2)
        else: d.ellipse([x, y, x + 5, y + 5], fill=c)
    return im
T["ARCADECARPET"] = arcade_carpet
def hallwall():
    a = colorize(noise(128, 128, 2) * 0.3, (26, 14, 44), (50, 28, 78))
    a[::64] = (90, 50, 140); a[:, ::64] = (90, 50, 140)
    return img(a)
T["HALLWALL"] = hallwall
def hallceil():
    a = colorize(noise(64, 64, 1) * 0.2, (10, 8, 20), (25, 20, 40))
    a[::32] = (60, 40, 90); a[:, ::32] = (60, 40, 90)
    return img(a)
T["HALLCEIL"] = hallceil
def stagefloor():
    a = colorize(noise(64, 64, 2) * 0.2, (8, 8, 14), (30, 26, 44))
    a[::16] = (255, 60, 190); a[:, ::16] = (40, 230, 255)
    return img(a)
T["STAGEFLOOR"] = stagefloor
def stageback():
    W_, H_ = 256, 128
    im = Image.new("RGB", (W_, H_)); d = ImageDraw.Draw(im)
    for y in range(64):                               # vaporwave sunset sky
        t = y / 63; d.line([0, y, W_, y], fill=(int(40 + 215 * t), int(10 + 70 * t), int(90 + 60 * (1 - t))))
    for i, y in enumerate(range(20, 60, 6)):          # striped sun
        w = int(math.sqrt(max(0, 34 ** 2 - (y - 50) ** 2)))
        d.line([128 - w, y, 128 + w, y], fill=(255, 200 - i * 12, 80 + i * 10), width=4 if i % 2 == 0 else 2)
    d.rectangle([0, 64, W_, H_], fill=(20, 6, 40))
    for x in range(-512, 768, 32): d.line([128, 64, x, H_], fill=(255, 60, 190))   # perspective grid
    for i in range(7):
        y = 64 + int(64 * (i / 6) ** 1.8); d.line([0, y, W_, y], fill=(40, 230, 255))
    return im
T["STAGEBACK"] = stageback
def neon_sign():
    im = Image.new("RGB", (256, 64), (6, 2, 14))
    g = Image.new("RGB", (256, 64)); ImageDraw.Draw(g).text((128, 30), "JustinCon", fill=(255, 70, 200), font=font(FB, 44), anchor="mm")
    g = g.filter(ImageFilter.GaussianBlur(5))
    im = Image.fromarray(np.clip(np.asarray(im).astype(int) + np.asarray(g).astype(int) * 2, 0, 255).astype(np.uint8))
    d = ImageDraw.Draw(im)
    d.text((128, 30), "JustinCon", fill=(255, 225, 250), font=font(FB, 44), anchor="mm")
    d.rectangle([2, 2, 253, 61], outline=(40, 230, 255), width=3)
    return im
T["NEON_JC"] = neon_sign
def matchboard():
    im = Image.new("RGB", (256, 64), (5, 5, 12)); d = ImageDraw.Draw(im)
    d.rectangle([0, 0, 127, 63], fill=(45, 8, 30)); d.rectangle([128, 0, 255, 63], fill=(8, 18, 50))
    axe_emblem(d, 20, 32, 14, (255, 70, 180), (255, 170, 150)); moon_emblem(d, 236, 32, 12, (60, 230, 255), (8, 18, 50))
    d.text((74, 20), "BATTLEAXE", fill=(255, 130, 200), font=font(FB, 15), anchor="mm")
    d.text((184, 20), "CRESCENT", fill=(110, 235, 255), font=font(FB, 15), anchor="mm")
    d.text((128, 20), "vs", fill=(255, 255, 255), font=font(FB, 12), anchor="mm")
    d.text((74, 46), "7", fill=(255, 210, 60), font=font(FB, 20), anchor="mm")
    d.text((184, 46), "7", fill=(255, 210, 60), font=font(FB, 20), anchor="mm")
    d.text((128, 46), "MAP 3", fill=(200, 200, 220), font=font(MONO, 9), anchor="mm")
    return im
T["MATCHBOARD"] = matchboard
def banner(kind):
    def f():
        W_, H_ = 128, 256
        if kind == "axe":
            bg, a1, a2, name = (40, 6, 30), (255, 70, 180), (255, 120, 90), "BATTLEAXE"
        else:
            bg, a1, a2, name = (6, 12, 44), (40, 230, 255), (150, 80, 255), "CRESCENT"
        im = Image.new("RGB", (W_, H_), bg); d = ImageDraw.Draw(im)
        d.rectangle([4, 4, W_ - 5, H_ - 5], outline=a2, width=4)
        d.polygon([(4, H_ - 40), (64, H_ - 5), (124, H_ - 40), (124, H_ - 5), (4, H_ - 5)], fill=(0, 0, 0))
        if kind == "axe": axe_emblem(d, 64, 92, 46, a1, (255, 200, 180))
        else: moon_emblem(d, 60, 92, 40, a1, bg)
        d.text((64, 170), name, fill=(255, 255, 255), font=font(FB, 17), anchor="mm")
        d.line([20, 186, 108, 186], fill=a2, width=3)
        return im
    return f
T["BANNER_AXE"] = banner("axe")
T["BANNER_MOON"] = banner("moon")
T["DESK"] = lambda: img(colorize(noise(32, 32, 2) * 0.2 + 0.8, (60, 55, 70), (85, 80, 95)))
T["CRT"] = lambda: img(colorize(noise(32, 32, 2) * 0.2 + 0.8, (175, 165, 140), (215, 205, 180)))
T["CHAIR"] = lambda: img(colorize(noise(16, 16, 1) * 0.4, (20, 20, 30), (55, 50, 80)))
T["KEYBOARD"] = lambda: img(colorize(noise(16, 16, 1) * 0.3 + 0.7, (170, 165, 150), (210, 205, 190)))

HANDLES = ['FragTaco', 'PixelMoose', 'LagWizard', 'SnowCrab', 'QuadToast', 'NullHornet', 'ByteKnight', 'CampKing', 'RocketRita', 'GibMaster', 'Wavedash', 'MidnightMod']  # invented handles
def handle_screen(name, faction):
    def f():
        W_, H_ = 128, 96
        bg = (16, 6, 30) if faction == "axe" else (4, 14, 34)
        acc = (255, 90, 190) if faction == "axe" else (60, 225, 255)
        im = Image.new("RGB", (W_, H_), bg); d = ImageDraw.Draw(im)
        d.rectangle([0, 0, W_ - 1, 15], fill=acc)
        d.text((4, 8), "PLAYER", fill=bg, font=font(MONO, 10), anchor="lm")
        d.text((W_ - 4, 8), "BATTLEAXE" if faction == "axe" else "CRESCENT", fill=bg, font=font(FB, 9), anchor="rm")
        size = 30
        while size > 10:
            fnt = font(FB, size)
            if d.textlength(name, font=fnt) <= W_ - 10: break
            size -= 1
        d.text((W_ // 2, 50), name, fill=(255, 255, 255), font=fnt, anchor="mm")
        d.line([10, 70, W_ - 10, 70], fill=acc, width=2)
        d.text((W_ // 2, 83), "READY  >_", fill=acc, font=font(MONO, 11), anchor="mm")
        a = np.asarray(im).astype(float); a[::2] *= 0.85           # scanlines
        return img(a)
    return f
def handle_tex(name): return ("H_" + name.upper())[:15]
for i, h in enumerate(HANDLES):
    T[handle_tex(h)] = handle_screen(h, "axe" if i % 2 == 0 else "moon")
def crt_game():
    im = Image.new("RGB", (128, 96), (0, 0, 0)); d = ImageDraw.Draw(im)
    for y in range(48): d.line([0, y, 127, y], fill=(20 + y, 10, 50 + y * 2))
    d.rectangle([0, 48, 127, 95], fill=(30, 20, 40))
    for x in range(-200, 330, 24): d.line([64, 48, x, 95], fill=(255, 60, 190))
    for i in range(5): d.line([0, 48 + 6 * i * i // 2, 127, 48 + 6 * i * i // 2], fill=(40, 230, 255))
    d.line([60, 40, 68, 40], fill=(255, 255, 255)); d.line([64, 36, 64, 44], fill=(255, 255, 255))
    d.text((4, 4), "FRAGS 12", fill=(255, 255, 255), font=font(MONO, 9))
    return im
def crt_saver():
    im = Image.new("RGB", (128, 96), (0, 0, 8)); d = ImageDraw.Draw(im)
    d.text((64, 40), "JustinCon", fill=(255, 90, 200), font=font(FB, 20), anchor="mm")
    d.text((64, 64), "LAN '99", fill=(60, 225, 255), font=font(MONO, 12), anchor="mm")
    return im
def crt_term():
    im = Image.new("RGB", (128, 96), (0, 10, 4)); d = ImageDraw.Draw(im); f = font(MONO, 10)
    for i, t in enumerate(["C:\\> ping 10.0.0.1", "Reply: time<1ms", "C:\\> net use", "OK", "C:\\> _"]):
        d.text((4, 4 + i * 17), t, fill=(70, 255, 120), font=f)
    return im
T["CRT_GAME"] = crt_game; T["CRT_SAVER"] = crt_saver; T["CRT_TERM"] = crt_term
def poster98():
    im = Image.new("RGB", (64, 96), (20, 8, 40)); d = ImageDraw.Draw(im)
    d.rectangle([0, 0, 63, 95], outline=(255, 70, 200), width=2)
    d.text((32, 22), "Justin", fill=(255, 255, 255), font=font(FB, 14), anchor="mm")
    d.text((32, 38), "Con", fill=(255, 90, 200), font=font(FB, 14), anchor="mm")
    d.text((32, 62), "'98", fill=(60, 225, 255), font=font(FB, 18), anchor="mm")
    d.text((32, 84), "BYOPC", fill=(255, 220, 90), font=font(MONO, 9), anchor="mm")
    return im
T["POSTER98"] = poster98
def newscast():
    im = Image.new("RGB", (128, 64), (10, 20, 60)); d = ImageDraw.Draw(im)
    d.text((64, 20), "WINTER STORM", fill=(255, 255, 255), font=font(FB, 15), anchor="mm")
    d.rectangle([0, 40, 127, 63], fill=(180, 20, 30))
    d.text((64, 51), "STAY INSIDE. PLAY LAN.", fill=(255, 255, 255), font=font(FB, 9), anchor="mm")
    return im
T["NEWSCAST"] = newscast

EMISSIVE = {"STRIP_PINK": 255, "STRIP_CORAL": 255, "STRIP_CYAN": 255, "STRIP_VIOLET": 255, "STRIP_WHITE": 230,
            "NEON_JC": 255, "MATCHBOARD": 240, "STAGEBACK": 200, "CRT_GAME": 220, "CRT_SAVER": 220, "CRT_TERM": 220,
            "LAMP": 255, "NEWSCAST": 220, "EMB_AXE": 140, "EMB_MOON": 140, "TS_AXE": 200, "TS_MOON": 200, "TS_JC": 200}
for h in HANDLES: EMISSIVE[handle_tex(h)] = 230

def to_mip(name, im):
    im = im.convert("RGB"); w, h = im.size
    q = im.quantize(colors=255, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE)
    idx = np.asarray(q, np.uint8)
    pal = q.getpalette()[:255 * 3]; pal += [0] * (255 * 3 - len(pal)); pal += [0, 0, 255]
    mips = [idx] + [np.asarray(q.resize((w // k, h // k), Image.Resampling.NEAREST), np.uint8) for k in (2, 4, 8)]
    offs, pos = [], 40
    for m in mips: offs.append(pos); pos += m.size
    data = struct.pack("<16sII4I", name.encode().ljust(16, b"\0")[:16], w, h, *offs)
    data += b"".join(np.ascontiguousarray(m).tobytes() for m in mips)
    return data + struct.pack("<H", 256) + bytes(pal) + b"\0\0"

def write_wad(path, preview=None):
    sizes, body, entries, pos = {}, b"", [], 12
    for name, fn in T.items():
        im = fn(); sizes[name] = im.size
        assert im.size[0] % 16 == 0 and im.size[1] % 16 == 0, name
        if preview: im.save(os.path.join(preview, name.replace("{", "") + ".png"))
        data = to_mip(name, im)
        entries.append(struct.pack("<IIIbbH16s", pos, len(data), len(data), 0x43, 0, 0, name.encode().ljust(16, b"\0")[:16]))
        body += data; pos += len(data)
    with open(path, "wb") as f:
        f.write(b"WAD3" + struct.pack("<II", len(entries), pos) + body + b"".join(entries))
    return sizes

# ------------------------------------------------------------------ overcast snowstorm skybox
def storm_sky(D):
    D = D / np.linalg.norm(D, axis=-1, keepdims=True)
    e = np.arcsin(np.clip(D[..., 2], -1, 1))
    k = np.clip(e / (math.pi / 2), 0, 1) ** 0.7
    col = np.array([62, 70, 86.0]) * (1 - k[..., None]) + np.array([24, 29, 40.0]) * k[..., None]
    cl = (np.sin(D @ np.array([7.0, 3, 5]) + 1) * np.sin(D @ np.array([-4.0, 9, 2]) + 2) +
          0.5 * np.sin(D @ np.array([13.0, -11, 6])) * np.sin(D @ np.array([5.0, 17, -9])))
    col += np.array([10, 11, 14.0]) * cl[..., None]
    col += np.array([40, 30, 22.0]) * np.exp(-np.maximum(e, 0) / 0.07)[..., None]   # town glow under the clouds
    ground = e < 0.012 + 0.006 * np.sin(5 * np.arctan2(D[..., 1], D[..., 0]))
    col[ground] = np.array([34, 37, 46.0])
    return col

def write_skybox(prefix, N=256):
    for f in ("rt", "bk", "lf", "ft", "up", "dn"):
        j, i = np.mgrid[0:N, 0:N]
        s = (i + 0.5) / N * 2 - 1; t = 1 - (j + 0.5) / N * 2
        col = storm_sky(sky.face_dirs(f, s, t))
        sky.write_tga(f"{prefix}{f}.tga", np.clip(col, 0, 255).astype(np.uint8))

# ------------------------------------------------------------------ snow sprites
def write_spr(path, frames, pal, sprtype=0, texfmt=1):
    h, w = frames[0].shape
    out = struct.pack("<4siiifiiifi", b"IDSP", 2, sprtype, texfmt, math.hypot(w, h) / 2, w, h, len(frames), 0.0, 1)
    out += struct.pack("<h", 256) + bytes(c for p in pal for c in p)
    for f in frames:
        out += struct.pack("<i", 0) + struct.pack("<iiii", -w // 2, h // 2, w, h) + np.clip(f, 0, 255).astype(np.uint8).tobytes()
    open(path, "wb").write(out)

SNOW_PAL = [(int(i * 0.92), int(i * 0.95), i) for i in range(256)]

def gust_profile(N, seed):
    """Wind speed per frame: calm drift + a couple of gusts; integrates to exactly one sprite width."""
    r = np.random.default_rng(seed)
    t = np.arange(N) / N
    g = 0.25 + 1.6 * np.exp(-((t - 0.3) / 0.07) ** 2) + 0.9 * np.exp(-((t - 0.72) / 0.05) ** 2)
    g *= 1 + 0.1 * np.sin(2 * math.pi * t * 3 + r.random())
    return g / g.sum()

def snow_frames(N=64, S=128, seed=3, n_fine=260, n_extra=220):
    r = np.random.default_rng(seed)
    vx = gust_profile(N, seed) * S                      # px per frame (sums to S -> seamless loop)
    cx = np.concatenate([[0], np.cumsum(vx)])
    x0 = r.random(n_fine) * S; y0 = r.random(n_fine) * S
    m = r.choice([1, 1, 1, 2], n_fine)                  # falls m sprite-heights per loop (seamless)
    bright = r.uniform(110, 230, n_fine); jitter = r.random(n_fine) * 2 * math.pi
    ex0 = r.random(n_extra) * S; ey0 = r.random(n_extra) * S
    frames = []
    for f in range(N):
        a = np.zeros((S, S))
        gust = vx[f] / vx.mean()
        def plot(x, y, xp, yp, b, streak):
            if streak and (abs(x - xp) > 1.5):
                n = int(abs(x - xp)) + 1
                for k in range(n):
                    xi, yi = int(xp + (x - xp) * k / n) % S, int(yp + (y - yp) * k / n) % S
                    a[yi, xi] = max(a[yi, xi], b * (0.35 + 0.65 * k / n))
            else:
                a[int(y) % S, int(x) % S] = max(a[int(y) % S, int(x) % S], b)
        for i in range(n_fine):
            sway = 1.5 * math.sin(2 * math.pi * f / N * 2 + jitter[i])
            x = x0[i] + cx[f] + sway; y = y0[i] + m[i] * S * f / N
            xp = x0[i] + cx[max(f - 1, 0)] + sway; yp = y0[i] + m[i] * S * max(f - 1, 0) / N
            plot(x, y, xp, yp, bright[i], gust > 1.6)
        burst = max(0.0, gust - 1.3) / 2.0             # extra flakes only while a gust blows
        if burst > 0:
            k = int(n_extra * min(1, burst))
            for i in range(k):
                x = ex0[i] + cx[f] * 1.3; y = ey0[i] + 2 * S * f / N
                plot(x, y, x - vx[f] * 1.3, y - 2 * S / N, 170, True)
        frames.append(a * EDGE_FADE(S))
    return frames

def EDGE_FADE(S):
    u = np.linspace(-1, 1, S)
    fx = np.clip((1 - np.abs(u)) / 0.35, 0, 1) ** 1.5
    return fx[None, :] * fx[:, None]

def burst_frames(N=64, S=128, seed=11):
    r = np.random.default_rng(seed)
    frames = []
    t = np.arange(N) / N
    env = np.exp(-((t - 0.45) / 0.12) ** 2)             # one heavy burst per loop
    hz = noise(S, S, 10)
    pts = r.random((900, 2)) * S
    for f in range(N):
        a = np.zeros((S, S))
        if env[f] > 0.02:
            yy = np.linspace(0, 1, S)[:, None]
            a += env[f] * 150 * hz * np.clip(1.2 - np.abs(yy - 0.5) * 1.6, 0, 1)   # whiteout haze
            for (x, y) in pts[: int(900 * env[f])]:
                xx = (x + f * 9) % S; yv = (y + f * 4) % S
                for k in range(6):
                    a[int(yv) % S, int(xx - k) % S] = max(a[int(yv) % S, int(xx - k) % S], 200 * env[f])
        frames.append(np.clip(a * EDGE_FADE(S), 0, 255))
    return frames

# ------------------------------------------------------------------ looping sounds (with cue point so GoldSrc loops them)
def write_wav(path, x, sr=22050):
    pcm = (np.clip(x, -1, 1) * 32000).astype("<i2").tobytes()
    fmt = struct.pack("<HHIIHH", 1, 1, sr, sr * 2, 2, 16)
    cue = struct.pack("<I", 1) + struct.pack("<II4sIII", 1, 0, b"data", 0, 0, 0)
    body = b"WAVE" + b"fmt " + struct.pack("<I", 16) + fmt + b"cue " + struct.pack("<I", len(cue)) + cue + \
        b"data" + struct.pack("<I", len(pcm)) + pcm
    open(path, "wb").write(b"RIFF" + struct.pack("<I", len(body)) + body)

def loop_noise(n, sr, lo, hi, seed, fade=0.25):
    from scipy.signal import butter, sosfiltfilt
    r = np.random.default_rng(seed); F = int(fade * sr)
    x = r.standard_normal(n + F)
    sos = butter(4, [lo, hi], btype="band", fs=sr, output="sos")
    y = sosfiltfilt(sos, x)
    w = np.linspace(0, 1, F)
    y[:F] = y[:F] * w + y[n:n + F] * (1 - w)          # crossfade tail into head -> seamless loop
    y = y[:n]
    return y / (np.abs(y).max() + 1e-9)

def make_sounds(dirpath, sr=22050):
    os.makedirs(dirpath, exist_ok=True)
    L = 8.0; n = int(L * sr); t = np.arange(n) / sr
    env = 0.35 + 0.65 * (0.5 + 0.5 * np.sin(2 * np.pi * t / L * 1 - 1.2)) ** 3 + 0.25 * (0.5 + 0.5 * np.sin(2 * np.pi * t / L * 3 + 0.4)) ** 4
    body = loop_noise(n, sr, 60, 700, 1)
    hiss = loop_noise(n, sr, 1200, 5000, 2)
    whistle = loop_noise(n, sr, 650, 820, 3)
    wind = body * env * 0.55 + hiss * env ** 2 * 0.18 + whistle * np.clip(env - 0.6, 0, 1) * 0.5
    write_wav(os.path.join(dirpath, "wind_loop.wav"), wind / np.abs(wind).max() * 0.8, sr)
    L = 4.0; n = int(L * sr); t = np.arange(n) / sr
    hum = sum(a * np.sin(2 * np.pi * f * t) for f, a in ((60, 0.35), (120, 0.3), (180, 0.1), (240, 0.08)))
    blades = 0.12 * np.sin(2 * np.pi * 212.0 * t) + 0.1 * np.sin(2 * np.pi * 213.25 * t)   # two fans beating
    air = loop_noise(n, sr, 300, 2500, 4) * 0.35
    fans = hum + blades + air
    write_wav(os.path.join(dirpath, "fans_loop.wav"), fans / np.abs(fans).max() * 0.6, sr)

# ================================================================== interactive-element assets (v2)
T["SERVERRACK"] = JT["SERVERRACK"]
def breaker():
    im = Image.new("RGB", (64, 64), (110, 112, 118)); d = ImageDraw.Draw(im)
    d.rectangle([2, 2, 61, 61], outline=(60, 62, 66), width=3)
    for i in range(0, 64, 8): d.line([i, 0, i + 8, 8], fill=(230, 190, 30), width=3)      # hazard band
    d.rectangle([22, 20, 42, 50], fill=(40, 40, 44)); d.rectangle([27, 24, 37, 36], fill=(220, 40, 40))   # lever
    d.text((32, 56), "BREAKER", fill=(20, 20, 20), font=font(FB, 8), anchor="mm")
    return im
T["BREAKER"] = breaker
def transformer():
    a = colorize(noise(64, 64, 1) * 0.3 + 0.7, (60, 90, 60), (90, 125, 90))
    im = img(a); d = ImageDraw.Draw(im)
    d.rectangle([8, 18, 56, 44], fill=(240, 200, 30))
    d.text((32, 26), "DANGER", fill=(0, 0, 0), font=font(FB, 9), anchor="mm")
    d.text((32, 37), "HIGH VOLTAGE", fill=(0, 0, 0), font=font(FB, 6), anchor="mm")
    return im
T["TRANSFORMER"] = transformer
def telepad():
    im = Image.new("RGB", (64, 64), (10, 6, 24)); d = ImageDraw.Draw(im)
    for r, c in ((30, (255, 70, 200)), (22, (60, 230, 255)), (14, (255, 70, 200)), (6, (255, 255, 255))):
        d.ellipse([32 - r, 32 - r, 32 + r, 32 + r], outline=c, width=3)
    return im
T["TELEPAD"] = telepad
def vent():
    a = colorize(noise(64, 64, 1) * 0.3 + 0.5, (40, 42, 46), (90, 92, 98))
    a[::8] *= 0.3
    return img(a)
T["VENT"] = vent
def serverdoor():
    a = colorize(noise(64, 64, 2) * 0.2 + 0.8, (70, 72, 82), (110, 112, 125))
    im = img(a); d = ImageDraw.Draw(im)
    d.rectangle([4, 4, 59, 59], outline=(40, 40, 48), width=2)
    d.rectangle([6, 22, 57, 36], fill=(30, 10, 40))
    d.text((32, 29), "SERVER ROOM", fill=(255, 100, 210), font=font(FB, 7), anchor="mm")
    for y in (46, 52): d.line([10, y, 54, y], fill=(50, 50, 58), width=2)
    return im
T["SERVERDOOR"] = serverdoor
def armory():
    im = Image.new("RGB", (64, 64), (30, 30, 36)); d = ImageDraw.Draw(im)
    d.ellipse([14, 10, 50, 46], fill=(200, 20, 30), outline=(255, 120, 120), width=3)
    d.text((32, 55), "ARMORY", fill=(255, 220, 90), font=font(FB, 10), anchor="mm")
    return im
T["ARMORY"] = armory
def charger(kind):
    def f():
        im = Image.new("RGB", (64, 64), (200, 205, 210) if kind == "h" else (40, 70, 140)); d = ImageDraw.Draw(im)
        d.rectangle([3, 3, 60, 60], outline=(70, 70, 80), width=3)
        if kind == "h":
            d.rectangle([27, 14, 37, 46], fill=(40, 180, 80)); d.rectangle([16, 25, 48, 35], fill=(40, 180, 80))   # green plus
            d.text((32, 55), "HEALTH", fill=(30, 30, 30), font=font(FB, 8), anchor="mm")
        else:
            d.polygon([(36, 10), (22, 36), (31, 36), (27, 54), (44, 26), (34, 26)], fill=(255, 220, 60))        # bolt
            d.text((32, 58), "ARMOR", fill=(255, 255, 255), font=font(FB, 7), anchor="mm")
        return im
    return f
T["CHARGER_H"] = charger("h"); T["CHARGER_A"] = charger("a")
T["FANBLADE"] = lambda: img(colorize(noise(32, 32, 1) * 0.3, (30, 24, 40), (70, 60, 90)))
T["ORIGIN"] = lambda: img(np.tile(np.array([255, 140, 0], float), (16, 16, 1)))
def crate():
    a = colorize(noise(64, 64, 0.8), (110, 80, 45), (165, 125, 75))
    a[:4] *= 0.5; a[-4:] *= 0.5; a[:, :4] *= 0.5; a[:, -4:] *= 0.5
    im = img(a); d = ImageDraw.Draw(im); d.line([4, 4, 60, 60], fill=(90, 60, 30), width=4)
    return im
T["CRATE"] = crate
def marquee_frame(k):
    def f():
        W_, H_ = 256, 32
        im = Image.new("RGB", (W_, H_), (12, 4, 24)); d = ImageDraw.Draw(im)
        d.text((W_ // 2, H_ // 2), "BATTLEAXE  vs  CRESCENT   * LIVE *", fill=(255, 240, 255), font=font(FB, 12), anchor="mm")
        cols = [(255, 70, 200), (60, 230, 255)]
        pts = [(x, 2) for x in range(2, W_, 8)] + [(W_ - 3, y) for y in range(2, H_, 8)] + \
              [(x, H_ - 3) for x in range(W_ - 3, 0, -8)] + [(2, y) for y in range(H_ - 3, 0, -8)]
        for i, (x, y) in enumerate(pts):                     # chasing lights: every 10th bulb lit, shifting each frame
            on = (i - k) % 10 < 3
            d.ellipse([x - 2, y - 2, x + 2, y + 2], fill=cols[(i // 10) % 2] if on else (40, 20, 50))
        return im
    return f
for k in range(10):
    T["+%dMARQUEE" % k] = marquee_frame(k)
EMISSIVE.update({"+0MARQUEE": 240, "TELEPAD": 230, "CHARGER_H": 120, "CHARGER_A": 120, "ARMORY": 150, "BREAKER": 60})

def snowfall_frame(k, N=10, S=128, n=110, seed=21):
    """One frame of the continuous snowfall sheet (additive: black = clear).
    Each flake falls 1 px (4 units at scale 4) per frame and fades in/out over its own 10-frame life,
    phases staggered, so the 10-frame loop (1 s in game) has no visible restart."""
    def f():
        r = np.random.default_rng(seed)
        x0 = r.random(n) * S; y0 = r.random(n) * S; ph = r.integers(0, N, n)
        drift = r.choice([0.0, 0.3, 0.5], n); peak = r.uniform(120, 210, n)
        a = np.zeros((S, S))
        for i in range(n):
            age = (k - ph[i]) % N
            b = peak[i] * math.sin(math.pi * (age + 0.5) / N)
            x = int(x0[i] + drift[i] * age) % S; y = int(y0[i] + age) % S
            a[y, x] = max(a[y, x], b)
            for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                yy, xx = (y + dy) % S, (x + dx) % S
                a[yy, xx] = max(a[yy, xx], b * 0.15)
        a = np.stack([a * 0.92, a * 0.95, a], -1)
        return img(a)
    return f
for k in range(10):
    T["+%dSNOWFALL" % k] = snowfall_frame(k)

def steam_frames(N=32, S=64, seed=5):
    r = np.random.default_rng(seed); frames = []
    puffs = [(r.random() * S, r.random(), r.uniform(5, 10)) for _ in range(14)]
    for f in range(N):
        a = np.zeros((S, S)); yy, xx = np.mgrid[0:S, 0:S]
        for (x0, ph, rad) in puffs:
            t = (f / N + ph) % 1.0
            cy = S - t * S; cx = x0 + 6 * math.sin(t * 6 + ph * 10); rr = rad * (0.6 + t)
            a += 110 * (1 - t) * np.exp(-(((xx - cx) ** 2 + (yy - cy) ** 2) / (rr ** 2)))
        frames.append(np.clip(a * EDGE_FADE(S), 0, 255))
    return frames

def make_alarm(path, sr=22050):
    L = 2.0; n = int(L * sr); t = np.arange(n) / sr
    f = np.where((t % 1.0) < 0.5, 880.0, 660.0)                 # two-tone siren, whole number of cycles per tone
    ph = 2 * np.pi * np.cumsum(f) / sr
    x = 0.6 * np.sign(np.sin(ph)) * 0.35 + 0.4 * np.sin(ph)
    env = np.minimum(1, np.minimum((t % 0.5) / 0.01, (0.5 - t % 0.5) / 0.01))
    write_wav(path, x * env * 0.7, sr)
