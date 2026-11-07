# -*- coding: utf-8 -*-
"""方案 C（纯文字）换背景色，出 4 个配色预览（各 2s / 1080x1920 / 30fps）。"""
import os

from PIL import Image, ImageDraw, ImageFont

W, H, FPS, DUR = 1080, 1920, 30, 2.0
OUTDIR = r"C:\Users\Administrator\Desktop"
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


def make_bg(top, bot):
    img = Image.new("RGB", (W, H))
    d = ImageDraw.Draw(img)
    for y in range(H):
        d.line([(0, y), (W, y)], fill=mix(top, bot, y / (H - 1)))
    return img.convert("RGBA")


# 4 套配色：背景渐变 / 主文字 / 副文字 / 亮带
THEMES = [
    ("1_纯黑", (0, 0, 0), (6, 6, 8), (255, 255, 255), (120, 128, 140), 20),
    ("2_深蓝黑", (8, 12, 22), (14, 20, 34), (255, 255, 255), (110, 130, 160), 26),
    ("3_墨绿黑", (6, 18, 14), (10, 26, 20), (245, 248, 246), (108, 140, 126), 24),
    ("4_暖灰白", (246, 247, 250), (232, 236, 243), (24, 28, 36), (110, 116, 128), 40),
]


def frame(t, bg, main, sub, shine):
    img = bg.copy()
    sp = 60 * (1 - eio(t / 0.9))
    al = eo(t / 0.7)
    center_text(img, 860, "Ajie", font(FL, 96), main, al, spacing=int(sp))
    if 0.7 <= t <= 1.5:
        g = eio((t - 0.7) / 0.8)
        lay = Image.new("RGBA", (W, H), (0, 0, 0, 0))
        d = ImageDraw.Draw(lay)
        x = int(-300 + g * (W + 300))
        d.polygon([(x, 820), (x + 200, 820), (x + 80, 1000), (x - 120, 1000)],
                  fill=(255, 255, 255, shine) if main[0] > 128 else (0, 0, 0, 18))
        img.alpha_composite(lay)
    sa = eo((t - 1.0) / 0.6)
    center_text(img, 1010, "正在加载环境", font(FR, 30), sub, sa, spacing=4)
    return img


import numpy as np
import imageio.v2 as imageio

for name, top, bot, main, sub, shine in THEMES:
    bg = make_bg(top, bot)
    out = os.path.join(OUTDIR, "启动动画_C_%s.mp4" % name)
    with imageio.get_writer(out, fps=FPS, macro_block_size=None) as w:
        for i in range(int(DUR * FPS)):
            w.append_data(np.array(frame(i / FPS, bg, main, sub, shine).convert("RGB")))
    print("  ✅ %s  %.0f KB" % (out, os.path.getsize(out) / 1024))
