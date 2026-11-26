"""Shared texture library: noise/colour helpers, base textures (T), WAD3 writer.
Originally the Justin Con texture set; other maps import its helpers and base textures."""
import struct, math
import numpy as np
from PIL import Image, ImageDraw, ImageFont, ImageFilter

rng = np.random.default_rng(42)
from tools.mapkit.paths import FONT_BOLD as FB, FONT_MONO as MONO

def font(path, size): return ImageFont.truetype(path, size)

def noise(w, h, blur=1.0, amp=1.0):
    n = rng.random((h, w))
    img = Image.fromarray((n * 255).astype(np.uint8)).filter(ImageFilter.GaussianBlur(blur))
    a = np.asarray(img).astype(float) / 255
    a = (a - a.min()) / (a.max() - a.min() + 1e-9)
    return a * amp

def colorize(a, c0, c1):
    c0, c1 = np.array(c0, float), np.array(c1, float)
    return (c0 + (c1 - c0) * a[..., None]).clip(0, 255)

def img(arr): return Image.fromarray(np.asarray(arr).clip(0, 255).astype(np.uint8), "RGB")

T = {}


def slabs(c0, c1, step=64):
    def f():
        a = colorize(noise(128, 128, 1.2) * 0.5 + rng.random((128, 128)) * 0.5, c0, c1)
        a[::step, :] *= 0.55; a[:, ::step] *= 0.55
        return img(a)
    return f
T["CONCRETE"] = slabs((95, 95, 92), (150, 148, 142), 128)

def siding_f(c0, c1):
  def siding():
    a = colorize(noise(128, 128, 1.5) * 0.3 + 0.7, c0, c1)
    for y in range(0, 128, 16):
        a[y:y + 2] *= 0.55
        a[y + 2:y + 4] *= 1.08
    return img(a)
  return siding


def brick():
    a = colorize(noise(128, 128, 0.7), (110, 45, 35), (165, 75, 55))
    for r, y in enumerate(range(0, 128, 16)):
        a[y:y + 3] = (150, 145, 135)
        off = 0 if r % 2 == 0 else 16
        for x in range(off, 128, 32): a[y:y + 16, x:x + 3] = (150, 145, 135)
    return img(a)
T["BRICK"] = brick
T["TRIM"] = lambda: img(colorize(noise(64, 64, 1) * 0.25 + 0.75, (200, 200, 195), (240, 240, 235)))

def wallpaper():
    a = colorize(noise(128, 128, 1) * 0.2 + 0.8, (170, 150, 115), (215, 195, 160))
    for x in range(0, 128, 32):
        a[:, x:x + 3] *= 0.85; a[:, x + 14:x + 16] *= 1.05
    return img(a)
T["WALLPAPER"] = wallpaper

def planks(c0, c1, vertical=False, size=128, gap=16):
    def f():
        n = noise(size, size // 4, 0.6)
        n = np.asarray(Image.fromarray((n * 255).astype(np.uint8)).resize((size, size))) / 255.0
        a = colorize(n * 0.6 + rng.random((size, size)) * 0.15, c0, c1)
        for i, y in enumerate(range(0, size, gap)):
            a[y:y + 2] *= 0.45
            j = int(rng.integers(0, size))
            a[y:y + gap, j:j + 2] *= 0.5
            a[y:y + gap] *= 0.85 + 0.3 * rng.random()
        im = img(a)
        return im.rotate(90) if vertical else im
    return f
T["WOODFLOOR"] = planks((95, 60, 30), (165, 115, 65))
T["BASEWALL"] = planks((70, 45, 25), (135, 90, 50), vertical=True, gap=32)
T["WOODFENCE"] = planks((90, 70, 50), (150, 120, 85), vertical=True, gap=16)
T["STEP"] = planks((80, 55, 30), (140, 100, 60))
T["CEILING"] = lambda: img(colorize(noise(64, 64, 0.8) * 0.15 + 0.85, (195, 192, 185), (230, 228, 222)))

def ceiltile():
    a = colorize(rng.random((128, 128)) * 0.3 + 0.7, (175, 172, 165), (215, 212, 205))
    for _ in range(300):
        x, y = rng.integers(0, 128, 2); a[y, x] *= 0.7
    a[::64] = (120, 120, 120); a[:, ::64] = (120, 120, 120)
    return img(a)
T["CEILTILE"] = ceiltile
T["CARPET"] = lambda: img(colorize(rng.random((64, 64)) * 0.7 + noise(64, 64, 2) * 0.3, (30, 35, 55), (75, 80, 105)))
T["TABLE"] = lambda: img(colorize(noise(64, 64, 2) * 0.2 + 0.8, (170, 170, 165), (205, 205, 200)))
T["METAL"] = lambda: img(colorize(noise(64, 64, 0.7) * 0.5 + 0.5, (45, 47, 52), (110, 112, 120)))
T["TIRE"] = lambda: img(colorize(rng.random((32, 32)), (12, 12, 12), (35, 35, 35)))
T["COUCH"] = lambda: img(colorize(rng.random((64, 64)) * 0.4 + noise(64, 64, 1) * 0.6, (70, 45, 35), (120, 80, 60)))
T["COUNTER"] = lambda: img(colorize(rng.random((64, 64)) ** 3, (40, 40, 45), (190, 185, 180)))
T["TVBLACK"] = lambda: img(colorize(noise(32, 32, 1) * 0.3, (8, 8, 10), (40, 40, 45)))
T["GLASS"] = lambda: img(colorize(noise(64, 64, 4) * 0.4 + 0.6, (120, 150, 170), (190, 215, 230)))
T["CARGLASS"] = lambda: img(colorize(noise(64, 64, 4), (15, 20, 30), (60, 75, 95)))

def fridge():
    a = colorize(noise(64, 64, 2) * 0.1 + 0.9, (200, 205, 205), (235, 238, 238))
    a[20:22] *= 0.6; a[28:52, 54:57] = (120, 120, 125)
    return img(a)
T["FRIDGE"] = fridge

def carpaint(c):
    def f():
        a = colorize(noise(64, 64, 3) * 0.4 + 0.6, tuple(x * 0.5 for x in c), c)
        a[30:32] *= 0.6
        return img(a)
    return f
T["CARRED"] = carpaint((170, 25, 25))
T["CARBLUE"] = carpaint((30, 70, 170))
T["CARSILVER"] = carpaint((170, 175, 180))


def lamp():
    a = colorize(noise(32, 32, 2) * 0.2 + 0.8, (230, 200, 140), (255, 240, 200))
    return img(a)
T["LAMP"] = lamp


def crt():
    a = colorize(noise(64, 64, 2) * 0.2 + 0.8, (170, 160, 135), (215, 205, 180))
    a[54:58] *= 0.7
    return img(a)
T["CRT"] = crt


T["sky"] = lambda: img(np.full((64, 64, 3), (10, 12, 30)))
T["AAATRIGGER"] = lambda: img(np.full((16, 16, 3), (255, 140, 0)))


def can(c0, c1):
    def f():
        im = Image.new("RGB", (16, 16), c0)
        d = ImageDraw.Draw(im)
        d.rectangle([0, 6, 15, 9], fill=c1)
        d.rectangle([0, 0, 15, 1], fill=(180, 180, 185))
        return im
    return f

def serverrack():
    im = Image.new("RGB", (64, 128), (15, 15, 18))
    d = ImageDraw.Draw(im)
    for y in range(4, 128, 12):
        d.rectangle([4, y, 59, y + 9], fill=(35, 35, 40))
        for x in range(8, 30, 5):
            c = [(0, 255, 80), (0, 160, 255), (255, 170, 0)][int(rng.integers(0, 3))]
            d.point((x, y + 4), fill=c); d.point((x + 1, y + 4), fill=c)
    return im
T["SERVERRACK"] = serverrack

# glowing textures (0..255): zone-built maps light them with tools.mapkit.texlights (SDHLT's info_texlights)
EMISSIVE = {"LAMP": 255, "SERVERRACK": 170}

def to_mip(name, im):
    im = im.convert("RGB")
    w, h = im.size
    assert w % 16 == 0 and h % 16 == 0, name
    q = im.quantize(colors=255, method=Image.Quantize.MEDIANCUT, dither=Image.Dither.NONE)
    pal = q.getpalette()[:255 * 3]
    pal += [0] * (255 * 3 - len(pal))
    pal += [0, 0, 255]
    idx = np.asarray(q, np.uint8)
    mips = [idx]
    for k in (2, 4, 8):
        mips.append(np.asarray(q.resize((w // k, h // k), Image.Resampling.NEAREST), np.uint8))
    hdr = 40
    offs, pos = [], hdr
    for m in mips: offs.append(pos); pos += m.size
    data = struct.pack("<16sII4I", name.encode().ljust(16, b"\0")[:16], w, h, *offs)
    data += b"".join(np.ascontiguousarray(m).tobytes() for m in mips)
    data += struct.pack("<H", 256) + bytes(pal) + b"\0\0"
    return data

def write_wad(path, preview_dir=None):
    lumps = []
    sizes = {}
    for name, fn in T.items():
        im = fn()
        sizes[name] = im.size
        if preview_dir: im.resize((im.size[0] * 2, im.size[1] * 2), Image.Resampling.NEAREST).save(f"{preview_dir}/{name}.png")
        lumps.append((name, to_mip(name, im)))
    body, entries, pos = b"", [], 12
    for name, data in lumps:
        entries.append(struct.pack("<IIIbbH16s", pos, len(data), len(data), 0x43, 0, 0, name.encode().ljust(16, b"\0")[:16]))
        body += data; pos += len(data)
    with open(path, "wb") as f:
        f.write(b"WAD3" + struct.pack("<II", len(lumps), pos) + body + b"".join(entries))
    return sizes
