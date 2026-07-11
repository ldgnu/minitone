#!/usr/bin/env python3
"""Render an ANSI terminal dump (lipgloss/bubbletea output) to a PNG image."""
import sys
import re
from PIL import Image, ImageDraw, ImageFont

FONT = "/usr/share/fonts/TTF/MesloLGLDZNerdFontMono-Regular.ttf"
FONT_SIZE = 15
DEFAULT_BG = (26, 27, 38)      # tokyonight bg
DEFAULT_FG = (192, 200, 245)   # tokyonight fg
PAD = 14
LINE_GAP = 3


def parse_ansi(text):
    """Yield (char, fg, bg) tokens. fg/bg are (r,g,b) or None (default)."""
    fg = None
    bg = None
    i = 0
    n = len(text)
    token_re = re.compile(r'\x1b\[([0-9;]*)m')
    out = []
    while i < n:
        m = token_re.search(text, i)
        if not m:
            seg = text[i:]
            for ch in seg:
                out.append((ch, fg, bg))
            break
        seg = text[i:m.start()]
        for ch in seg:
            out.append((ch, fg, bg))
        codes = [c for c in m.group(1).split(';') if c != '']
        if not codes:
            codes = ['0']
        j = 0
        while j < len(codes):
            c = codes[j]
            if c == '0':
                fg, bg = None, None
            elif c == '1':
                pass
            elif c == '39':
                fg = None
            elif c == '49':
                bg = None
            elif c == '38' and j + 2 < len(codes) and codes[j + 1] == '2':
                fg = (int(codes[j + 2]), int(codes[j + 3]), int(codes[j + 4]))
                j += 4
            elif c == '48' and j + 2 < len(codes) and codes[j + 1] == '2':
                bg = (int(codes[j + 2]), int(codes[j + 3]), int(codes[j + 4]))
                j += 4
            j += 1
        i = m.end()
    return out


def main():
    data = open(sys.argv[1], 'r', encoding='utf-8').read()
    tokens = parse_ansi(data)

    font = ImageFont.truetype(FONT, FONT_SIZE)
    cell_w = font.getbbox('M')[2]
    cell_h = FONT_SIZE + LINE_GAP

    # first pass: size
    max_x = 0
    max_y = 0
    cx, cy = 0, 0
    for ch, fg, bg in tokens:
        if ch == '\n':
            cx = 0
            cy += 1
            continue
        if ch == '\x1b':
            continue
        max_x = max(max_x, cx)
        max_y = max(max_y, cy)
        cx += 1
    cols = max_x + 1
    rows = max_y + 1

    img = Image.new('RGB', (cols * cell_w + PAD * 2, rows * cell_h + PAD * 2), DEFAULT_BG)
    d = ImageDraw.Draw(img)

    cx, cy = 0, 0
    for ch, fg, bg in tokens:
        if ch == '\n':
            cx, cy = 0, cy + 1
            continue
        if ch == '\x1b':
            continue
        x = PAD + cx * cell_w
        y = PAD + cy * cell_h
        if bg is not None:
            d.rectangle([x, y, x + cell_w, y + cell_h], fill=bg)
        d.text((x, y), ch, fill=fg if fg else DEFAULT_FG, font=font)
        cx += 1

    out = sys.argv[2] if len(sys.argv) > 2 else 'out.png'
    img.save(out)
    print(out, img.size)


if __name__ == '__main__':
    main()
