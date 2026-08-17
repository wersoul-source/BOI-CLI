#!/usr/bin/env python3
"""Generate BOI CLI ASCII logo with ANSI TrueColor gradients.

Features:
  - Block font (5x7) for main text
  - Box frame with box-drawing characters
  - ANSI 24-bit TrueColor gradient
  - BOI palette presets: boi-purple, boi-cyan, boi-fire
  - Supports --text, --subtitle, --palette, --style, --no-color, --output
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
}

BLOCK_FONT = {
    "A": [" ███ ", "█   █", "█   █", "█████", "█   █", "█   █", "█   █"],
    "B": ["████ ", "█   █", "█   █", "████ ", "█   █", "█   █", "████ "],
    "C": [" ███ ", "█   █", "█    ", "█    ", "█    ", "█   █", " ███ "],
    "D": ["████ ", "█   █", "█   █", "█   █", "█   █", "█   █", "████ "],
    "E": ["█████", "█    ", "█    ", "█████", "█    ", "█    ", "█████"],
    "F": ["█████", "█    ", "█    ", "█████", "█    ", "█    ", "█    "],
    "G": [" ███ ", "█   █", "█    ", "█ ███", "█   █", "█   █", " ███ "],
    "H": ["█   █", "█   █", "█   █", "█████", "█   █", "█   █", "█   █"],
    "I": ["█████", "  █  ", "  █  ", "  █  ", "  █  ", "  █  ", "█████"],
    "J": ["█████", "   █ ", "   █ ", "   █ ", "   █ ", "█  █ ", " ██  "],
    "K": ["█   █", "█  █ ", "█ █  ", "██   ", "█ █  ", "█  █ ", "█   █"],
    "L": ["█    ", "█    ", "█    ", "█    ", "█    ", "█    ", "█████"],
    "M": ["█   █", "██ ██", "█ █ █", "█   █", "█   █", "█   █", "█   █"],
    "N": ["█   █", "██  █", "█ █ █", "█  ██", "█   █", "█   █", "█   █"],
    "O": [" ███ ", "█   █", "█   █", "█   █", "█   █", "█   █", " ███ "],
    "P": ["████ ", "█   █", "█   █", "████ ", "█    ", "█    ", "█    "],
    "Q": [" ███ ", "█   █", "█   █", "█   █", "█ █ █", "█  ██", " ████"],
    "R": ["████ ", "█   █", "█   █", "████ ", "█ █  ", "█  █ ", "█   █"],
    "S": [" ███ ", "█   █", "█    ", " ███ ", "    █", "█   █", " ███ "],
    "T": ["█████", "  █  ", "  █  ", "  █  ", "  █  ", "  █  ", "  █  "],
    "U": ["█   █", "█   █", "█   █", "█   █", "█   █", "█   █", " ███ "],
    "V": ["█   █", "█   █", "█   █", "█   █", " █ █ ", " █ █ ", "  █  "],
    "W": ["█   █", "█   █", "█   █", "█ █ █", "█ █ █", "██ ██", "█   █"],
    "X": ["█   █", "█   █", " █ █ ", "  █  ", " █ █ ", "█   █", "█   █"],
    "Y": ["█   █", "█   █", " █ █ ", "  █  ", "  █  ", "  █  ", "  █  "],
    "Z": ["█████", "    █", "   █ ", "  █  ", " █   ", "█    ", "█████"],
    "0": [" ███ ", "█  ██", "█ █ █", "██  █", "█   █", "█   █", " ███ "],
    "1": ["  █  ", " ██  ", "  █  ", "  █  ", "  █  ", "  █  ", "█████"],
    "2": [" ███ ", "█   █", "    █", "   █ ", "  █  ", " █   ", "█████"],
    "3": [" ███ ", "█   █", "    █", "  ██ ", "    █", "█   █", " ███ "],
    "4": ["   █ ", "  ██ ", " █ █ ", "█  █ ", "█████", "   █ ", "   █ "],
    "5": ["█████", "█    ", "████ ", "    █", "    █", "█   █", " ███ "],
    "6": [" ███ ", "█   █", "█    ", "████ ", "█   █", "█   █", " ███ "],
    "7": ["█████", "    █", "   █ ", "  █  ", "  █  ", "  █  ", "  █  "],
    "8": [" ███ ", "█   █", "█   █", " ███ ", "█   █", "█   █", " ███ "],
    "9": [" ███ ", "█   █", "█   █", " ████", "    █", "█   █", " ███ "],
    " ": ["     ", "     ", "     ", "     ", "     ", "     ", "     "],
    ".": ["     ", "     ", "     ", "     ", "     ", "  █  ", "     "],
    "-": ["     ", "     ", "     ", "█████", "     ", "     ", "     "],
    "_": ["     ", "     ", "     ", "     ", "     ", "     ", "█████"],
    "!": ["  █  ", "  █  ", "  █  ", "  █  ", "  █  ", "     ", "  █  "],
    "?": [" ███ ", "█   █", "   █ ", "  █  ", "  █  ", "     ", "  █  "],
    ":": ["     ", "  █  ", "     ", "     ", "  █  ", "     ", "     "],
    "/": ["    █", "   █ ", "   █ ", "  █  ", " █   ", " █   ", "█    "],
}

ICONS = {
    "dna": "🧬",
    "bolt": "⚡",
    "fire": "🔥",
    "star": "⭐",
    "rocket": "🚀",
    "sparkles": "✨",
    "crystal": "💎",
    "brain": "🧠",
}

CHARS_PER_ROW = 5


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


def build_block_text(text, style, start_color, end_color, no_color):
    text = text.upper()
    rows = [""] * 7

    for ch in text:
        glyph = BLOCK_FONT.get(ch, BLOCK_FONT[" "])
        for i in range(7):
            rows[i] += glyph[i] + " "

    for i in range(7):
        rows[i] = rows[i].rstrip()

    max_width = max(len(r) for r in rows)
    total_chars = sum(len(rows[i]) for i in range(7))
    chars_per_row = sum(1 for c in rows[0] if c not in " \n")

    if no_color or style == "plain":
        return "\n".join(rows)

    gradient = rgb_gradient(start_color, end_color, total_chars or 1)

    result_rows = []
    char_idx = 0
    for i in range(7):
        colored = ""
        for ch in rows[i]:
            if ch != " " and char_idx < len(gradient):
                r, g, b = gradient[char_idx]
                colored += f"{ansi_truecolor(r, g, b)}{ch}{ansi_reset()}"
                char_idx += 1
            else:
                colored += ch
        result_rows.append(colored)

    return "\n".join(result_rows)


def build_box(text_rows, subtitle, width, start_color, end_color, no_color):
    lines = text_rows.split("\n")
    total_width = visible_len(lines[0]) if lines else 0
    box_width = total_width + 4
    if width and width > box_width:
        box_width = width

    top = f"╔{'═' * (box_width - 2)}╗"
    bottom = f"╚{'═' * (box_width - 2)}╝"

    if no_color:
        top = top
        bottom = bottom
    else:
        top = f"{ansi_truecolor(*start_color)}{top}{ansi_reset()}"
        bottom = f"{ansi_truecolor(*start_color)}{bottom}{ansi_reset()}"

    padded_lines = []
    for line in lines:
        vlen = visible_len(line)
        padding = box_width - 2 - vlen
        left_pad = padding // 2
        right_pad = padding - left_pad
        if no_color:
            padded = f"║{' ' * left_pad}{line}{' ' * right_pad}║"
        else:
            padded = (f"{ansi_truecolor(*start_color)}║{ansi_reset()}"
                      f"{' ' * left_pad}{line}{' ' * right_pad}"
                      f"{ansi_truecolor(*start_color)}║{ansi_reset()}")
        padded_lines.append(padded)

    result = [top] + padded_lines

    if subtitle:
        sub_width = len(subtitle)
        sub_padding = (box_width - sub_width - 2)
        sub_left = sub_padding // 2
        sub_right = sub_padding - sub_left
        if no_color:
            sub_line = f"║{' ' * sub_left}{subtitle}{' ' * sub_right}║"
        else:
            sub_grad = rgb_gradient(start_color, end_color, len(subtitle))
            colored_sub = ""
            for i, ch in enumerate(subtitle):
                r, g, b = sub_grad[i] if i < len(sub_grad) else start_color
                colored_sub += f"{ansi_truecolor(r, g, b)}{ch}{ansi_reset()}"
            sub_line = (f"{ansi_truecolor(*start_color)}║{ansi_reset()}"
                        f"{' ' * sub_left}{colored_sub}{' ' * sub_right}"
                        f"{ansi_truecolor(*start_color)}║{ansi_reset()}")
        result.append(sub_line)

    result.append(bottom)

    return "\n".join(result)


def build_banner(text_rows, subtitle, width, start_color, end_color, no_color):
    lines = text_rows.split("\n")
    total_width = visible_len(lines[0]) if lines else 0
    banner_width = total_width + 4

    sep = "─" * banner_width
    if no_color:
        sep_top = f"┌{sep}┐"
        sep_bot = f"└{sep}┘"
    else:
        gr = rgb_gradient(start_color, end_color, banner_width)
        sep_top_chars = ""
        sep_bot_chars = ""
        for i in range(banner_width):
            r, g, b = gr[i] if i < len(gr) else start_color
            sep_top_chars += f"{ansi_truecolor(r, g, b)}─{ansi_reset()}"
            sep_bot_chars += f"{ansi_truecolor(r, g, b)}─{ansi_reset()}"
        sep_top = f"{ansi_truecolor(*start_color)}┌{ansi_reset()}{sep_top_chars}{ansi_truecolor(*start_color)}┐{ansi_reset()}"
        sep_bot = f"{ansi_truecolor(*start_color)}└{ansi_reset()}{sep_bot_chars}{ansi_truecolor(*end_color)}┘{ansi_reset()}"

    result = [sep_top]

    for line in lines:
        vlen = visible_len(line)
        padding = total_width - vlen
        left_pad = padding // 2
        right_pad = padding - left_pad
        padded = f"{' ' * left_pad}{line}{' ' * right_pad}"
        result.append(padded)

    if subtitle:
        sub_vlen = visible_len(subtitle) if not no_color else len(subtitle)
        sub_pad = (total_width - sub_vlen) // 2
        sub_line = f"{' ' * sub_pad}{subtitle}"
        if not no_color:
            sub_grad = rgb_gradient(start_color, end_color, len(subtitle))
            colored_sub = ""
            for i, ch in enumerate(subtitle):
                r, g, b = sub_grad[i] if i < len(sub_grad) else start_color
                colored_sub += f"{ansi_truecolor(r, g, b)}{ch}{ansi_reset()}"
            sub_line = f"{' ' * sub_pad}{colored_sub}{' ' * (total_width - sub_pad - len(subtitle))}"
        result.append(sub_line)

    result.append(sep_bot)

    return "\n".join(result)


def build_minimal(text_rows, subtitle, width, start_color, end_color, no_color):
    result = [text_rows]
    if subtitle:
        prefix = "  "
        if no_color:
            result.append(f"{prefix}{subtitle}")
        else:
            sub_grad = rgb_gradient(start_color, end_color, len(subtitle))
            colored_sub = ""
            for i, ch in enumerate(subtitle):
                r, g, b = sub_grad[i] if i < len(sub_grad) else start_color
                colored_sub += f"{ansi_truecolor(r, g, b)}{ch}{ansi_reset()}"
            result.append(f"{prefix}{colored_sub}")

    return "\n".join(result)


def generate_logo(text, subtitle, palette_name, style, width, icons, no_color, output_file):
    palette = PALETTES.get(palette_name, PALETTES["boi-purple"])
    start_color, end_color = palette

    text_rows = build_block_text(text, style, start_color, end_color, no_color)

    if style == "boxed":
        result = build_box(text_rows, subtitle, width, start_color, end_color, no_color)
    elif style == "banner":
        result = build_banner(text_rows, subtitle, width, start_color, end_color, no_color)
    elif style == "minimal":
        result = build_minimal(text_rows, subtitle, width, start_color, end_color, no_color)
    else:
        result = text_rows.split("\n")
        if subtitle:
            result.append(subtitle)
        result = "\n".join(result)

    if icons:
        icon_str = " ".join(ICONS.get(i, i) for i in icons)
        result = f"{icon_str}\n{result}"

    if output_file:
        with open(output_file, "w", encoding="utf-8") as f:
            f.write(result + "\n")
        print(f"Logo saved to {output_file}")

    return result


def list_palettes():
    print("Available palettes:")
    for name, (start, end) in PALETTES.items():
        r1, g1, b1 = start
        r2, g2, b2 = end
        print(f"  {ansi_truecolor(r1, g1, b1)}{name}{ansi_reset()}"
              f"  ({ansi_truecolor(r1, g1, b1)}#{r1:02X}{g1:02X}{b1:02X}{ansi_reset()}"
              f" → {ansi_truecolor(r2, g2, b2)}#{r2:02X}{g2:02X}{b2:02X}{ansi_reset()})")


def main():
    parser = argparse.ArgumentParser(
        description="Generate BOI CLI ASCII logo with ANSI gradients",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=textwrap.dedent("""\
            Examples:
              %(prog)s --text "BOI CLI" --subtitle "Chimera Architecture"
              %(prog)s --text "BOI" --palette boi-cyan --style minimal
              %(prog)s --text "BOI" --palette boi-fire --icons bolt,fire
              %(prog)s --list-palettes
              %(prog)s --text "BOI" --output logo.txt
        """),
    )

    parser.add_argument("--text", default="BOI CLI",
                        help="Main text to render (default: 'BOI CLI')")
    parser.add_argument("--subtitle", default=None,
                        help="Subtitle text below the logo")
    parser.add_argument("--palette", default="boi-purple",
                        choices=list(PALETTES.keys()),
                        help="Color palette preset (default: boi-purple)")
    parser.add_argument("--style", default="boxed",
                        choices=["boxed", "banner", "minimal", "plain"],
                        help="Logo style (default: boxed)")
    parser.add_argument("--width", type=int, default=0,
                        help="Minimum box width (default: auto-fit content)")
    parser.add_argument("--no-color", action="store_true",
                        help="Disable ANSI colors (also respects NO_COLOR env var)")
    parser.add_argument("--output", default=None,
                        help="Save output to file")
    parser.add_argument("--icons", default=None,
                        help="Comma-separated icons to display above logo: dna,bolt,fire,star,rocket,sparkles,crystal,brain")
    parser.add_argument("--list-palettes", action="store_true",
                        help="Show available palettes and exit")
    parser.add_argument("--version", action="version",
                        version=f"BOI Logo Generator v{VERSION}")

    args = parser.parse_args()

    if args.list_palettes:
        list_palettes()
        return

    no_color = args.no_color or os.environ.get("NO_COLOR", "").strip() != ""

    icons = []
    if args.icons:
        icons = [i.strip() for i in args.icons.split(",")]

    result = generate_logo(
        text=args.text,
        subtitle=args.subtitle,
        palette_name=args.palette,
        style=args.style,
        width=args.width,
        icons=icons,
        no_color=no_color,
        output_file=args.output,
    )

    if not args.output:
        print(result)


if __name__ == "__main__":
    main()
