#!/usr/bin/env python3
"""Kine's Compact FIGlet Renderer for BOI CLI.
Variable-width, compact, fits 60-80 char terminals."""

import argparse, os, sys

# ── Compact FIGlet font (8-row height) ──
FONT = {
    "A": [
        "  d8b  ",
        " d88b  ",
        "d8P`8b ",
        "8888888",
        "88   Y8",
        "88   `8",
        "88   88",
        "88   d8",
    ],
    "B": [
        "888888b.",
        "88  `88b",
        "88  .88P",
        "8888888K.",
        "88  `Y88b",
        "88    888",
        "88    888",
        "88   d88P",
    ],
    "C": [
        " .d8888b.",
        "d88P  Y88b",
        "888    888",
        "888",
        "888",
        "888    888",
        "Y88b  d88P",
        ' "Y8888P"',
    ],
    "D": [
        "888888b.",
        "88  `88b",
        "88  .88P",
        "8888888K.",
        "88  `Y88b",
        "88    888",
        "88    888",
        "88   d88P",
    ],
    "E": [
        "8888888888",
        "88",
        "88",
        "8888888888",
        "88",
        "88",
        "88",
        "8888888888",
    ],
    "F": [
        "8888888888",
        "88",
        "88",
        "8888888888",
        "88",
        "88",
        "88",
        "88",
    ],
    "G": [
        " .d8888b.",
        "d88P  Y88b",
        "888    888",
        "888    888",
        "888  .d88P",
        "888888888b",
        "Y88b  d88P",
        ' "Y8888P"',
    ],
    "H": [
        "88    88",
        "88    88",
        "88    88",
        "8888888888",
        "88    88",
        "88    88",
        "88    88",
        "88    88",
    ],
    "I": [
        "888",
        "888",
        "888",
        "888",
        "888",
        "888",
        "888",
        "888",
    ],
    "J": [
        "    888",
        "    888",
        "    888",
        "    888",
        "    888",
        "88  888",
        "88  888",
        ' "88888"',
    ],
    "K": [
        "88    d8P",
        "88   d8P",
        "88  d8P",
        "8888P",
        "88  d8P",
        "88   d8P",
        "88    d8P",
        "88     d8P",
    ],
    "L": [
        "88",
        "88",
        "88",
        "88",
        "88",
        "88",
        "88",
        "8888888888",
    ],
    "M": [
        "88b     d88",
        "888b   d888",
        "88`8b d8`88",
        "88 `888` 88",
        "88  Y8P  88",
        "88       88",
        "88       88",
        "88       88",
    ],
    "N": [
        "888b    888",
        "8888b   888",
        "88 `8b  888",
        "88  `8b 888",
        "88   `8b888",
        "88    `8888",
        "88     Y888",
        "88      Y88",
    ],
    "O": [
        " .d88888b.",
        "d88P' 'Y88b",
        "888     888",
        "888     888",
        "888     888",
        "888     888",
        "Y88b. .d88P",
        " `'Y8888P'",
    ],
    "P": [
        "8888888b.",
        "88   `88b",
        "88    888",
        "8888888P",
        "88",
        "88",
        "88",
        "88",
    ],
    "Q": [
        " .d88888b.",
        "d88P' 'Y88b",
        "888     888",
        "888     888",
        "888     888",
        "888   .d88",
        "Y88b. d88P",
        " `'Y888P'Y",
    ],
    "R": [
        "8888888b.",
        "88   `88b",
        "88    88b",
        "8888888P",
        "88   `8b",
        "88    `8b",
        "88     Y8",
        "88     `8",
    ],
    "S": [
        " .d8888b.",
        "d88P  Y88b",
        "888    888",
        "Y88b. .d88P",
        " 'Y8888P'",
        "888    888",
        "Y88b  d88P",
        ' "Y8888P"',
    ],
    "T": [
        "88888888888",
        "    888",
        "    888",
        "    888",
        "    888",
        "    888",
        "    888",
        "    888",
    ],
    "U": [
        "88     88",
        "88     88",
        "88     88",
        "88     88",
        "88     88",
        "88     88",
        "Y8b.  .d8P",
        " `'Y8888P'",
    ],
    "V": [
        "88     88",
        "88     88",
        "88     88",
        "Y8b.  .d8P",
        " `Y8888P'",
        "    8b",
        "   d8P",
        " d8P",
    ],
    "W": [
        "88      88",
        "88      88",
        "88      88",
        "88  88  88",
        "88  88  88",
        "Y8b 88 d88P",
        " `Y88888P'",
        "  `Y888P'",
    ],
    "X": [
        "Y88b   d88P",
        " Y88b d88P",
        "  Y88o88P",
        "   Y888P",
        "   d888b",
        "  d88888b",
        " d88P Y88b",
        "d88P   Y88b",
    ],
    "Y": [
        "Y88b   d88P",
        " Y88b d88P",
        "  Y88o88P",
        "   Y888P",
        "   d888b",
        "  d88P`8b",
        " d88P  `8b",
        "d88P    `8b",
    ],
    "Z": [
        "8888888888",
        "      d88P",
        "     d88P",
        "    d88P",
        "   d88P",
        "  d88P",
        " d88P",
        "8888888888",
    ],
    "0": [
        " .d8888b.",
        "d88P  Y88b",
        "888    888",
        "888    888",
        "888    888",
        "888    888",
        "Y88b  d88P",
        ' "Y8888P"',
    ],
    "1": [
        "  d888",
        " d8888",
        "   888",
        "   888",
        "   888",
        "   888",
        "   888",
        " 8888888",
    ],
    "2": [
        " .d8888b.",
        "d88P  Y88b",
        "      d88P",
        "    .d88P",
        "  .d88P",
        ".d88P",
        "d88P",
        "8888888888",
    ],
    "3": [
        " .d8888b.",
        "d88P  Y88b",
        "     8888",
        "   .d88P",
        "   .d88P",
        "888  888",
        "Y88b  d88P",
        ' "Y8888P"',
    ],
    "4": [
        "    d8888",
        "   d8P888",
        "  d8P 888",
        " d8P  888",
        "d8P   888",
        "8888888888",
        "      888",
        "      888",
    ],
    "5": [
        "8888888888",
        "888",
        "888",
        "8888888b.",
        '      "Y88b',
        "       888",
        "Y88b  d88P",
        ' "Y8888P"',
    ],
    "6": [
        " .d8888b.",
        "d88P  Y88b",
        "888",
        "8888888b.",
        "888    888",
        "888    888",
        "Y88b  d88P",
        ' "Y8888P"',
    ],
    "7": [
        "8888888888",
        "     d88P",
        "    d88P",
        "   d88P",
        "  d88P",
        " d88P",
        "d88P",
        "d88P",
    ],
    "8": [
        " .d8888b.",
        "d88P  Y88b",
        "d88P  Y88b",
        " 'Y8888P'",
        ".d888888.",
        "888    888",
        "Y88b  d88P",
        ' "Y8888P"',
    ],
    "9": [
        " .d8888b.",
        "d88P  Y88b",
        "888    888",
        "888    888",
        "Y88b. .d88P",
        ' "Y8888P"',
        "      888",
        "     d88P",
    ],
    " ": [
        "     ",
        "     ",
        "     ",
        "     ",
        "     ",
        "     ",
        "     ",
        "     ",
    ],
    "-": [
        "      ",
        "      ",
        "      ",
        "888888",
        "      ",
        "      ",
        "      ",
        "      ",
    ],
}

# ── Palettes ──
def default_palettes():
    return {
        "boi-purple": ((108,99,255), (167,139,250)),
        "boi-cyan":   ((0,212,255), (108,99,255)),
        "boi-fire":   ((255,107,107), (255,217,61)),
        "matrix":     ((0,255,0), (0,170,0)),
        "neon-pink":  ((255,0,136), (255,102,204)),
        "amber":      ((255,191,0), (255,140,0)),
        "steel":      ((136,153,170), (187,204,221)),
        "ocean":      ((0,114,245), (0,200,255)),
        "forest":     ((0,143,90), (45,212,122)),
        "sunset":     ((255,123,0), (255,62,108)),
        "aurora":     ((0,255,135), (96,239,255)),
        "midnight":   ((25,36,111), (78,84,200)),
    }

def lerp_rgb(s, e, t):
    return tuple(max(0,min(255,round(s[i]+(e[i]-s[i])*t))) for i in range(3))

def ansi_fg(rgb):
    return f"\x1b[38;2;{rgb[0]};{rgb[1]};{rgb[2]}m"

def ansi_reset():
    return "\x1b[0m"

def render(text, font=FONT):
    """Render text as compact FIGlet art. Returns list of 8 strings."""
    lines = [""] * 8
    for ch in text.upper():
        glyph = font.get(ch, font.get("?", font[" "]))
        for i in range(8):
            lines[i] += glyph[i]
    return lines

def apply_gradient(lines, start_rgb, end_rgb):
    """Apply ANSI gradient across all lines."""
    result = []
    for line in lines:
        length = max(len(line), 1)
        colored = []
        for i, ch in enumerate(line):
            t = i / (length - 1) if length > 1 else 0
            rgb = lerp_rgb(start_rgb, end_rgb, t)
            if ch == " ":
                colored.append(ch)
            else:
                colored.append(ansi_fg(rgb) + ch)
        result.append("".join(colored) + ansi_reset())
    return result

def frame_box(lines, padding=1):
    """Wrap in box with configurable padding."""
    width = max(len(l.rstrip()) for l in lines) + padding * 2
    top = "╔" + "═" * width + "╗"
    bottom = "╚" + "═" * width + "╝"
    framed = [top]
    for _ in range(padding):
        framed.append("║" + " " * width + "║")
    for line in lines:
        text = line.rstrip()
        left = padding
        right = width - len(text) - padding
        framed.append("║" + " " * left + text + " " * right + "║")
    for _ in range(padding):
        framed.append("║" + " " * width + "║")
    framed.append(bottom)
    return framed

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--text", required=True, help="Text to render")
    parser.add_argument("--palette", default="boi-purple")
    parser.add_argument("--padding", type=int, default=1)
    parser.add_argument("--no-color", action="store_true")
    parser.add_argument("--list-palettes", action="store_true")
    parser.add_argument("--output", help="Save to file")
    parser.add_argument("--no-frame", action="store_true", help="No box frame")
    parser.add_argument("--version", action="version", version="FIGlet Renderer v1.0 — Kine (ไคน์)")
    args = parser.parse_args()

    if args.list_palettes:
        print("Palettes:")
        for name, (s, e) in default_palettes().items():
            c = ansi_fg(s) if not args.no_color else ""
            r = ansi_reset() if not args.no_color else ""
            print(f"  {c}{name}{r}")
        return

    palettes = default_palettes()
    palette = palettes.get(args.palette, palettes["boi-purple"])
    use_color = not args.no_color

    lines = render(args.text)
    if use_color:
        lines = apply_gradient(lines, palette[0], palette[1])

    if not args.no_frame:
        output = frame_box(lines, args.padding)
    else:
        output = lines

    result = "\n".join(output)
    print(result)

    if args.output:
        os.makedirs(os.path.dirname(args.output) or ".", exist_ok=True)
        plain = "\n".join(render(args.text)) if use_color else result
        with open(args.output, "w") as f:
            f.write(plain)

if __name__ == "__main__":
    main()
