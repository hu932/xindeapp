# -*- coding: utf-8 -*-
"""重排版启动动画 + 输出 MP4（无 ffmpeg 时退回 GIF）。

版式修正：
  - Logo 260 → 220，上移到 y=380
  - 标题 y=560，副标题 y=650（不再被卡片压住）
  - 卡片下移到 y=800
  - 去掉按钮上那块难看的斜切灰块，改成整块轻微高光呼吸
"""
import math
import os

from PIL import Image, ImageDraw, ImageFilter, ImageFont

W, H, FPS = 1080, 1920, 30
OUTDIR = r"C:\Users\Administrator\Desktop"
FRAME_DIR = os.path.join(OUTDIR, "appnx_xinde", "_prev_frames")
os.makedirs(FRAME_DIR, exist_ok=True)
for f in os.listdir(FRAME_DIR):
    os.remove(os.path.join(FRAME_DIR, f))

FB = r"C:\Windows\Fonts\msyhbd.ttc"
FR = r"C:\Windows\Fonts\msyh.ttc"


def font(p, s):
    try:
        return ImageFont.truetype(p, s)
    except Exception:
        return ImageFont.load_default()


F_TITLE = font(FB, 88)
F_SUB = font(FR, 36)
F_LABEL = font(FR, 38)
F_INPUT = font(FR, 42)
F_BTN = font(FB, 44)
F_BADGE = font(FR, 30)

ACCENT, ACCENT2 = (59, 130, 246), (139, 92, 246)
WHITE, MUTED = (255, 255, 255), (148, 163, 184)
CARD = (23, 32, 50)


def mix(a, b, t):
    return tuple(int(a[i] + (b[i] - a[i]) * t) for i in range(3))


def ease_out(t):
    return 1 - (1 - max(0.0, min(1.0, t))) ** 3


def ease_io(t):
    t = max(0.0, min(1.0, t))
    return 3 * t * t - 2 * t * t * t


def make_bg():
    img = Image.new("RGB", (W, H))
    d = ImageDraw.Draw(img)
    for y in range(H):
        t = y / (H - 1)
        col = mix((11, 18, 32), (16, 26, 43), t * 2) if t < 0.5 else mix((16, 26, 43), (11, 18, 32), (t - 0.5) * 2)
        d.line([(0, y), (W, y)], fill=col)
    glow = Image.new("RGB", (W, H), (0, 0, 0))
    gd = ImageDraw.Draw(glow)
    gd.ellipse([int(W * 0.45), int(-H * 0.15), int(W * 1.35), int(H * 0.32)], fill=(26, 40, 68))
    glow = glow.filter(ImageFilter.GaussianBlur(170))
    return Image.blend(img, glow, 0.55)


BG = make_bg()


def logo_layer(size, scale):
    s = max(8, int(size * scale))
    lay = Image.new("RGBA", (s, s), (0, 0, 0, 0))
    d = ImageDraw.Draw(lay)
    for y in range(s):
        d.line([(0, y), (s, y)], fill=(*mix(ACCENT, ACCENT2, y / (s - 1)), 255))
    mask = Image.new("L", (s, s), 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, s - 1, s - 1], radius=int(s * 0.3), fill=255)
    lay.putalpha(mask)
    d = ImageDraw.Draw(lay)
    f = font(FB, int(s * 0.56))
    tb = d.textbbox((0, 0), "A", font=f)
    d.text(((s - (tb[2] - tb[0])) / 2 - tb[0], (s - (tb[3] - tb[1])) / 2 - tb[1] - s * 0.02),
           "A", font=f, fill=(255, 255, 255, 255))
    return lay


def text_layer(text, f, fill, alpha):
    lay = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    d = ImageDraw.Draw(lay)
    tb = d.textbbox((0, 0), text, font=f)
    d.text((W / 2 - (tb[2] - tb[0]) / 2 - tb[0], 0), text, font=f, fill=(*fill, int(255 * alpha)))
    return lay


def card_layer(y_off, alpha, glow):
    m, h = 88, 700
    top = int(880 + y_off)
    lay = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    d = ImageDraw.Draw(lay)
    d.rounded_rectangle([m, top, W - m, top + h], radius=42, fill=(*CARD, 255),
                        outline=(46, 60, 86, 255), width=2)
    inner = W - 2 * m - 112
    d.text((m + 56, top + 64), "下发 ID", font=F_LABEL, fill=(*MUTED, 255))
    d.rounded_rectangle([m + 56, top + 132, W - m - 56, top + 240], radius=22, fill=(15, 22, 36, 255))
    d.text((m + 88, top + 166), "Ajie", font=F_INPUT, fill=(*WHITE, 255))
    d.rectangle([m + 190, top + 164, m + 195, top + 214], fill=(*ACCENT, 255))

    # 按钮：整块渐变更亮 = 高光呼吸，不再用斜切灰块
    by = top + 306
    bh = 124
    btn = Image.new("RGBA", (inner, bh), (0, 0, 0, 0))
    bd = ImageDraw.Draw(btn)
    for y in range(bh):
        c = mix(ACCENT, ACCENT2, y / (bh - 1))
        if glow > 0:
            c = mix(c, (255, 255, 255), 0.14 * glow)
        bd.line([(0, y), (inner, y)], fill=(*c, 255))
    mask = Image.new("L", btn.size, 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, inner - 1, bh - 1], radius=24, fill=255)
    btn.putalpha(mask)
    bd = ImageDraw.Draw(btn)
    tb = bd.textbbox((0, 0), "登录", font=F_BTN)
    bd.text(((inner - (tb[2] - tb[0])) / 2 - tb[0], (bh - (tb[3] - tb[1])) / 2 - tb[1] - 3),
            "登录", font=F_BTN, fill=(255, 255, 255, 255))
    lay.alpha_composite(btn, (m + 56, by))

    d.text((m + 56, top + 486), "登录后自动绑定设备指纹", font=F_BADGE, fill=(*MUTED, 255))

    if alpha < 1:
        lay.putalpha(lay.getchannel("A").point(lambda v: int(v * alpha)))
    return lay


def frame(t):
    img = BG.convert("RGBA")

    la = ease_out((t - 0.02) / 0.42)
    if la > 0:
        lay = logo_layer(220, 0.90 + 0.10 * la)
        lay.putalpha(lay.getchannel("A").point(lambda v: int(v * la)))
        img.alpha_composite(lay, (int(W / 2 - lay.width / 2), int(400 - lay.height / 2)))

    ta = ease_out((t - 0.12) / 0.40)
    if ta > 0:
        lay = text_layer("Ajie", F_TITLE, WHITE, ta)
        img.alpha_composite(lay, (0, 620))

    sa = ease_out((t - 0.22) / 0.40)
    if sa > 0:
        lay = text_layer("正在准备运行环境…", F_SUB, MUTED, sa)
        img.alpha_composite(lay, (0, 726))

    ct = ease_out((t - 0.35) / 0.60)
    if ct > 0:
        cy = int(56 * (1 - ct))
        glow = 0.0
        if t > 1.8:
            glow = ease_io((t - 1.8) / 0.5) * (1 - ease_io((t - 2.3) / 0.5)) if t < 2.3 else 0.0
            if 2.3 <= t <= 2.8:
                glow = 0.0
        img.alpha_composite(card_layer(cy, ct, glow))

    return img.convert("RGB")


FRAMES = [frame(i / FPS) for i in range(int(2.8 * FPS))]
for i, im in enumerate(FRAMES):
    im.save(os.path.join(FRAME_DIR, "f%04d.png" % i))
print("  帧:", len(FRAMES))

# 输出：优先 MP4（imageio/ffmpeg），否则 GIF
mp4 = os.path.join(OUTDIR, "Ajie_启动动画.mp4")
gif = os.path.join(OUTDIR, "Ajie_启动动画.gif")
done = False
try:
    import imageio.v2 as imageio
    with imageio.get_writer(mp4, fps=FPS, macro_block_size=None) as w:
        for im in FRAMES:
            w.append_data(__import__("numpy").array(im))
    print("  ✅ MP4:", mp4, os.path.getsize(mp4) // 1024, "KB")
    done = True
except Exception as e:
    print("  MP4 不可用:", e)

if not done:
    FRAMES[0].save(gif, save_all=True, append_images=FRAMES[1:], duration=int(1000 / FPS), loop=0)
    print("  ✅ GIF:", gif, os.path.getsize(gif) // 1024, "KB")
