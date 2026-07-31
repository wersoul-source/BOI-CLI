#!/usr/bin/env python3
"""Generate hacker-style 8-Art ASCII text using "8" as the solid block character.

Features:
  - 8x7 pixel bitmaps for A-Z, 0-9, space, dash
  - "8" = solid block (dense hacker aesthetic)
  - ANSI 24-bit TrueColor gradient coloring
  - Palette presets shared with generate_boi_logo.py
  - Framed output with box-drawing borders
  - Supports --text, --palette, --no-color, --output
  - Respects NO_COLOR env var
"""

import argparse
import os
import sys
import textwrap

VERSION = "1.0.0"

PALETTES = {
    "boi-purple": ((0x6C, 0x63, 0xFF), (0xA7, 0x8B, 0xFA)),
    "boi-cyan":    ((0x00, 0xD4, 0xFF), (0x6C, 0x63, 0xFF)),
    "boi-fire":    ((0xFF, 0x6B, 0x6B), (0xFF, 0xD9, 0x3D)),
    "ocean":       ((0x00, 0x72, 0xF5), (0x00, 0xC8, 0xFF)),
    "forest":      ((0x00, 0x8F, 0x5A), (0x2D, 0xD4, 0x7A)),
    "sunset":      ((0xFF, 0x7B, 0x00), (0xFF, 0x3E, 0x6C)),
    "aurora":      ((0x00, 0xFF, 0x87), (0x60, 0xEF, 0xFF)),
    "midnight":    ((0x19, 0x24, 0x6F), (0x4E, 0x54, 0xC8)),
    "matrix":      ((0x00, 0xFF, 0x00), (0x00, 0xAA, 0x00)),
    "neon-pink":   ((0xFF, 0x00, 0x88), (0xFF, 0x66, 0xCC)),
    "amber":       ((0xFF, 0xBF, 0x00), (0xFF, 0x8C, 0x00)),
    "steel":       ((0x88, 0x99, 0xAA), (0xBB, 0xCC, 0xDD)),
}

FONT_8x7 = {
    "A": [
        " 888888 ",
        "88    88",
        "88    88",
        "88888888",
        "88    88",
        "88    88",
        "88    88",
    ],
    "B": [
        "8888888 ",
        "88    88",
        "88    88",
        "8888888 ",
        "88    88",
        "88    88",
        "8888888 ",
    ],
    "C": [
        " 888888 ",
        "88    88",
        "88      ",
        "88      ",
        "88      ",
        "88    88",
        " 888888 ",
    ],
    "D": [
        "888888  ",
        "88    88",
        "88    88",
        "88    88",
        "88    88",
        "88    88",
        "888888  ",
    ],
    "E": [
        "88888888",
        "88      ",
        "88      ",
        "88888888",
        "88      ",
        "88      ",
        "88888888",
    ],
    "F": [
        "88888888",
        "88      ",
        "88      ",
        "88888888",
        "88      ",
        "88      ",
        "88      ",
    ],
    "G": [
        " 888888 ",
        "88    88",
        "88      ",
        "88  8888",
        "88    88",
        "88    88",
        " 888888 ",
    ],
    "H": [
        "88    88",
        "88    88",
        "88    88",
        "88888888",
        "88    88",
        "88    88",
        "88    88",
    ],
    "I": [
        "88888888",
        "   88   ",
        "   88   ",
        "   88   ",
        "   88   ",
        "   88   ",
        "88888888",
    ],
    "J": [
        "    8888",
        "     88 ",
        "     88 ",
        "     88 ",
        "88   88 ",
        "88   88 ",
        " 88888  ",
    ],
    "K": [
        "88   88",
        "88  88 ",
        "88 88  ",
        "8888   ",
        "88 88  ",
        "88  88 ",
        "88   88",
    ],
    "L": [
        "88      ",
        "88      ",
        "88      ",
        "88      ",
        "88      ",
        "88      ",
        "88888888",
    ],
    "M": [
        "88      88",
        "888    888",
        "88 88 88 88",
        "88  8 8  88",
        "88      88",
        "88      88",
        "88      88",
    ],
    "N": [
        "88    88",
        "888   88",
        "88 8  88",
        "88  8 88",
        "88   888",
        "88    88",
        "88    88",
    ],
    "O": [
        " 888888 ",
        "88    88",
        "88    88",
        "88    88",
        "88    88",
        "88    88",
        " 888888 ",
    ],
    "P": [
        "888888  ",
        "88    88",
        "88    88",
        "888888  ",
        "88      ",
        "88      ",
        "88      ",
    ],
    "Q": [
        " 888888 ",
        "88    88",
        "88    88",
        "88    88",
        "88 88 88",
        "88   888",
        " 8888888",
    ],
    "R": [
        "888888  ",
        "88    88",
        "88    88",
        "888888  ",
        "88  88  ",
        "88   88 ",
        "88    88",
    ],
    "S": [
        " 888888 ",
        "88    88",
        "88      ",
        " 888888 ",
        "      88",
        "88    88",
        " 888888 ",
    ],
    "T": [
        "88888888",
        "   88   ",
        "   88   ",
        "   88   ",
        "   88   ",
        "   88   ",
        "   88   ",
    ],
    "U": [
        "88    88",
        "88    88",
        "88    88",
        "88    88",
        "88    88",
        "88    88",
        " 888888 ",
    ],
    "V": [
        "88    88",
        "88    88",
        "88    88",
        "88    88",
        " 88  88 ",
        " 88  88 ",
        "   88   ",
    ],
    "W": [
        "88      88",
        "88      88",
        "88      88",
        "88  88  88",
        "88 8  8 88",
        "888    888",
        "88      88",
    ],
    "X": [
        "88    88",
        "88    88",
        " 88  88 ",
        "   88   ",
        " 88  88 ",
        "88    88",
        "88    88",
    ],
    "Y": [
        "88    88",
        "88    88",
        " 88  88 ",
        "   88   ",
        "   88   ",
        "   88   ",
        "   88   ",
    ],
    "Z": [
        "88888888",
        "      88",
        "     88 ",
        "    88  ",
        "   88   ",
        "  88    ",
        "88888888",
    ],
    "0": [
        " 888888 ",
        "88    88",
        "88   888",
        "88 8  88",
        "888   88",
        "88    88",
        " 888888 ",
    ],
    "1": [
        "   88   ",
        "  888   ",
        "   88   ",
        "   88   ",
        "   88   ",
        "   88   ",
        "88888888",
    ],
    "2": [
        " 888888 ",
        "88    88",
        "      88",
        "    888 ",
        "   88   ",
        "  88    ",
        "88888888",
    ],
    "3": [
        " 888888 ",
        "88    88",
        "      88",
        "   8888 ",
        "      88",
        "88    88",
        " 888888 ",
    ],
    "4": [
        "     88 ",
        "    888 ",
        "   8 88 ",
        "  88 88 ",
        "88888888",
        "     88 ",
        "     88 ",
    ],
    "5": [
        "88888888",
        "88      ",
        "888888  ",
        "      88",
        "      88",
        "88    88",
        " 888888 ",
    ],
    "6": [
        " 888888 ",
        "88    88",
        "88      ",
        "888888  ",
        "88    88",
        "88    88",
        " 888888 ",
    ],
    "7": [
        "88888888",
        "      88",
        "     88 ",
        "    88  ",
        "    88  ",
        "    88  ",
        "    88  ",
    ],
    "8": [
        " 888888 ",
        "88    88",
        "88    88",
        " 888888 ",
        "88    88",
        "88    88",
        " 888888 ",
    ],
    "9": [
        " 888888 ",
        "88    88",
        "88    88",
        " 8888888",
        "      88",
        "88    88",
        " 888888 ",
    ],
    " ": [
        "        ",
        "        ",
        "        ",
        "        ",
        "        ",
        "        ",
        "        ",
    ],
    "-": [
        "        ",
        "        ",
        "        ",
        " 888888 ",
        "        ",
        "        ",
        "        ",
    ],
    "_": [
        "        ",
        "        ",
        "        ",
        "        ",
        "        ",
        "        ",
        "88888888",
    ],
    ".": [
        "        ",
        "        ",
        "        ",
        "        ",
        "        ",
        "   88   ",
        "        ",
    ],
    "!": [
        "   88   ",
        "   88   ",
        "   88   ",
        "   88   ",
        "   88   ",
        "        ",
        "   88   ",
    ],
    ":": [
        "        ",
        "   88   ",
        "        ",
        "        ",
        "   88   ",
        "        ",
        "        ",
    ],
    "?": [
        " 888888 ",
        "88    88",
        "      88",
        "    88  ",
        "    88  ",
        "        ",
        "    88  ",
    ],
    "/": [
        "      88",
        "     88 ",
        "     88 ",
        "    88  ",
        "   88   ",
        "   88   ",
        " 88     ",
    ],
}


def visible_len(s):
    import re
    return len(re.sub(r"\033\[[0-9;]*m", "", s))


def rgb_gradient(start, end, steps):
    if steps <= 1:
        return [start]
    result = []
    for i in range(steps):
        t = i / (steps - 1)
        r = int(start[0] + (end[0] - start[0]) * t)
        g = int(start[1] + (end[1] - start[1]) * t)
        b = int(start[2] + (end[2] - start[2]) * t)
        result.append((r, g, b))
    return result


def ansi_truecolor(r, g, b, bg=False):
    if bg:
        return f"\033[48;2;{r};{g};{b}m"
    return f"\033[38;2;{r};{g};{b}m"


def ansi_reset():
    return "\033[0m"


def render_8art(text, font=None):
    if font is None:
        font = FONT_8x7
    lines = [""] * 7
    for ch in text.upper():
        glyph = font.get(ch, font.get("?"))
        if glyph is None:
            glyph = font[" "]
        for i in range(7):
            lines[i] += glyph[i] + "  "
    for i in range(7):
        lines[i] = lines[i].rstrip()
    return lines


def apply_gradient(lines, palette_name, no_color):
    if no_color:
        return lines

    palette = PALETTES.get(palette_name, PALETTES["boi-purple"])
    start_color, end_color = palette

    total_chars = sum(sum(1 for c in line if c == "8") for line in lines)

    if total_chars == 0:
        return lines

    gradient = rgb_gradient(start_color, end_color, total_chars)

    char_idx = 0
    result = []
    for line in lines:
        colored = ""
        for ch in line:
            if ch == "8" and char_idx < len(gradient):
                r, g, b = gradient[char_idx]
                colored += f"{ansi_truecolor(r, g, b)}8{ansi_reset()}"
                char_idx += 1
            else:
                colored += ch
        result.append(colored)
    return result


def frame_box(lines, no_color, palette_name):
    width = max(len(l) if no_color else visible_len(l) for l in lines)
    box_width = width + 2

    if no_color:
        top = "╔" + "═" * width + "╗"
        bottom = "╚" + "═" * width + "╝"
        framed = [top]
        for line in lines:
            padded = line.ljust(width)
            framed.append("║" + padded + "║")
        framed.append(bottom)
    else:
        palette = PALETTES.get(palette_name, PALETTES["boi-purple"])
        start_color, _ = palette

        top = f"{ansi_truecolor(*start_color)}╔{'═' * width}╗{ansi_reset()}"
        bottom = f"{ansi_truecolor(*start_color)}╚{'═' * width}╝{ansi_reset()}"

        framed = [top]
        for line in lines:
            vlen = visible_len(line)
            padding = width - vlen
            padded = f"{ansi_truecolor(*start_color)}║{ansi_reset()}{line}{' ' * padding}{ansi_truecolor(*start_color)}║{ansi_reset()}"
            framed.append(padded)
        framed.append(bottom)
    return framed


def generate_8art(text, palette_name="boi-purple", no_color=False, output_file=None):
    lines = render_8art(text)
    lines = apply_gradient(lines, palette_name, no_color)
    framed = frame_box(lines, no_color, palette_name)

    result = "\n".join(framed)

    if output_file:
        with open(output_file, "w", encoding="utf-8") as f:
            f.write(result + "\n")
        print(f"8-Art saved to {output_file}")

    return result


def list_palettes():
    print("Available palettes:")
    for name, (start, end) in PALETTES.items():
        r1, g1, b1 = start
        r2, g2, b2 = end
        print(f"  {ansi_truecolor(r1, g1, b1)}{name}{ansi_reset()}"
              f"  ({ansi_truecolor(r1, g1, b1)}#{r1:02X}{g1:02X}{b1:02X}{ansi_reset()}"
              f" \u2192 {ansi_truecolor(r2, g2, b2)}#{r2:02X}{g2:02X}{b2:02X}{ansi_reset()})")


def main():
    parser = argparse.ArgumentParser(
        description="Generate hacker-style 8-Art ASCII using '8' as solid block",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=textwrap.dedent("""\
            Examples:
              %(prog)s --text "BOI CLI" --palette boi-purple
              %(prog)s --text "KAMKAEW" --palette boi-cyan
              %(prog)s --text "HACK" --palette matrix --no-color
              %(prog)s --text "TIS" --output 8art_tis.txt
              %(prog)s --list-palettes
        """),
    )

    parser.add_argument("--text", default="BOI CLI",
                        help="Text to render as 8-Art (default: 'BOI CLI')")
    parser.add_argument("--palette", default="boi-purple",
                        choices=list(PALETTES.keys()),
                        help="Color palette preset (default: boi-purple)")
    parser.add_argument("--no-color", action="store_true",
                        help="Disable ANSI colors (also respects NO_COLOR env var)")
    parser.add_argument("--output", default=None,
                        help="Save output to file")
    parser.add_argument("--list-palettes", action="store_true",
                        help="Show available palettes and exit")
    parser.add_argument("--version", action="version",
                        version=f"8-Art Generator v{VERSION}")

    args = parser.parse_args()

    if args.list_palettes:
        list_palettes()
        return

    no_color = args.no_color or os.environ.get("NO_COLOR", "").strip() != ""

    result = generate_8art(
        text=args.text,
        palette_name=args.palette,
        no_color=no_color,
        output_file=args.output,
    )

    if not args.output:
        print(result)


if __name__ == "__main__":
    main()
