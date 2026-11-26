"""Shared texture library: noise/colour helpers, base textures (T), WAD3 writer.
Originally the Justin Con texture set; other maps import its helpers and base textures."""
import struct, math
import numpy as np
from PIL import Image, ImageDraw, ImageFont, ImageFilter

rng = np.random.default_rng(42)
from tools.paths import FONT_BOLD as FB, FONT_MONO as MONO

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

T["GRASS"] = lambda: img(colorize(noise(128, 128, 0.8) * 0.6 + rng.random((128, 128)) * 0.4, (25, 60, 20), (80, 130, 50)))

def asphalt():
    a = colorize(rng.random((128, 128)) * 0.6 + noise(128, 128, 3) * 0.4, (35, 35, 38), (85, 85, 90))
    return img(a)
T["ASPHALT"] = asphalt

def slabs(c0, c1, step=64):
    def f():
        a = colorize(noise(128, 128, 1.2) * 0.5 + rng.random((128, 128)) * 0.5, c0, c1)
        a[::step, :] *= 0.55; a[:, ::step] *= 0.55
        return img(a)
    return f
T["SIDEWALK"] = slabs((120, 118, 112), (175, 172, 165))
T["CONCRETE"] = slabs((95, 95, 92), (150, 148, 142), 128)
T["STREETLINE"] = lambda: img(colorize(rng.random((64, 64)), (190, 160, 30), (240, 210, 60)))

def siding_f(c0, c1):
  def siding():
    a = colorize(noise(128, 128, 1.5) * 0.3 + 0.7, c0, c1)
    for y in range(0, 128, 16):
        a[y:y + 2] *= 0.55
        a[y + 2:y + 4] *= 1.08
    return img(a)
  return siding
T["SIDING"] = siding_f((120, 140, 160), (175, 195, 210))
T["SIDING_G"] = siding_f((95, 120, 100), (150, 175, 150))
T["SIDING_Y"] = siding_f((170, 155, 105), (225, 210, 150))

def shingle():
    a = colorize(rng.random((128, 128)) * 0.5 + noise(128, 128, 1) * 0.5, (35, 33, 38), (85, 80, 85))
    for r, y in enumerate(range(0, 128, 16)):
        a[y:y + 2] *= 0.4
        off = 0 if r % 2 == 0 else 16
        for x in range(off, 128, 32): a[y:y + 16, x:x + 2] *= 0.5
    return img(a)
T["SHINGLE"] = shingle

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
T["HEDGE"] = lambda: img(colorize(rng.random((64, 64)) * 0.5 + noise(64, 64, 1) * 0.5, (10, 35, 12), (45, 95, 40)))
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

def neighbor():
    a = colorize(noise(128, 128, 1) * 0.2 + 0.8, (120, 105, 85), (165, 150, 125))
    for y in range(0, 128, 16): a[y:y + 2] *= 0.6
    return img(a)
T["NEIGHBOR"] = neighbor

def nwindow():
    im = Image.new("RGB", (64, 64), (255, 210, 120))
    d = ImageDraw.Draw(im)
    d.rectangle([0, 0, 63, 63], outline=(240, 240, 235), width=4)
    d.line([32, 0, 32, 63], fill=(240, 240, 235), width=3)
    d.line([0, 32, 63, 32], fill=(240, 240, 235), width=3)
    a = np.asarray(im).astype(float)
    a[4:60, 4:60] *= (0.85 + 0.15 * noise(56, 56, 6))[..., None]
    return img(a)
T["NWINDOW"] = nwindow

def lamp():
    a = colorize(noise(32, 32, 2) * 0.2 + 0.8, (230, 200, 140), (255, 240, 200))
    return img(a)
T["LAMP"] = lamp

def pccase():
    im = Image.new("RGB", (64, 64), (18, 18, 22))
    d = ImageDraw.Draw(im)
    cols = [(255, 0, 120), (0, 200, 255), (120, 255, 60), (255, 160, 0)]
    for i, c in enumerate(cols):
        d.rectangle([4, 6 + i * 14, 59, 10 + i * 14], fill=c)
    d.rectangle([0, 0, 63, 63], outline=(60, 60, 70), width=2)
    return im
T["PCCASE"] = pccase

def screen_game():
    im = Image.new("RGB", (64, 48))
    d = ImageDraw.Draw(im)
    for y in range(24): d.line([0, y, 63, y], fill=(40 + y * 4, 90 + y * 4, 170 + y * 2))
    d.rectangle([0, 28, 63, 47], fill=(70, 120, 50))
    d.polygon([(0, 30), (14, 18), (26, 30)], fill=(90, 95, 105))
    d.polygon([(30, 30), (46, 14), (63, 30)], fill=(100, 105, 115))
    d.rectangle([40, 32, 52, 44], fill=(150, 90, 50))
    d.line([29, 22, 35, 22], fill=(255, 255, 255)); d.line([32, 19, 32, 25], fill=(255, 255, 255))
    d.rectangle([2, 42, 18, 45], fill=(220, 40, 40))
    return im
def screen_term():
    im = Image.new("RGB", (64, 48), (5, 12, 5))
    d = ImageDraw.Draw(im)
    f = font(MONO, 7)
    for i, t in enumerate(["> ping justin", "64 bytes ok", "> join lan", "players: 23", "> gg"]):
        d.text((2, 1 + i * 9), t, fill=(60, 255, 90), font=f)
    return im
def screen_score():
    im = Image.new("RGB", (64, 48), (15, 15, 35))
    d = ImageDraw.Draw(im)
    f = font(MONO, 7)
    d.rectangle([0, 0, 63, 9], fill=(200, 60, 40))
    d.text((2, 0), "SCORES", fill=(255, 255, 255), font=f)
    for i, (n, s) in enumerate([("JUSTIN", 42), ("MOM", 3), ("GIBLET", 17), ("KEVLAR", 9)]):
        d.text((2, 11 + i * 9), "%-6s %2d" % (n, s), fill=(220, 220, 255), font=f)
    return im
T["SCREEN1"] = screen_game
T["SCREEN2"] = screen_term
T["SCREEN3"] = screen_score

def screentv():
    im = Image.new("RGB", (128, 64))
    d = ImageDraw.Draw(im)
    for y in range(64): d.line([0, y, 127, y], fill=(20 + y, 10, 60 + y * 2))
    d.text((64, 20), "JUSTIN CON", fill=(255, 230, 80), font=font(FB, 17), anchor="mm")
    d.text((64, 44), "LOBBY  23/24", fill=(180, 230, 255), font=font(MONO, 10), anchor="mm")
    return im
T["SCREENTV"] = screentv

def banner_f(text, glow, border, bg=(12, 8, 24), size=40):
    def f():
        im = Image.new("RGB", (256, 64), bg)
        d = ImageDraw.Draw(im)
        for x in range(0, 256, 8):
            d.line([x, 0, x + 32, 64], fill=tuple(min(255, c + 13) for c in bg))
        g = Image.new("RGB", (256, 64), (0, 0, 0))
        ImageDraw.Draw(g).text((128, 31), text, fill=glow, font=font(FB, size), anchor="mm")
        g = g.filter(ImageFilter.GaussianBlur(4))
        im = Image.fromarray(np.clip(np.asarray(im).astype(int) + np.asarray(g).astype(int) * 2, 0, 255).astype(np.uint8))
        d = ImageDraw.Draw(im)
        d.text((128, 31), text, fill=tuple(min(255, 200 + c // 4) for c in glow), font=font(FB, size), anchor="mm")
        d.rectangle([2, 2, 253, 61], outline=border, width=2)
        return im
    return f
T["BANNER"] = banner_f("JUSTIN CON", (255, 40, 200), (0, 220, 255))
T["BANNER_B"] = banner_f("BUNKER CON", (40, 255, 90), (255, 255, 255), bg=(5, 18, 8))
T["BANNER_R"] = banner_f("RESPAWN CON", (255, 140, 20), (0, 230, 210), bg=(20, 10, 4), size=44)

def poster_f(title, l1, l2, l3, bg, fg, accent):
    def f():
        im = Image.new("RGB", (128, 80), bg)
        d = ImageDraw.Draw(im)
        d.rectangle([0, 0, 127, 79], outline=fg, width=3)
        d.text((64, 18), title, fill=fg, font=font(FB, 20 if len(title) <= 10 else 17), anchor="mm")
        d.text((64, 38), l1, fill=accent, font=font(FB, 16), anchor="mm")
        d.text((64, 56), l2, fill=fg, font=font(FB, 9), anchor="mm")
        d.text((64, 68), l3, fill=fg, font=font(FB, 8), anchor="mm")
        return im
    return f
T["POSTER"] = poster_f("JUSTIN CON", "2026", "BRING YOUR OWN PC", "NO SLEEP. NO MERCY.", (250, 200, 30), (20, 20, 20), (200, 30, 30))
T["POSTER_B"] = poster_f("BUNKER CON", "FINALS", "WINNER STAYS ON", "LOSER BUYS PIZZA", (15, 30, 20), (230, 255, 230), (60, 255, 100))
T["POSTER_R"] = poster_f("RESPAWN CON", "RETRO", "CRT MONITORS ONLY", "BLOW ON THE CART", (30, 60, 70), (255, 235, 200), (255, 150, 40))

def lawnsign_f(title, color):
    def f():
        im = Image.new("RGB", (128, 64), (245, 245, 240))
        d = ImageDraw.Draw(im)
        d.rectangle([0, 0, 127, 63], outline=color, width=3)
        d.text((64, 16), title, fill=color, font=font(FB, 19), anchor="mm")
        d.text((64, 36), "LAN PARTY TONIGHT", fill=(200, 30, 30), font=font(FB, 10), anchor="mm")
        d.text((58, 51), "BASEMENT", fill=(20, 20, 20), font=font(FB, 10), anchor="mm")
        d.polygon([(92, 46), (104, 46), (98, 58)], fill=(20, 20, 20))
        return im
    return f
T["LAWNSIGN"] = lawnsign_f("JUSTIN CON", (30, 30, 160))
T["LAWN_B"] = lawnsign_f("BUNKER CON", (20, 120, 40))
T["LAWN_R"] = lawnsign_f("RESPAWN CON", (220, 110, 10))

def plate_f(text, fg, bg=(30, 30, 40), size=11, w=128, h=32):
    def f():
        im = Image.new("RGB", (w, h), bg)
        ImageDraw.Draw(im).text((w // 2, h // 2), text, fill=fg, font=font(FB, size), anchor="mm")
        return im
    return f
T["REGSIGN"] = plate_f("JUSTIN CON CHECK-IN", (120, 255, 200))
T["REG_B"] = plate_f("BUNKER CON SIGN-UPS", (90, 255, 120))
T["REG_R"] = plate_f("RESPAWN CON ARRIVALS", (255, 170, 60))
T["TS_JUSTIN"] = plate_f("TUNNEL TO JUSTIN CON", (255, 120, 220), bg=(15, 10, 20), size=10)
T["TS_BUNKER"] = plate_f("TUNNEL TO BUNKER CON", (90, 255, 120), bg=(10, 20, 12), size=10)
T["TS_RESPAWN"] = plate_f("TUNNEL TO RESPAWN CON", (255, 170, 60), bg=(20, 14, 8), size=10)

def bracket():
    im = Image.new("RGB", (128, 64), (8, 14, 10))
    d = ImageDraw.Draw(im); f7 = font(MONO, 7)
    d.text((64, 5), "BUNKER CON BRACKET", fill=(90, 255, 120), font=f7, anchor="mm")
    names = ["FRAGTACO", "JUSTIN", "LAGWIZ", "N00BTUBE", "KEVLAR", "MOM", "QUADDMG", "TOASTR"]
    for i, n in enumerate(names):
        y = 12 + i * 6
        d.text((2, y), n, fill=(220, 255, 220), font=f7)
        d.line([30, y + 3, 36, y + 3], fill=(90, 255, 120))
    for i in range(4):
        y = 15 + i * 12
        d.line([36, y, 36, y + 6], fill=(90, 255, 120)); d.text((40, y), names[i * 2], fill=(255, 255, 160), font=f7)
    for i in range(2):
        y = 21 + i * 24
        d.line([70, y - 6, 70, y + 6], fill=(90, 255, 120)); d.text((74, y - 3), ["FRAGTACO", "LAGWIZ"][i], fill=(255, 255, 160), font=f7)
    d.text((104, 30), "???", fill=(255, 80, 80), font=font(MONO, 10))
    return im
T["BRACKET"] = bracket

def crt():
    a = colorize(noise(64, 64, 2) * 0.2 + 0.8, (170, 160, 135), (215, 205, 180))
    a[54:58] *= 0.7
    return img(a)
T["CRT"] = crt

def screen_maze():
    """Original retro 'lander' game: starfield, jagged terrain, a little ship."""
    im = Image.new("RGB", (64, 48), (0, 0, 0))
    d = ImageDraw.Draw(im)
    for _ in range(25):
        d.point((int(rng.integers(0, 64)), int(rng.integers(0, 30))), fill=(200, 200, 255))
    pts = [(0, 40), (8, 34), (14, 38), (22, 30), (28, 41), (40, 41), (46, 32), (54, 36), (63, 28)]
    d.line(pts, fill=(80, 255, 120), width=1)
    d.line([28, 41, 40, 41], fill=(255, 255, 255), width=2)
    d.polygon([(33, 14), (29, 22), (37, 22)], outline=(255, 200, 60))
    d.line([31, 23, 33, 27], fill=(255, 90, 30)); d.line([35, 23, 33, 27], fill=(255, 90, 30))
    d.text((2, 1), "FUEL 42", fill=(80, 255, 120), font=font(MONO, 7))
    return im
T["SCREEN_R1"] = screen_maze

def screen_amber():
    im = Image.new("RGB", (64, 48), (14, 6, 0))
    d = ImageDraw.Draw(im); f = font(MONO, 7)
    for i, t in enumerate(["READY.", "LOAD \"*\",8", "SEARCHING", "LOADING", "RUN"]):
        d.text((2, 1 + i * 9), t, fill=(255, 170, 40), font=f)
    return im
T["SCREEN_R2"] = screen_amber

def arcade():
    a = colorize(noise(64, 64, 1) * 0.3, (20, 15, 40), (60, 30, 90))
    a[:, :6] = (255, 120, 20); a[:, -6:] = (0, 220, 200)
    return img(a)
T["ARCADE"] = arcade

T["DIRT"] = lambda: img(colorize(rng.random((128, 128)) * 0.45 + noise(128, 128, 2.5) * 0.55, (45, 32, 22), (110, 85, 60)))
T["CABLEB"] = lambda: img(colorize(noise(16, 16, 1) * 0.3 + 0.7, (20, 60, 170), (60, 120, 255)))
T["CABLEY"] = lambda: img(colorize(noise(16, 16, 1) * 0.3 + 0.7, (170, 150, 20), (255, 230, 60)))

def logs():
    a = colorize(noise(64, 64, 0.8), (60, 40, 25), (120, 85, 55))
    for y in range(0, 64, 16):
        a[y:y + 2] *= 0.4
    a[:, ::9] *= 0.8
    return img(a)
T["LOGS"] = logs
def char():
    a = colorize(rng.random((64, 64)) * 0.5 + noise(64, 64, 1) * 0.5, (12, 8, 6), (70, 40, 25))
    hot = rng.random((64, 64)) > 0.97
    a[hot] = (255, 120, 20)
    return img(a)
T["CHAR"] = char

T["sky"] = lambda: img(np.full((64, 64, 3), (10, 12, 30)))
T["AAATRIGGER"] = lambda: img(np.full((16, 16, 3), (255, 140, 0)))

def pizzabox():
    im = Image.new("RGB", (32, 32), (190, 150, 95))
    d = ImageDraw.Draw(im)
    d.ellipse([6, 6, 25, 25], outline=(190, 30, 30), width=2)
    return im
T["PIZZABOX"] = pizzabox

def can(c0, c1):
    def f():
        im = Image.new("RGB", (16, 16), c0)
        d = ImageDraw.Draw(im)
        d.rectangle([0, 6, 15, 9], fill=c1)
        d.rectangle([0, 0, 15, 1], fill=(180, 180, 185))
        return im
    return f
T["SODA"] = can((200, 20, 30), (240, 240, 240))
T["ENERGY"] = can((20, 20, 20), (90, 255, 40))

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
EMISSIVE = {"BANNER_B": 210, "BANNER_R": 210, "BRACKET": 230, "SCREEN_R1": 220, "SCREEN_R2": 220, "CHAR": 70, "NWINDOW": 230, "LAMP": 255, "SCREEN1": 230, "SCREEN2": 230,
            "SCREEN3": 230, "SCREENTV": 230, "BANNER": 210, "SERVERRACK": 170, "PCCASE": 150}

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
