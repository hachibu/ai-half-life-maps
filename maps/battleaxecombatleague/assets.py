"""Original textures for BattleAxe Combat League (BCL) Fight Night (fictional promotion, invented names/sponsors)."""
import os, sys, struct
import numpy as np
from PIL import Image, ImageDraw, ImageFilter
from tools.assets.library import noise, colorize, img, font, FB, MONO, rng, T as JT

T = {}
for k in ("CONCRETE", "METAL", "TRIM", "LAMP", "TABLE", "TVBLACK", "CEILTILE", "STEP", "COUCH", "sky", "AAATRIGGER"):
    T[k] = JT[k]

def floorblack():
    return img(colorize(rng.random((64, 64)) * 0.4 + noise(64, 64, 2) * 0.6, (10, 10, 12), (35, 35, 40)))
T["ARENAFLOOR"] = floorblack
T["APRON"] = lambda: img(colorize(noise(64, 64, 2) * 0.3, (8, 8, 10), (40, 40, 45)))
T["CAGEPAD"] = lambda: img(colorize(noise(32, 32, 1) * 0.4, (10, 10, 12), (45, 45, 50)))

def canvas():
    W = 256
    a = colorize(noise(W, W, 3) * 0.12 + 0.88, (200, 200, 205), (238, 238, 240))
    im = img(a); d = ImageDraw.Draw(im)
    c = W / 2
    import math
    pts = [(c + 118 * math.cos(math.radians(22.5 + 45 * i)), c + 118 * math.sin(math.radians(22.5 + 45 * i))) for i in range(8)]
    d.polygon(pts, outline=(180, 20, 30))
    d.ellipse([c - 56, c - 56, c + 56, c + 56], fill=(20, 20, 26), outline=(200, 30, 40), width=4)
    # crossed battle-axes emblem behind the letters
    for sgn in (1, -1):
        d.line([c - 34 * sgn, c + 34, c + 30 * sgn, c - 30], fill=(120, 90, 60), width=4)
        hx, hy = c + 30 * sgn, c - 30
        d.polygon([(hx, hy), (hx + 14 * sgn, hy - 14), (hx + 22 * sgn, hy + 2), (hx + 6 * sgn, hy + 10)], fill=(190, 195, 205))
    d.text((c, c - 4), "BCL", fill=(245, 245, 245), font=font(FB, 34), anchor="mm")
    d.text((c, c + 30), "FIGHT NIGHT", fill=(220, 40, 50), font=font(FB, 11), anchor="mm")
    d.text((c, 40), "BATTLEAXE COMBAT LEAGUE", fill=(60, 60, 70), font=font(FB, 12), anchor="mm")
    d.text((c, W - 40), "KRAKEN COLA", fill=(30, 90, 170), font=font(FB, 14), anchor="mm")
    lay = Image.new("RGB", (120, 24), (0, 0, 0)); ImageDraw.Draw(lay).text((60, 12), "ZORP ENERGY", fill=(40, 170, 60), font=font(FB, 13), anchor="mm")
    msk = lay.convert("L").point(lambda v: 255 if v > 40 else 0)
    for ang, pos in ((90, (16, int(c - 60))), (-90, (W - 40, int(c - 60)))):
        im.paste(lay.rotate(ang, expand=True), pos, msk.rotate(ang, expand=True))
    return im
T["CANVAS"] = canvas

def fence():
    w = h = 64
    yy, xx = np.mgrid[0:h, 0:w]
    a = np.zeros((h, w, 3)); a[:] = (0, 0, 255)
    wire = ((xx + yy) % 16 < 2) | ((xx - yy) % 16 < 2)
    shade = (110 + 60 * ((xx + yy) % 16) / 16)
    a[wire] = np.stack([shade, shade, shade + 8], -1)[wire]
    return img(a)
T["{CAGEFENCE"] = fence

def person_row(width, seed, rows=1, phones=0.08, signs=0.05):
    r = np.random.default_rng(seed)
    im = Image.new("RGB", (width, 64), (14, 14, 20))
    d = ImageDraw.Draw(im)
    shirts = [(200, 30, 30), (30, 60, 170), (230, 230, 230), (20, 20, 20), (250, 200, 30), (40, 140, 60),
              (120, 30, 120), (90, 90, 95), (230, 110, 20), (60, 170, 200)]
    skins = [(240, 200, 170), (210, 160, 120), (170, 115, 80), (120, 80, 55), (90, 60, 40)]
    hairs = [(30, 20, 15), (70, 45, 25), (200, 170, 100), (15, 15, 15), (120, 120, 120)]
    x = 2
    while x < width - 12:
        pw = int(r.integers(14, 19))
        cx = x + pw // 2
        top = int(r.integers(14, 22))
        sc = shirts[int(r.integers(len(shirts)))]; sk = skins[int(r.integers(len(skins)))]
        d.rounded_rectangle([x + 1, top + 12, x + pw - 1, 64], 4, fill=sc)
        d.ellipse([cx - 5, top, cx + 5, top + 12], fill=sk)
        d.chord([cx - 5, top - 1, cx + 5, top + 8], 180, 360, fill=hairs[int(r.integers(len(hairs)))])
        q = r.random()
        if q < phones:
            d.line([cx + 4, top + 16, cx + 7, top - 2], fill=sc, width=3)
            d.rectangle([cx + 5, top - 8, cx + 9, top - 1], fill=(255, 255, 240))
        elif q < phones + signs:
            d.line([cx, top + 14, cx, top - 6], fill=(200, 200, 200))
            d.rectangle([cx - 9, top - 16, cx + 9, top - 6], fill=(250, 250, 250))
            d.line([cx - 6, top - 11, cx + 6, top - 11], fill=(200, 20, 20), width=2)
        elif q < phones + signs + 0.12:
            d.line([x + 2, top + 16, x - 2, top - 4], fill=sk, width=3)
            d.line([x + pw - 2, top + 16, x + pw + 2, top - 4], fill=sk, width=3)
        x += pw
    return im
T["CROWD1"] = lambda: person_row(256, 1).resize((128, 32), Image.Resampling.LANCZOS)
T["CROWD2"] = lambda: person_row(256, 2, phones=0.15).resize((128, 32), Image.Resampling.LANCZOS)
T["CROWD3"] = lambda: person_row(256, 3, signs=0.12).resize((128, 32), Image.Resampling.LANCZOS)

def crowdtop():
    """Heads seen from above; drawn for scale 4 (1 px = 4 units, a head ~ 8 units)."""
    r = np.random.default_rng(9)
    a = np.zeros((64, 64, 3)); a[:] = (16, 16, 22)
    cols = np.array([(30, 20, 15), (70, 45, 25), (200, 170, 100), (15, 15, 15), (160, 115, 80), (120, 120, 120)])
    for y in range(0, 64, 3):
        for x in range(int(r.integers(0, 3)), 64, 3):
            if r.random() < 0.8:
                a[y:y + 2, x:x + 2] = cols[int(r.integers(len(cols)))]
    return img(a)
T["CROWDTOP"] = crowdtop
T["SEATS"] = lambda: img(colorize(noise(64, 64, 1) * 0.3, (20, 30, 70), (40, 55, 110)))
T["STANDS"] = lambda: img(colorize(rng.random((64, 64)) * 0.5 + noise(64, 64, 1) * 0.5, (55, 55, 60), (95, 95, 100)))
def stadwall():
    a = colorize(noise(128, 128, 1) * 0.3, (30, 32, 40), (60, 62, 75))
    a[::32] *= 0.6; a[:, ::64] *= 0.6
    return img(a)
T["STADWALL"] = stadwall

def ribbon(text, fg, bg, accent):
    def f():
        im = Image.new("RGB", (256, 32), bg); d = ImageDraw.Draw(im)
        d.rectangle([0, 0, 255, 2], fill=accent); d.rectangle([0, 29, 255, 31], fill=accent)
        d.text((128, 16), text, fill=fg, font=font(FB, 15), anchor="mm")
        a = np.asarray(im).astype(float)
        a[:, ::2] *= 0.82; a[::2] *= 0.9          # LED pixel grid
        return img(a)
    return f
T["RIBBON1"] = ribbon("LIVE ON PAY-PER-VIEW", (255, 255, 255), (150, 10, 20), (255, 200, 40))
T["RIBBON2"] = ribbon("BCL FIGHT NIGHT 7", (255, 220, 60), (10, 10, 30), (220, 30, 40))
T["RIBBON3"] = ribbon("KRAKEN COLA  *  ZORP ENERGY", (255, 255, 255), (20, 60, 140), (60, 220, 90))

def jumbo1():
    im = Image.new("RGB", (128, 80), (8, 8, 14)); d = ImageDraw.Draw(im)
    d.rectangle([0, 0, 63, 80], fill=(110, 10, 20)); d.rectangle([64, 0, 127, 80], fill=(10, 30, 110))
    d.text((64, 9), "MAIN EVENT", fill=(255, 220, 60), font=font(FB, 10), anchor="mm")
    for cx, nm, rec in ((32, "MORROW", "22-1"), (96, "REYES", "19-2")):
        d.ellipse([cx - 11, 20, cx + 11, 42], fill=(25, 25, 30))
        d.rectangle([cx - 18, 42, cx + 18, 58], fill=(25, 25, 30))
        d.text((cx, 64), nm, fill=(255, 255, 255), font=font(FB, 10), anchor="mm")
        d.text((cx, 74), rec, fill=(220, 220, 220), font=font(MONO, 8), anchor="mm")
    d.text((64, 40), "VS", fill=(255, 255, 255), font=font(FB, 14), anchor="mm")
    return im
def jumbo2():
    im = Image.new("RGB", (128, 80), (6, 6, 12)); d = ImageDraw.Draw(im)
    d.text((64, 14), "ROUND 3", fill=(255, 255, 255), font=font(FB, 16), anchor="mm")
    d.text((64, 38), "4:32", fill=(255, 210, 40), font=font(FB, 22), anchor="mm")
    d.rectangle([8, 56, 120, 74], fill=(150, 10, 20))
    d.text((64, 65), "LIVE PAY-PER-VIEW", fill=(255, 255, 255), font=font(FB, 9), anchor="mm")
    return im
T["JUMBO1"] = jumbo1
T["JUMBO2"] = jumbo2

def walkout():
    im = Image.new("RGB", (256, 128), (0, 0, 0)); d = ImageDraw.Draw(im)
    for y in range(128):
        t = y / 127
        d.line([0, y, 255, y], fill=(int(40 + 200 * t), int(10 + 60 * t ** 2), int(30 * (1 - t))))
    g = Image.new("RGB", (256, 128)); ImageDraw.Draw(g).text((128, 50), "FIGHT NIGHT", fill=(255, 120, 30), font=font(FB, 38), anchor="mm")
    g = g.filter(ImageFilter.GaussianBlur(5))
    im = Image.fromarray(np.clip(np.asarray(im).astype(int) + np.asarray(g).astype(int) * 2, 0, 255).astype(np.uint8))
    d = ImageDraw.Draw(im)
    d.text((128, 50), "FIGHT NIGHT", fill=(255, 245, 220), font=font(FB, 38), anchor="mm")
    d.text((128, 88), "BATTLEAXE COMBAT LEAGUE", fill=(255, 255, 255), font=font(FB, 13), anchor="mm")
    d.text((128, 110), "MORROW  vs  REYES", fill=(255, 210, 60), font=font(FB, 11), anchor="mm")
    return im
T["WALKOUT"] = walkout

def broadcast():
    im = Image.new("RGB", (64, 48), (10, 10, 14)); d = ImageDraw.Draw(im)
    d.rectangle([0, 0, 63, 34], fill=(40, 40, 46))
    d.polygon([(32 + 22 * np.cos(np.radians(22.5 + 45 * i)), 18 + 13 * np.sin(np.radians(22.5 + 45 * i))) for i in range(8)], fill=(210, 210, 215), outline=(20, 20, 20))
    d.rectangle([25, 12, 28, 22], fill=(170, 20, 20)); d.rectangle([36, 13, 39, 23], fill=(20, 40, 170))
    d.rectangle([0, 36, 63, 47], fill=(150, 10, 20))
    d.text((32, 41), "LIVE  R3 4:32", fill=(255, 255, 255), font=font(MONO, 7), anchor="mm")
    return im
T["BROADCAST"] = broadcast

def press():
    im = Image.new("RGB", (128, 32), (15, 15, 18)); d = ImageDraw.Draw(im)
    d.text((64, 16), "BCL  MEDIA", fill=(200, 200, 210), font=font(FB, 12), anchor="mm")
    return im
T["PRESS"] = press

def lightbank():
    im = Image.new("RGB", (64, 32), (20, 20, 24)); d = ImageDraw.Draw(im)
    for x in range(4, 64, 15):
        for y in (3, 17):
            d.ellipse([x, y, x + 11, y + 11], fill=(255, 250, 230))
    return im
T["LIGHTBANK"] = lightbank

def truss():
    im = Image.new("RGB", (32, 32), (0, 0, 0)); d = ImageDraw.Draw(im)
    d.rectangle([0, 0, 31, 31], fill=(40, 40, 44))
    d.rectangle([0, 0, 31, 3], fill=(150, 150, 160)); d.rectangle([0, 28, 31, 31], fill=(150, 150, 160))
    d.line([0, 3, 31, 28], fill=(130, 130, 140), width=2); d.line([0, 28, 31, 3], fill=(130, 130, 140), width=2)
    return im
T["TRUSS"] = truss

def beam():
    """Smooth, untextured glow (any repeat pattern would show as stripes on a big cone)."""
    a = np.zeros((16, 16, 3)); a[:] = (255, 245, 225)
    return img(a)
T["LIGHTBEAM"] = beam

def lockers(c0, c1):
    def f():
        a = colorize(noise(64, 128, 1) * 0.3 + 0.7, c0, c1)
        a[:, ::32] *= 0.4; a[:, 1::32] *= 0.6
        for x in range(0, 64, 32):
            for y in (12, 18, 24): a[y:y + 2, x + 8:x + 24] *= 0.45
            a[60:66, x + 26:x + 28] = (220, 220, 220)
        return img(a)
    return f
T["LOCKER_R"] = lockers((110, 15, 20), (170, 30, 35))
T["LOCKER_B"] = lockers((15, 35, 110), (35, 65, 170))
T["CORNER_R"] = lambda: _plate("RED CORNER", (255, 255, 255), (150, 10, 20))
T["CORNER_B"] = lambda: _plate("BLUE CORNER", (255, 255, 255), (15, 40, 150))
def _plate(text, fg, bg):
    im = Image.new("RGB", (128, 32), bg)
    ImageDraw.Draw(im).text((64, 16), text, fill=fg, font=font(FB, 15), anchor="mm")
    return im
T["TUNNELWALL"] = lambda: img(colorize(noise(64, 64, 1) * 0.3, (18, 18, 24), (45, 45, 55)))
T["STAGE"] = lambda: img(colorize(noise(64, 64, 3) * 0.3 + rng.random((64, 64)) * 0.1, (10, 10, 12), (40, 40, 48)))
T["CAMERA"] = lambda: img(colorize(noise(32, 32, 1) * 0.4, (20, 20, 22), (60, 60, 66)))

EMISSIVE = {"RIBBON1": 230, "RIBBON2": 230, "RIBBON3": 230, "JUMBO1": 240, "JUMBO2": 240, "WALKOUT": 240,
            "BROADCAST": 220, "LIGHTBANK": 255, "LAMP": 255, "LIGHTBEAM": 255, "CORNER_R": 200, "CORNER_B": 200}

def to_mip(name, im):
    im = im.convert("RGB"); w, h = im.size
    arr = np.asarray(im)
    if name.startswith("{"):
        mask = np.all(arr == (0, 0, 255), -1)
        base = arr.copy(); base[mask] = 0
        q = Image.fromarray(base).quantize(colors=255, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE)
        idx = np.asarray(q, np.uint8).copy(); idx[mask] = 255
    else:
        q = im.quantize(colors=255, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE)
        idx = np.asarray(q, np.uint8)
    pal = q.getpalette()[:255 * 3]; pal += [0] * (255 * 3 - len(pal)); pal += [0, 0, 255]
    mips = [idx] + [idx[::k, ::k] for k in (2, 4, 8)]
    offs, pos = [], 40
    for m in mips: offs.append(pos); pos += m.size
    data = struct.pack("<16sII4I", name.encode().ljust(16, b"\0")[:16], w, h, *offs)
    data += b"".join(np.ascontiguousarray(m).tobytes() for m in mips)
    return data + struct.pack("<H", 256) + bytes(pal) + b"\0\0"

def write_wad(path):
    sizes, body, entries, pos = {}, b"", [], 12
    for name, fn in T.items():
        im = fn(); sizes[name] = im.size
        data = to_mip(name, im)
        entries.append(struct.pack("<IIIbbH16s", pos, len(data), len(data), 0x43, 0, 0, name.encode().ljust(16, b"\0")[:16]))
        body += data; pos += len(data)
    with open(path, "wb") as f:
        f.write(b"WAD3" + struct.pack("<II", len(entries), pos) + body + b"".join(entries))
    return sizes
