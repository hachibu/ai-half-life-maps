"""Texture recipes for casaroble's zones. Each zone registers the ones it uses under its own prefixed names."""
import numpy as np
from PIL import Image

from tools.mapkit import textures as L

rng = np.random.default_rng(1922)


def stucco(c0=(190, 168, 134), c1=(224, 206, 176)):
    def f():
        a = L.noise(128, 128, 0.6) * 0.45 + L.noise(128, 128, 3) * 0.55
        return L.img(L.colorize(a, c0, c1))
    return f


def plaster():
    return L.img(L.colorize(L.noise(64, 64, 1.5) * 0.3 + 0.7, (214, 208, 196), (238, 234, 226)))


def rooftile():
    """Red clay barrel tiles seen side-on: rounded columns, staggered courses."""
    x = np.arange(64)
    col = 0.55 + 0.45 * np.sin((x % 16) / 16 * np.pi)
    a = np.tile(col, (64, 1)) * (0.85 + 0.15 * L.noise(64, 64, 1))
    for y in range(0, 64, 16):
        a[y:y + 2] *= 0.45
    return L.img(L.colorize(a, (95, 38, 22), (196, 92, 58)))


def terracotta():
    a = L.colorize(L.noise(64, 64, 1.2) * 0.6 + rng.random((64, 64)) * 0.2, (150, 72, 44), (198, 110, 70))
    a[::32, :] = (170, 160, 145)
    a[:, ::32] = (170, 160, 145)
    return L.img(a)


def copper():
    n = L.noise(32, 32, 1.5)
    a = L.colorize(n, (150, 82, 40), (210, 130, 75))
    v = L.noise(32, 32, 0.8) > 0.72                   # verdigris patches
    a[v] = (95, 160, 135)
    return L.img(a)


def gravel():
    return L.img(L.colorize(rng.random((128, 128)) * 0.7 + L.noise(128, 128, 2) * 0.3, (118, 106, 88), (205, 190, 160)))


def lawn():
    return L.img(L.colorize(rng.random((128, 128)) * 0.55 + L.noise(128, 128, 4) * 0.45, (52, 88, 34), (108, 146, 62)))


def asphalt():
    return L.img(L.colorize(rng.random((128, 128)) * 0.6 + L.noise(128, 128, 2) * 0.4, (38, 38, 40), (82, 82, 86)))


def bark():
    n = L.noise(16, 64, 0.8)
    n = np.asarray(Image.fromarray((n * 255).astype(np.uint8)).resize((64, 64))) / 255.0
    return L.img(L.colorize(n * 0.7 + rng.random((64, 64)) * 0.3, (48, 38, 28), (112, 92, 70)))


def leaves():
    a = (L.noise(64, 64, 0.7) > 0.35) * 0.6 + rng.random((64, 64)) * 0.4
    return L.img(L.colorize(a, (34, 52, 22), (92, 118, 50)))


def agave():
    return L.img(L.colorize(L.noise(32, 32, 0.6), (70, 120, 110), (140, 180, 160)))


def bedspread(c0, c1):
    def f():
        a = L.noise(64, 64, 2) * 0.3 + 0.7
        a[::16, :] *= 0.8
        return L.img(L.colorize(a, c0, c1))
    return f


def cardboard():
    a = L.colorize(L.noise(32, 32, 1) * 0.4 + 0.6, (140, 102, 60), (190, 150, 100))
    a[14:18, :] = (200, 190, 150)                      # tape
    return L.img(a)


def membrane():
    return L.img(L.colorize(L.noise(64, 64, 1) * 0.5 + 0.5, (120, 112, 100), (160, 150, 135)))


def whitepaint():
    return L.img(L.colorize(L.noise(32, 32, 1) * 0.15 + 0.85, (215, 215, 212), (245, 245, 242)))


def hedge():
    return L.img(L.colorize(rng.random((64, 64)) * 0.5 + L.noise(64, 64, 1.5) * 0.5, (28, 58, 26), (74, 112, 52)))


planks = L.planks
