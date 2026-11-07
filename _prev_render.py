# -*- coding: utf-8 -*-
"""生成 App 启动动画预览（MP4）。

还原目标形态：
  0.0-0.45s  深色渐变底 + Ajie 标识轻微缩放淡入（scale 0.92→1, alpha 0→1）
  0.25-0.85s 登录卡片自下而上滑入并淡入（位移 40px→0）
  0.85-2.4s  静止（模拟真实等待状态，可见输入框与按钮）
  2.4-2.8s  轻微高光扫过按钮（呼吸感，非必需但更"真"）

输出：桌面/Ajie_启动动画.mp4（1080x1920, 30fps）
"""
import math
import os

from PIL import Image, ImageDraw, ImageFilter, ImageFont

W, H = 1080, 1920
FPS = 30
OUT = r"C:\Users\Administrator\Desktop\Ajie_启动动画.mp4"
FRAME_DIR = r"C:\Users\Administrator\Desktop\appnx_xinde\_prev_frames"
os.makedirs(FRAME_DIR, exist_ok=True)
for f in os.listdir(FRAME_DIR):
    os.remove(os.path.join(FRAME_DIR, f))

FONT_B = r"C:\Windows\Fonts\msyhbd.ttc"
FONT_R = r"C:\Windows\Fonts\msyh.ttc"


def font(path, size):
    try:
        return ImageFont.truetype(path, size)
    except Exception:
        return ImageFont.load_default()


F_TITLE = font(FONT_B, 96)
F_SUB = font(FONT_R, 38)
F_LABEL = font(FONT_R, 40)
F_INPUT = font(FONT_R, 44)
F_BTN = font(FONT_B, 46)
F_BADGE = font(FONT_R, 32)

BG_TOP = (11, 18, 32)
BG_MID = (16, 26, 43)
BG_BOT = (11, 18, 32)
ACCENT = (59, 130, 246)
ACCENT2 = (139, 92, 246)
WHITE = (255, 255, 255)
MUTED = (148, 163, 184)
CARD = (23, 32, 50)


def lerp(a, b, t):
    return a + (b - a) * t


def mix(c1, c2, t):
    return tuple(int(lerp(c1[i], c2[i], t)) for i in range(3))


def ease_out(t):
    return 1 - (1 - t) ** 3


def ease_in_out(t):
    return 3 * t * t - 2 * t * t * t


def bg_layer():
    """竖直渐变背景，只算一次"""
    img = Image.new("RGB", (W, H))
    d = ImageDraw.Draw(img)
    for y in range(H):
        t = y / (H - 1)
        col = mix(BG_TOP, BG_MID, t * 2) if t < 0.5 else mix(BG_MID, BG_BOT, (t - 0.5) * 2)
        d.line([(0, y), (W, y)], fill=col)
    # 右上角柔光
    glow = Image.new("RGB", (W, H), (0, 0, 0))
    gd = ImageDraw.Draw(glow)
    gd.ellipse([W * 0.45, -H * 0.15, W * 1.35, H * 0.35], fill=(28, 44, 74))
    glow = glow.filter(ImageFilter.GaussianBlur(180))
    img = Image.blend(img, Image.blend(img, glow, 0.55), 0.9)
    return img


BG = bg_layer()


def rounded(draw, box, r, fill=None, outline=None, width=1):
    draw.rounded_rectangle(box, radius=r, fill=fill, outline=outline, width=width)


def draw_logo(img, cx, cy, size, alpha, scale):
    """A 字标：圆角方块 + 渐变 + 中心字母"""
    s = int(size * scale)
    if s <= 4 or alpha <= 0:
        return
    layer = Image.new("RGBA", (s, s), (0, 0, 0, 0))
    d = ImageDraw.Draw(layer)
    r = int(s * 0.30)
    # 渐变填充
    for y in range(s):
        t = y / max(1, s - 1)
        d.line([(0, y), (s, y)], fill=(*mix(ACCENT, ACCENT2, t), 255))
    # 圆角蒙版
    mask = Image.new("L", (s, s), 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, s - 1, s - 1], radius=r, fill=255)
    layer.putalpha(mask)
    d2 = ImageDraw.Draw(layer)
    letter = "A"
    tb = d2.textbbox((0, 0), letter, font=F_TITLE)
    d2.text(((s - (tb[2] - tb[0])) / 2 - tb[0],
             (s - (tb[3] - tb[1])) / 2 - tb[1] - s * 0.02),
            letter, font=ImageFont.truetype(FONT_B, int(s * 0.55)), fill=(255, 255, 255, 255))
    if alpha < 1:
        a = layer.getchannel("A").point(lambda v: int(v * alpha))
        layer.putalpha(a)
    img.paste(layer, (int(cx - s / 2), int(cy - s / 2)), layer)


def draw_text(img, xy, text, f, fill, alpha=1.0):
    if alpha <= 0:
        return
    layer = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    d = ImageDraw.Draw(layer)
    tb = d.textbbox((0, 0), text, font=f)
    x = xy[0] - (tb[2] - tb[0]) / 2 - tb[0]
    d.text((x, xy[1]), text, font=f, fill=(*fill, int(255 * alpha)))
    img.alpha_composite(layer) if img.mode == "RGBA" else img.paste(layer, (0, 0), layer)


def draw_card(img, y_off, alpha, btn_glow):
    """登录卡片"""
    if alpha <= 0:
        return
    m = 96
    top = int(760 + y_off)
    h = 720
    layer = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    d = ImageDraw.Draw(layer)
    rounded(d, [m, top, W - m, top + h], 44, fill=(*CARD, 255))
    rounded(d, [m, top, W - m, top + h], 44, outline=(46, 60, 86, 255), width=2)

    # 标签 + 输入框
    d.text((m + 56, top + 70), "下发 ID", font=F_LABEL, fill=(*MUTED, 255))
    rounded(d, [m + 56, top + 140, W - m - 56, top + 250], 24, fill=(15, 22, 36, 255))
    d.text((m + 88, top + 176), "Ajie", font=F_INPUT, fill=(*WHITE, 255))
    # 光标
    d.rectangle([m + 190, top + 172, m + 195, top + 226], fill=(*ACCENT, 255))

    # 按钮（带高光扫过）
    by = int(top + 320)
    btn = Image.new("RGBA", (W - 2 * m - 112, 130), (0, 0, 0, 0))
    bd = ImageDraw.Draw(btn)
    bw, bh = btn.size
    for y in range(bh):
        t = y / (bh - 1)
        bd.line([(0, y), (bw, y)], fill=(*mix(ACCENT, ACCENT2, t), 255))
    mask = Image.new("L", btn.size, 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, bw - 1, bh - 1], radius=26, fill=255)
    btn.putalpha(mask)
    if btn_glow > 0:
        gd = ImageDraw.Draw(btn)
        gx = int(btn_glow * (bw + 300)) - 150
        gd.polygon([(gx, 0), (gx + 110, 0), (gx + 30, bh), (gx - 80, bh)],
                   fill=(255, 255, 255, 38))
    d3 = ImageDraw.Draw(btn)
    tb = d3.textbbox((0, 0), "登录", font=F_BTN)
    d3.text(((bw - (tb[2] - tb[0])) / 2 - tb[0], (bh - (tb[3] - tb[1])) / 2 - tb[1] - 4),
            "登录", font=F_BTN, fill=(255, 255, 255, 255))
    layer.alpha_composite(btn, (m + 56, by))

    # 底部说明
    d.text((m + 56, top + 510), "登录后自动绑定设备指纹", font=F_BADGE, fill=(*MUTED, 255))

    if alpha < 1:
        layer.putalpha(layer.getchannel("A").point(lambda v: int(v * alpha)))
    img.alpha_composite(layer) if img.mode == "RGBA" else img.paste(layer, (0, 0), layer)


def frame(i, total_sec=2.8):
    t = i / FPS
    img = BG.copy()

    # 标识：0~0.45s 淡入放大
    la = min(1.0, max(0.0, (t - 0.02) / 0.42))
    ls = 0.92 + 0.08 * ease_out(la)
    draw_logo(img, W / 2, 470, 300, ease_out(la), ls)
    # 标题
    ta = min(1.0, max(0.0, (t - 0.12) / 0.40))
    draw_text(img, (W / 2, 660), "Ajie", F_TITLE, WHITE, ease_out(ta))
    sa = min(1.0, max(0.0, (t - 0.22) / 0.40))
    draw_text(img, (W / 2, 790), "正在准备运行环境…", F_SUB, MUTED, ease_out(sa))

    # 卡片：0.35s 起上滑淡入，0.95s 完成
    ct = min(1.0, max(0.0, (t - 0.35) / 0.60))
    cy = 60 * (1 - ease_out(ct))
    draw_card(img, cy, ease_out(ct), 0.0)

    # 高光：1.9~2.4s 扫过按钮
    if 1.9 <= t <= 2.45:
        g = ease_in_out((t - 1.9) / 0.55)
        draw_card(img, 0, 1.0, g)
    elif t > 2.45:
        draw_card(img, 0, 1.0, 1.0)

    return img


total = int(FPS * 2.8)
for i in range(total):
    frame(i).save(os.path.join(FRAME_DIR, "f%04d.png" % i))
print("  渲染帧:", total, "→", FRAME_DIR)
print("  首帧:", os.path.join(FRAME_DIR, "f0000.png"))
print("  中间帧:", os.path.join(FRAME_DIR, "f0060.png"))
