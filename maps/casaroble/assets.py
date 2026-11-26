"""casaroble's own game assets: a warm late-afternoon skybox and the fireplace's fire sprite."""
import numpy as np

from tools.mapkit import sky

SUN = np.array([-0.45, -0.55, 0.7])          # roughly where light_environment's sun sits (south-west, high)
SUN = SUN / np.linalg.norm(SUN)


def afternoon(d):
    d = d / np.linalg.norm(d, axis=-1, keepdims=True)
    up = np.clip(d[..., 2], -1, 1)
    zenith, horizon, ground = np.array([70, 120, 196]), np.array([236, 214, 182]), np.array([120, 112, 98])
    t = np.clip(up, 0, 1)[..., None] ** 0.55
    col = horizon + (zenith - horizon) * t
    col = np.where(up[..., None] < 0, ground + (horizon - ground) * np.clip(1 + up * 4, 0, 1)[..., None], col)
    sun = np.clip(d @ SUN, 0, 1)
    col = col + np.array([255, 226, 170]) * (sun ** 48 * 1.2 + sun ** 6 * 0.25)[..., None]
    # a few soft cumulus bands near the horizon
    az = np.arctan2(d[..., 1], d[..., 0])
    cloud = np.clip(np.sin(az * 5 + 1.3) * np.sin(az * 11) - 0.2, 0, 1) * np.exp(-((up - 0.18) / 0.08) ** 2)
    return col + (np.array([250, 244, 236]) - col) * (cloud * 0.6)[..., None]


def write_skybox(prefix, n=256):
    for f in ("rt", "bk", "lf", "ft", "up", "dn"):
        j, i = np.mgrid[0:n, 0:n]
        s = (i + 0.5) / n * 2 - 1
        t = 1 - (j + 0.5) / n * 2
        sky.write_tga(f"{prefix}{f}.tga", np.clip(afternoon(sky.face_dirs(f, s, t)), 0, 255).astype(np.uint8))


def write(valve, name):
    (valve / "gfx" / "env").mkdir(parents=True, exist_ok=True)
    (valve / "sprites").mkdir(parents=True, exist_ok=True)
    write_skybox(str(valve / "gfx" / "env" / name))
    sky.write_sprite(str(valve / "sprites" / (name + "_fire.spr")))
