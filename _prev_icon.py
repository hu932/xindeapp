# -*- coding: utf-8 -*-
"""方案 C + 真实 App 图标 + 纯黑背景，时长拉长到 3.2s 的预览。"""
import os

import numpy as np
import imageio.v2 as imageio
from PIL import Image, ImageDraw, ImageFont

W, H, FPS = 1080, 1920, 30
DUR = 3.2
OUTDIR = r"C:\Users\Administrator\Desktop"
ICON = r"C:\Users\Administrator\Desktop\appnx_xinde\app\src\main\res\mipmap-xxxhdpi\ic_launcher.png"
FL = r"C:\Windows\Fonts\msyhl.ttc"
FR = r"C:\Windows\Fonts\msyh.ttc"


def font(p, s):
    try:
        return ImageFont.truetype(p, s)
    except Exception:
        return ImageFont.load_default()


def eo(t):
    t = max(0.0, min(1.0, t))
    return 1 - (1 - t) ** 3


def center_letters(img, y, text, f, fill, alpha, spacing):
    if alpha <= 0:
        return
    lay = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    d = ImageDraw.Draw(lay)
    ws = [d.textlength(ch, font=f) for ch in text]
    total = sum(ws) + spacing * (len(text) - 1)
    x = W / 2 - total / 2
    for ch, w in zip(text, ws):
        d.text((x, y), ch, font=f, fill=(*fill, int(255 * alpha)))
        x += w + spacing
    img.alpha_composite(lay)


ICON_IMG = Image.open(ICON).convert("RGBA")
ICON_SIZE = 300


def icon_at(scale, alpha):
    s = max(8, int(ICON_SIZE * scale))
    im = ICON_IMG.resize((s, s), Image.LANCZOS)
    if alpha < 1:
        im.putalpha(im.getchannel("A").point(lambda v: int(v * alpha)))
    return im


def frame(t):
    img = Image.new("RGBA", (W, H), (0, 0, 0, 255))

    # 图标：0-0.55s 淡入 + 从 0.90 放大到 1.0
    ia = eo(t / 0.55)
    if ia > 0:
        ic = icon_at(0.90 + 0.10 * ia, ia)
        img.alpha_composite(ic, (int(W / 2 - ic.width / 2), int(700 - ic.height / 2)))

    # 标题：0.35-0.95s，字距 18→0
    ta = eo((t - 0.35) / 0.60)
    sp = 18 * (1 - ta)
    center_letters(img, 900, "Ajie", font(FL, 96), (255, 255, 255), ta, sp)

    # 副标题：0.95-1.55s
    sa = eo((t - 0.95) / 0.60)
    if sa > 0:
        lay = Image.new("RGBA", (W, H), (0, 0, 0, 0))
        d = ImageDraw.Draw(lay)
        txt = "正在加载环境"
        f = font(FR, 30)
        ws = [d.textlength(ch, font=f) for ch in txt]
        total = sum(ws) + 4 * (len(txt) - 1)
        x = W / 2 - total / 2
        for ch, w in zip(txt, ws):
            d.text((x, 1044), ch, font=f, fill=(138, 143, 152, int(255 * sa)))
            x += w + 4
        img.alpha_composite(lay)

    # 1.6-2.2s 静止；2.4s 后整体淡出
    if t > 2.4:
        k = eo((t - 2.4) / 0.7)
        fade = Image.new("RGBA", (W, H), (0, 0, 0, int(255 * k)))
        img.alpha_composite(fade)

    return img.convert("RGB")


out = os.path.join(OUTDIR, "启动动画_C_Ajie图标_纯黑_3.2s.mp4")
with imageio.get_writer(out, fps=FPS, macro_block_size=None) as w:
    for i in range(int(DUR * FPS)):
        w.append_data(np.array(frame(i / FPS)))
print("  ✅", out, "%.0f KB" % (os.path.getsize(out) / 1024))
