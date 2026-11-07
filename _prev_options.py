# -*- coding: utf-8 -*-
"""出 3 版启动动画方案（各 2.0s / 1080x1920 / 30fps）

A 极简描边：圆形 Logo 用描边"画"出来，然后 Ajie 淡入，无卡片
B 收拢入场：图标从 1.7 倍缩到 1.0，带运动模糊残影，像原生启动图标
C 纯文字：只做字距收拢 + 亮度扫过，最克制
"""
import math
import os

from PIL import Image, ImageDraw, ImageFilter, ImageFont

W, H, FPS, DUR = 1080, 1920, 30, 2.0
OUTDIR = r"C:\Users\Administrator\Desktop"
TMP = os.path.join(OUTDIR, "appnx_xinde", "_prev_opt")
os.makedirs(TMP, exist_ok=True)

FB = r"C:\Windows\Fonts\msyhbd.ttc"
FR = r"C:\Windows\Fonts\msyh.ttc"
FL = r"C:\Windows\Fonts\msyhl.ttc"


def font(p, s):
    try:
        return ImageFont.truetype(p, s)
    except Exception:
        return ImageFont.load_default()


def mix(a, b, t):
    return tuple(int(a[i] + (b[i] - a[i]) * t) for i in range(3))


def eo(t):
    t = max(0.0, min(1.0, t))
    return 1 - (1 - t) ** 3


def eio(t):
    t = max(0.0, min(1.0, t))
    return 3 * t * t - 2 * t * t * t


def bg():
    img = Image.new("RGB", (W, H))
    d = ImageDraw.Draw(img)
    for y in range(H):
        t = y / (H - 1)
        c = mix((8, 12, 22), (14, 20, 34), t)
        d.line([(0, y), (W, y)], fill=c)
    return img.convert("RGBA")


BASE = bg()


def center_text(img, y, text, f, fill, alpha, spacing=0):
    if alpha <= 0:
        return
    lay = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    d = ImageDraw.Draw(lay)
    widths = [d.textlength(ch, font=f) for ch in text]
    total = sum(widths) + spacing * (len(text) - 1)
    x = W / 2 - total / 2
    for ch, w in zip(text, widths):
        d.text((x, y), ch, font=f, fill=(*fill, int(255 * alpha)))
        x += w + spacing
    img.alpha_composite(lay)


# ---------------- A：极简描边 ----------------
def opt_a(t):
    img = BASE.copy()
    cx, cy, r = W / 2, 760, 120
    prog = eo(t / 0.85)
    if prog > 0:
        lay = Image.new("RGBA", (W, H), (0, 0, 0, 0))
        d = ImageDraw.Draw(lay)
        # 描边圆环
        d.arc([cx - r, cy - r, cx + r, cy + r], -90, -90 + 360 * prog,
              fill=(120, 170, 255, 255), width=6)
        # 内部 A
        if prog > 0.55:
            a2 = (prog - 0.55) / 0.45
            f = font(FL, 150)
            tb = d.textbbox((0, 0), "A", font=f)
            d.text((cx - (tb[2] - tb[0]) / 2 - tb[0], cy - (tb[3] - tb[1]) / 2 - tb[1]),
                   "A", font=f, fill=(255, 255, 255, int(235 * a2)))
        img.alpha_composite(lay)
    ta = eo((t - 0.55) / 0.5)
    center_text(img, 940, "Ajie", font(FL, 76), (255, 255, 255), ta, spacing=14)
    sa = eo((t - 0.95) / 0.5)
    center_text(img, 1060, "AJIE SYSTEM", font(FR, 26), (110, 130, 160), sa, spacing=8)
    return img


# ---------------- B：收拢入场 ----------------
def opt_b(t):
    img = BASE.copy()
    p = eio(t / 0.75)
    scale = 1.7 - 0.7 * p
    alpha = min(1.0, t / 0.25)
    s = int(300 * scale)
    lay = Image.new("RGBA", (s, s), (0, 0, 0, 0))
    d = ImageDraw.Draw(lay)
    for y in range(s):
        d.line([(0, y), (s, y)], fill=(*mix((70, 110, 220), (150, 110, 240), y / (s - 1)), 255))
    mask = Image.new("L", (s, s), 0)
    ImageDraw.Draw(mask).ellipse([0, 0, s - 1, s - 1], fill=255)
    lay.putalpha(mask)
    d = ImageDraw.Draw(lay)
    f = font(FB, int(s * 0.5))
    tb = d.textbbox((0, 0), "A", font=f)
    d.text(((s - (tb[2] - tb[0])) / 2 - tb[0], (s - (tb[3] - tb[1])) / 2 - tb[1] - s * 0.02),
           "A", font=f, fill=(255, 255, 255, 255))
    if alpha < 1:
        lay.putalpha(lay.getchannel("A").point(lambda v: int(v * alpha)))
    # 残影（缩放过程中的拖影感）
    if 0.02 < p < 0.96:
        ghost = lay.filter(ImageFilter.GaussianBlur(18))
        ghost.putalpha(ghost.getchannel("A").point(lambda v: int(v * 0.35)))
        gs = int(300 * (scale + 0.12))
        ghost = ghost.resize((gs, gs))
        img.alpha_composite(ghost, (int(W / 2 - gs / 2), int(780 - gs / 2)))
    img.alpha_composite(lay, (int(W / 2 - s / 2), int(780 - s / 2)))

    ty = 980 + int(20 * (1 - eo((t - 0.35) / 0.5)))
    ta = eo((t - 0.35) / 0.5)
    center_text(img, ty, "Ajie", font(FB, 64), (255, 255, 255), ta, spacing=6)
    return img


# ---------------- C：纯文字 ----------------
def opt_c(t):
    img = BASE.copy()
    sp = 60 * (1 - eio(t / 0.9))
    al = eo(t / 0.7)
    center_text(img, 860, "Ajie", font(FL, 96), (255, 255, 255), al, spacing=int(sp))
    # 亮度扫过
    if 0.7 <= t <= 1.5:
        g = eio((t - 0.7) / 0.8)
        lay = Image.new("RGBA", (W, H), (0, 0, 0, 0))
        d = ImageDraw.Draw(lay)
        x = int(-300 + g * (W + 300))
        d.polygon([(x, 820), (x + 200, 820), (x + 80, 1000), (x - 120, 1000)],
                  fill=(255, 255, 255, 26))
        img.alpha_composite(lay)
    sa = eo((t - 1.0) / 0.6)
    center_text(img, 1010, "正在加载环境", font(FR, 30), (120, 138, 165), sa, spacing=4)
    return img


def render(fn, name):
    frames = []
    d = os.path.join(TMP, name)
    os.makedirs(d, exist_ok=True)
    for i in range(int(DUR * FPS)):
        im = fn(i / FPS).convert("RGB")
        frames.append(im)
    import imageio.v2 as imageio
    import numpy as np
    out = os.path.join(OUTDIR, "启动动画_%s.mp4" % name)
    with imageio.get_writer(out, fps=FPS, macro_block_size=None) as w:
        for im in frames:
            w.append_data(np.array(im))
    print("  ✅ %s  %.0f KB" % (out, os.path.getsize(out) / 1024))
    return out


render(opt_a, "A_极简描边")
render(opt_b, "B_收拢入场")
render(opt_c, "C_纯文字")
