"""Original textures for BattleAxe Combat League (BACL) Fight Night: an indoor arena look.
Charcoal riveted panels, red/black branding with a double-bitted axe mark, dark chain-link cage.
All names (league, fighters) are invented."""
import math
import struct

import numpy as np
from PIL import Image, ImageDraw, ImageFilter

from tools.assets.library import noise, colorize, img, font, FB, MONO, rng, T as JT

T = {}
for k in ("CONCRETE", "METAL", "TRIM", "LAMP", "TABLE", "TVBLACK", "CEILTILE", "STEP", "AAATRIGGER"):
    T[k] = JT[k]

RED, RED_D, BLUE, WHITE, INK = (225, 35, 45), (120, 18, 26), (45, 115, 255), (245, 245, 245), (14, 14, 17)


# ------------------------------------------------------------------ logo pieces
def axe_icon(d, cx, cy, s, col):
    """Double-bitted battle axe, s = overall height in px."""
    hw = max(1.0, s * 0.055)
    d.rectangle([cx - hw, cy - s * 0.5, cx + hw, cy + s * 0.5], fill=col)
    top, bot = cy - s * 0.44, cy + s * 0.02
    for sg in (-1, 1):
        d.polygon([(cx + sg * hw, top + s * 0.08), (cx + sg * s * 0.40, top - s * 0.02),
                   (cx + sg * s * 0.31, (top + bot) / 2), (cx + sg * s * 0.40, bot + s * 0.02),
                   (cx + sg * hw, bot - s * 0.08)], fill=col)
    d.ellipse([cx - hw * 1.8, cy - s * 0.5 - hw * 1.5, cx + hw * 1.8, cy - s * 0.5 + hw * 2], fill=col)


def text_img(text, size, fill, f=FB, italic=0.0, spacing=0):
    fo = font(f, size)
    x0, y0, x1, y1 = fo.getbbox(text)
    w = int(x1 - x0 + spacing * len(text) + 4); h = int(y1 - y0 + 4)
    w += int(italic * h) + 2
    lay = Image.new("RGBA", (w, h), (0, 0, 0, 0)); d = ImageDraw.Draw(lay)
    if spacing:
        x = 2 - x0
        for ch in text:
            d.text((x, 2 - y0), ch, font=fo, fill=fill); x += fo.getlength(ch) + spacing
    else:
        d.text((2 - x0, 2 - y0), text, font=fo, fill=fill)
    if italic:
        lay = lay.transform(lay.size, Image.Transform.AFFINE, (1, italic, -italic * h, 0, 1, 0), Image.Resampling.BICUBIC)
    return lay


def logo(h=64, axe=RED, fg=WHITE, sub=True):
    """Axe mark + italic 'BattleAxe' + spaced 'COMBAT LEAGUE', as RGBA."""
    big = text_img("BattleAxe", int(h * 0.66), fg, italic=0.2)
    parts = [big]
    small = text_img("COMBAT LEAGUE", int(h * 0.2), fg, spacing=max(1, int(h * 0.05))) if sub else None
    aw = int(h * 0.78)
    W = aw + max(big.width, small.width if small else 0) + 4
    im = Image.new("RGBA", (W, h), (0, 0, 0, 0)); d = ImageDraw.Draw(im)
    axe_icon(d, aw * 0.45, h * 0.5, h * 0.92, axe)
    by = int(h * 0.04) if sub else (h - big.height) // 2
    im.alpha_composite(big, (aw, max(0, by)))
    if small:
        im.alpha_composite(small, (aw + int(h * 0.1), min(h - small.height, by + big.height + 1)))
    return im


def paste_fit(dst, src, box):
    x0, y0, x1, y1 = box
    k = min((x1 - x0) / src.width, (y1 - y0) / src.height)
    s = src.resize((max(1, int(src.width * k)), max(1, int(src.height * k))), Image.Resampling.LANCZOS)
    dst.alpha_composite(s, (int(x0 + (x1 - x0 - s.width) / 2), int(y0 + (y1 - y0 - s.height) / 2)))


def slashes(d, x, y, n, w, h, col, gap=None):
    gap = gap or w * 0.6
    for i in range(n):
        xi = x + i * (w + gap)
        d.polygon([(xi + h * 0.5, y), (xi + h * 0.5 + w, y), (xi + w, y + h), (xi, y + h)], fill=col)


def rgba(w, h, col): return Image.new("RGBA", (w, h), col + (255,))


def led(im, strength=0.85):
    a = np.asarray(im.convert("RGB")).astype(float)
    a[:, ::2] *= strength; a[::2] *= (strength + 1) / 2
    return img(a)


# ------------------------------------------------------------------ building surfaces
def panels(w, h, c0, c1, cell, rivet=True):
    a = colorize(noise(w, h, 2) * 0.22 + rng.random((h, w)) * 0.05, c0, c1)
    for k in range(0, h, cell):
        a[k] *= 0.55; a[(k + 1) % h] *= 1.12
    for k in range(0, w, cell):
        a[:, k] *= 0.55; a[:, (k + 1) % w] *= 1.12
    if rivet:
        for y in range(0, h, cell):
            for x in range(0, w, cell):
                for dx, dy in ((4, 4), (cell - 5, 4), (4, cell - 5), (cell - 5, cell - 5)):
                    yy, xx = (y + dy) % h, (x + dx) % w
                    a[yy, xx] = np.minimum(a[yy, xx] * 1.6, 255)
    return img(a)


T["STADWALL"] = lambda: panels(128, 128, (66, 70, 78), (88, 92, 100), 32)
T["TUNNELWALL"] = lambda: panels(64, 64, (52, 55, 62), (68, 71, 78), 32)
T["ARENAFLOOR"] = lambda: img(colorize(rng.random((64, 64)) * 0.35 + noise(64, 64, 2) * 0.65, (34, 36, 40), (56, 58, 63)))
T["STANDS"] = lambda: img(colorize(rng.random((64, 64)) * 0.4 + noise(64, 64, 1) * 0.6, (76, 78, 84), (100, 102, 108)))
T["SEATS"] = lambda: img(colorize(noise(64, 64, 1) * 0.3, (46, 48, 54), (64, 66, 72)))
T["STAGE"] = lambda: img(colorize(noise(64, 64, 3) * 0.3 + rng.random((64, 64)) * 0.1, (12, 12, 14), (34, 34, 38)))
T["APRON"] = lambda: img(colorize(noise(64, 64, 2) * 0.3, (10, 10, 12), (30, 30, 34)))
T["CAGEPAD"] = lambda: img(colorize(noise(32, 32, 1) * 0.4, (10, 10, 12), (36, 36, 40)))
T["COUCH"] = lambda: img(colorize(noise(32, 32, 1) * 0.4, (26, 26, 30), (44, 44, 50)))
T["CAMERA"] = lambda: img(colorize(noise(32, 32, 1) * 0.4, (20, 20, 22), (60, 60, 66)))
T["REDSTRIP"] = lambda: img(np.full((16, 16, 3), (255, 45, 55), float))
T["BLUESTRIP"] = lambda: img(np.full((16, 16, 3), (60, 130, 255), float))


def roof():
    """Dark ceiling deck with a beam grid and strip-light fixtures (drawn for scale 4)."""
    a = colorize(noise(128, 128, 2) * 0.25, (30, 32, 37), (42, 44, 50))
    for k in range(0, 128, 64):
        a[k:k + 3] = (62, 64, 70); a[:, k:k + 3] = (62, 64, 70)
    for k in range(0, 128, 16):
        a[k] *= 0.8
    im = img(a); d = ImageDraw.Draw(im)
    for cy in (32, 96):
        for cx in (32, 96):
            d.rectangle([cx - 14, cy - 3, cx + 14, cy + 3], fill=(70, 70, 74))
            d.rectangle([cx - 12, cy - 2, cx + 12, cy + 2], fill=(235, 235, 225))
    return im
T["ROOF"] = roof


def truss():
    im = Image.new("RGB", (32, 32), (52, 54, 60)); d = ImageDraw.Draw(im)
    d.rectangle([0, 0, 31, 3], fill=(120, 122, 130)); d.rectangle([0, 28, 31, 31], fill=(120, 122, 130))
    d.line([0, 3, 31, 28], fill=(105, 107, 115), width=2); d.line([0, 28, 31, 3], fill=(105, 107, 115), width=2)
    return im
T["TRUSS"] = truss


def suites():
    """Glass-fronted suite level: lit rooms with seated silhouettes, red trim under the sill."""
    r = np.random.default_rng(14)
    im = Image.new("RGB", (256, 64), (86, 90, 98)); d = ImageDraw.Draw(im)
    d.rectangle([0, 0, 255, 7], fill=(40, 42, 48))
    for i in range(8):
        x0 = i * 32 + 3; x1 = x0 + 27
        for y in range(10, 53):
            t = (y - 10) / 43
            d.line([x0, y, x1, y], fill=(int(150 - 60 * t), int(118 - 48 * t), int(84 - 30 * t)))
        d.rectangle([x0, 11, x1, 13], fill=(250, 240, 210))        # ceiling light
        x = x0 + 1
        while x < x1 - 4:
            w = int(r.integers(5, 8)); hy = int(r.integers(30, 36))
            col = [(60, 40, 38), (40, 40, 52), (70, 55, 45), (30, 30, 34)][int(r.integers(4))]
            d.ellipse([x, hy, x + 4, hy + 5], fill=(150, 110, 85) if r.random() < 0.7 else (90, 65, 50))
            d.rounded_rectangle([x - 1, hy + 5, x + w - 1, 53], 2, fill=col)
            x += w
        d.line([x0 + 4, 52, x0 + 14, 14], fill=(170, 160, 150), width=1)  # reflection
    d.rectangle([0, 53, 255, 59], fill=(110, 114, 122))
    d.rectangle([0, 60, 255, 63], fill=RED)
    return im
T["SUITES"] = suites


# ------------------------------------------------------------------ cage and canvas
def fence():
    w = h = 64
    yy, xx = np.mgrid[0:h, 0:w]
    a = np.zeros((h, w, 3)); a[:] = (0, 0, 255)
    wire = ((xx + yy) % 16 < 2) | ((xx - yy) % 16 < 2)
    shade = 46 + 26 * ((xx + yy) % 16) / 16
    a[wire] = np.stack([shade, shade, shade + 4], -1)[wire]
    return img(a)
T["{CAGEFENCE"] = fence


def cagepost():
    im = rgba(32, 128, INK); d = ImageDraw.Draw(im)
    d.rectangle([0, 0, 31, 5], fill=RED)
    axe_icon(d, 16, 26, 22, WHITE)
    for ch, y in (("B", 62), ("A", 92)):
        d.text((16, y), ch, font=font(FB, 22), fill=WHITE, anchor="mm")
    return im.convert("RGB")
T["CAGEPOST"] = cagepost


def canvas():
    W = 256
    im = img(colorize(noise(W, W, 3) * 0.10 + 0.9, (214, 214, 218), (238, 238, 240))).convert("RGBA")
    d = ImageDraw.Draw(im); c = W / 2
    pts = [(c + 120 * math.cos(math.radians(22.5 + 45 * i)), c + 120 * math.sin(math.radians(22.5 + 45 * i))) for i in range(8)]
    d.polygon(pts, outline=(170, 170, 176))
    paste_fit(im, logo(80, RED, (38, 38, 44)), (24, 92, 232, 164))
    d.text((c, 200), "FIGHT NIGHT", fill=(120, 120, 128), font=font(FB, 11), anchor="mm")
    return im.convert("RGB")
T["CANVAS"] = canvas


def apronside():
    im = rgba(256, 64, INK); d = ImageDraw.Draw(im)
    d.rectangle([0, 0, 255, 5], fill=RED)
    axe_icon(d, 30, 36, 40, WHITE)
    paste_fit(im, logo(48, RED, WHITE), (60, 16, 210, 56))
    slashes(d, 214, 26, 3, 6, 18, RED)
    return im.convert("RGB")
T["APRONSIDE"] = apronside

T["AXELOGO"] = lambda: (lambda im: (axe_icon(ImageDraw.Draw(im), 32, 32, 48, RED), im)[1])(Image.new("RGB", (64, 64), INK))


# ------------------------------------------------------------------ crowd
def person_row(width, seed, phones=0.04, signs=0.02):
    r = np.random.default_rng(seed)
    im = Image.new("RGB", (width, 64), (36, 36, 42)); d = ImageDraw.Draw(im)
    shirts = [(120, 45, 45), (72, 62, 52), (150, 120, 90), (58, 68, 92), (92, 92, 98), (40, 40, 46),
              (130, 95, 70), (100, 40, 42), (168, 150, 122), (56, 74, 60), (30, 50, 90), (180, 180, 182)]
    skins = [(232, 196, 166), (210, 160, 120), (172, 118, 84), (122, 82, 58), (92, 62, 44)]
    hairs = [(30, 20, 15), (70, 45, 25), (180, 150, 95), (15, 15, 15), (110, 110, 110)]
    x = 1
    while x < width - 12:
        pw = int(r.integers(15, 20)); cx = x + pw // 2; top = int(r.integers(12, 20))
        sc = shirts[int(r.integers(len(shirts)))]; sk = skins[int(r.integers(len(skins)))]
        d.rounded_rectangle([x, top + 12, x + pw - 1, 64], 5, fill=sc)
        d.ellipse([cx - 5, top, cx + 5, top + 13], fill=sk)
        d.chord([cx - 5, top - 1, cx + 5, top + 8], 180, 360, fill=hairs[int(r.integers(len(hairs)))])
        q = r.random()
        if q < phones:
            d.rectangle([cx + 5, top - 6, cx + 8, top], fill=(235, 240, 255))
        elif q < phones + signs:
            d.line([cx, top + 14, cx, top - 6], fill=(170, 170, 170))
            d.rectangle([cx - 9, top - 16, cx + 9, top - 6], fill=INK)
            d.line([cx - 6, top - 11, cx + 6, top - 11], fill=RED, width=2)
        elif q < phones + signs + 0.1:
            d.line([x + 2, top + 16, x - 2, top - 4], fill=sk, width=3)
            d.line([x + pw - 2, top + 16, x + pw + 2, top - 4], fill=sk, width=3)
        x += pw
    return im
T["CROWD1"] = lambda: person_row(256, 1).resize((128, 32), Image.Resampling.LANCZOS)
T["CROWD2"] = lambda: person_row(256, 2, phones=0.08).resize((128, 32), Image.Resampling.LANCZOS)
T["CROWD3"] = lambda: person_row(256, 3, signs=0.06).resize((128, 32), Image.Resampling.LANCZOS)
# The stands' crowd (CROWDF*): the same rows at half resolution, drawn at twice the scale in build.py, so they
# cover the same area with a quarter of the light samples (lighting is most of the compile). Only the floor rows
# next to the octagon, which players see up close, use the full-resolution CROWD*.
for _i in (1, 2, 3):
    T["CROWDF%d" % _i] = (lambda f: lambda: f().resize((64, 16), Image.Resampling.LANCZOS))(T["CROWD%d" % _i])


def crowdtop():
    r = np.random.default_rng(9)
    a = np.zeros((64, 64, 3)); a[:] = (36, 36, 42)
    cols = np.array([(30, 20, 15), (70, 45, 25), (180, 150, 95), (15, 15, 15), (160, 115, 80), (110, 110, 110)])
    for y in range(0, 64, 3):
        for x in range(int(r.integers(0, 3)), 64, 3):
            if r.random() < 0.8:
                a[y:y + 2, x:x + 2] = cols[int(r.integers(len(cols)))]
    return img(a)
T["CROWDTOP"] = crowdtop


# ------------------------------------------------------------------ boards and signs
def ribbon(kind):
    def f():
        im = rgba(256, 32, INK); d = ImageDraw.Draw(im)
        d.rectangle([0, 0, 255, 1], fill=RED); d.rectangle([0, 30, 255, 31], fill=RED)
        if kind == 1:
            paste_fit(im, logo(40), (6, 4, 120, 28))
            d.line([128, 8, 128, 24], fill=(120, 120, 128))
            d.ellipse([138, 13, 144, 19], fill=RED)
            d.text((150, 16), "LIVE  FIGHT NIGHT", fill=WHITE, font=font(FB, 11), anchor="lm")
        elif kind == 2:
            slashes(d, 8, 9, 3, 5, 14, RED); slashes(d, 222, 9, 3, 5, 14, RED)
            paste_fit(im, logo(44), (48, 3, 208, 29))
        else:
            for x in (14, 242):
                axe_icon(d, x, 16, 20, RED)
            d.text((128, 16), "CHAMPIONSHIP SERIES", fill=WHITE, font=font(FB, 13), anchor="mm")
        return led(im)
    return f
T["RIBBON1"] = ribbon(1)
T["RIBBON2"] = ribbon(2)
T["RIBBON3"] = ribbon(3)


def scoreboard(variant):
    def f():
        im = rgba(256, 160, (16, 16, 20)); d = ImageDraw.Draw(im)
        d.rectangle([0, 0, 255, 159], outline=RED, width=4)
        d.text((12, 16), "BACL  /  LIVE", fill=(170, 170, 178), font=font(MONO, 10), anchor="lm")
        d.text((244, 16), "FIGHT NIGHT", fill=(170, 170, 178), font=font(MONO, 10), anchor="rm")
        if variant == 1:
            paste_fit(im, logo(80), (20, 30, 236, 104))
        else:
            d.text((128, 58), "MORROW  vs  REYES", fill=WHITE, font=font(FB, 20), anchor="mm")
            d.rectangle([40, 78, 127, 84], fill=RED_D); d.rectangle([129, 78, 216, 84], fill=(20, 40, 110))
            d.text((128, 98), "MIDDLEWEIGHT  CHAMPIONSHIP", fill=(200, 200, 206), font=font(MONO, 10), anchor="mm")
        y0 = 114
        cells = [(10, 128, "MAIN EVENT"), (132, 178, "01"), (182, 246, "05:00")]
        for x0, x1, t in cells:
            d.rectangle([x0, y0, x1, y0 + 34], fill=(34, 34, 40))
            d.text(((x0 + x1) / 2, y0 + 17), t, fill=RED if t == "05:00" else WHITE, font=font(FB, 16), anchor="mm")
        return led(im, 0.9)
    return f
T["JUMBO1"] = scoreboard(1)
T["JUMBO2"] = scoreboard(2)


def walkout():
    """Red-corner entrance wall, 512x144 across ~1420 x 400 units; gate opening at the bottom centre."""
    im = panels(512, 144, (66, 70, 78), (88, 92, 100), 48).convert("RGBA"); d = ImageDraw.Draw(im)
    gx0, gx1, gy = 238, 274, 104                      # gate opening (cut in the wall itself)
    d.rectangle([gx0 - 14, gy - 12, gx1 + 14, 143], fill=(112, 116, 124))
    d.rectangle([gx0, gy, gx1, 143], fill=(10, 10, 12))
    for x in (gx0 - 20, gx1 + 14):
        d.rectangle([x, gy - 20, x + 6, 143], fill=(255, 45, 55))
    d.rectangle([gx0 - 20, gy - 20, gx1 + 20, gy - 15], fill=(255, 45, 55))
    d.rectangle([186, 52, 326, 80], fill=INK, outline=RED, width=2)
    paste_fit(im, logo(48), (192, 55, 320, 77))
    for bx in (24, 348):
        d.rectangle([bx, 20, bx + 140, 100], fill=INK)
        d.rectangle([bx, 94, bx + 140, 100], fill=RED)
        paste_fit(im, logo(64), (bx + 8, 30, bx + 132, 86))
    for x in range(8, 512, 32):
        d.rectangle([x, 136, x + 16, 139], fill=(255, 45, 55) if abs(x + 8 - 256) > 40 else (10, 10, 12))
    return im.convert("RGB")
T["WALKOUT"] = walkout


def walkway(col, flip):
    def f():
        im = img(colorize(noise(128, 128, 2) * 0.3, (12, 12, 14), (28, 28, 32))); d = ImageDraw.Draw(im)
        for x in (0, 122):
            d.rectangle([x, 0, x + 5, 127], fill=col)
        for y in range(4, 128, 16):
            d.rectangle([10, y, 12, y + 6], fill=(230, 230, 230)); d.rectangle([115, y, 117, y + 6], fill=(230, 230, 230))
        for cy in (40, 104):
            d.line([(36, cy + 14), (64, cy - 6), (92, cy + 14)], fill=col, width=7)
        return im.transpose(Image.Transpose.FLIP_TOP_BOTTOM) if flip else im
    return f
T["WALKWAY_R"] = walkway(RED, False)
T["WALKWAY_B"] = walkway(BLUE, True)


def corner_sign(word, base, accent, gate):
    def f():
        im = rgba(256, 128, (22, 22, 28)); d = ImageDraw.Draw(im)
        d.polygon([(0, 0), (150, 0), (104, 127), (0, 127)], fill=base)
        d.rectangle([0, 0, 255, 5], fill=accent); d.rectangle([0, 122, 255, 127], fill=accent)
        d.text((14, 20), "BACL  /  ATHLETE ENTRANCE", fill=(220, 220, 226), font=font(MONO, 9), anchor="lm")
        d.text((242, 20), gate, fill=(220, 220, 226), font=font(MONO, 9), anchor="rm")
        big = text_img(word, 40, WHITE)
        big = big.resize((min(228, big.width), big.height))
        im.alpha_composite(big, (12, 34))
        d.rectangle([14, 84, 70, 86], fill=accent)
        d.text((14, 100), "FIGHTER WALKOUT", fill=(220, 220, 226), font=font(MONO, 10), anchor="lm")
        slashes(d, 176, 92, 4, 8, 18, accent, gap=5)
        return im.convert("RGB")
    return f
T["CORNER_R"] = corner_sign("RED CORNER", (110, 20, 28), RED, "GATE 01")
T["CORNER_B"] = corner_sign("BLUE CORNER", (20, 40, 110), BLUE, "GATE 02")


def lockers(c0, c1, prefix):
    def f():
        a = colorize(noise(64, 128, 1) * 0.3 + 0.7, c0, c1)
        a[:, ::32] *= 0.35; a[:, 1::32] *= 0.6
        for x in range(0, 64, 32):
            for y in (10, 15, 20, 25): a[y:y + 2, x + 8:x + 24] *= 0.45
            a[64:72, x + 26:x + 28] = (200, 200, 205)
        im = img(a); d = ImageDraw.Draw(im)
        for i, x in enumerate((0, 32)):
            d.rectangle([x + 7, 34, x + 25, 42], fill=(225, 225, 228))
            d.text((x + 16, 38), "%s-0%d" % (prefix, i + 1), fill=(20, 20, 24), font=font(MONO, 7), anchor="mm")
        return im
    return f
T["LOCKER_R"] = lockers((88, 30, 28), (118, 44, 40), "A")
T["LOCKER_B"] = lockers((38, 48, 68), (54, 66, 90), "B")


def broadcast():
    im = Image.new("RGB", (64, 48), (10, 10, 14)); d = ImageDraw.Draw(im)
    d.rectangle([0, 0, 63, 34], fill=(40, 40, 46))
    d.polygon([(32 + 22 * np.cos(np.radians(22.5 + 45 * i)), 18 + 13 * np.sin(np.radians(22.5 + 45 * i))) for i in range(8)],
              fill=(210, 210, 215), outline=(20, 20, 20))
    d.rectangle([25, 12, 28, 22], fill=(170, 20, 20)); d.rectangle([36, 13, 39, 23], fill=(20, 40, 170))
    d.rectangle([0, 36, 63, 47], fill=(150, 10, 20))
    d.text((32, 41), "LIVE  R1 05:00", fill=(255, 255, 255), font=font(MONO, 7), anchor="mm")
    return im
T["BROADCAST"] = broadcast


def press():
    im = rgba(128, 32, INK); d = ImageDraw.Draw(im)
    d.rectangle([0, 0, 127, 2], fill=RED)
    axe_icon(d, 14, 17, 20, RED)
    d.text((72, 17), "BACL  MEDIA", fill=(210, 210, 218), font=font(FB, 12), anchor="mm")
    return im.convert("RGB")
T["PRESS"] = press


def lightbank():
    im = Image.new("RGB", (64, 32), (20, 20, 24)); d = ImageDraw.Draw(im)
    for x in range(4, 64, 15):
        for y in (3, 17):
            d.ellipse([x, y, x + 11, y + 11], fill=(255, 250, 230))
    return im
T["LIGHTBANK"] = lightbank


def beam():
    """Smooth, untextured glow (any repeat pattern would show as stripes on a big cone)."""
    return img(np.full((16, 16, 3), (255, 245, 225), float))
T["LIGHTBEAM"] = beam

EMISSIVE = {"RIBBON1": 200, "RIBBON2": 200, "RIBBON3": 200, "JUMBO1": 230, "JUMBO2": 230, "WALKOUT": 90,
            "BROADCAST": 220, "LIGHTBANK": 255, "LAMP": 255, "CORNER_R": 170, "CORNER_B": 170,
            "REDSTRIP": 255, "BLUESTRIP": 255, "SUITES": 105, "AXELOGO": 200}


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
        assert im.size[0] % 16 == 0 and im.size[1] % 16 == 0, (name, im.size)
        data = to_mip(name, im)
        entries.append(struct.pack("<IIIbbH16s", pos, len(data), len(data), 0x43, 0, 0, name.encode().ljust(16, b"\0")[:16]))
        body += data; pos += len(data)
    with open(path, "wb") as f:
        f.write(b"WAD3" + struct.pack("<II", len(entries), pos) + body + b"".join(entries))
    return sizes
