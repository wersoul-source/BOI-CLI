#!/usr/bin/env python3
"""DOS/ANSI-Shadow FIGlet font generator — ▓ blocks, 9-row glyphs for A-Z."""

import argparse
import os
import sys

# ── DOS/ANSI-Shadow font (9 rows, variable width, ▓ fill) ──
DOS_FONT: dict[str, list[str]] = {
    "A": [
        "   ___   ",
        "  /   \\  ",
        " |  ▓▓▓▓| ",
        " | ▓▓▓▓/  ",
        " | ▓▓▓▓▓\\ ",
        " | ▓▓| ▓▓ ",
        " | ▓▓| ▓▓ ",
        " | ▓▓| ▓▓ ",
        "  \\▓▓▓▓▓▓",
    ],
    "B": [
        "_______ ",
        "|       \\",
        "| ▓▓▓▓▓▓▓\\",
        "| ▓▓__/ ▓▓",
        "| ▓▓    ▓▓",
        "| ▓▓▓▓▓▓▓\\",
        "| ▓▓__/ ▓▓",
        "| ▓▓    ▓▓",
        " \\▓▓▓▓▓▓▓",
    ],
    "C": [
        " ______ ",
        "/      \\",
        "|  ▓▓▓▓▓▓\\",
        "| ▓▓__/ ▓▓",
        "| ▓▓      ",
        "| ▓▓      ",
        "| ▓▓__/ ▓▓",
        "|  \\▓▓▓▓▓▓",
        " \\_____/",
    ],
    "D": [
        "_______ ",
        "|       \\",
        "| ▓▓▓▓▓▓▓\\",
        "| ▓▓    ▓▓",
        "| ▓▓    ▓▓",
        "| ▓▓    ▓▓",
        "| ▓▓__/ ▓▓",
        "|  \\▓▓▓▓▓▓",
        " \\_____/",
    ],
    "E": [
        "_______ ",
        "|       \\",
        "| ▓▓▓▓▓▓▓\\",
        "| ▓▓      ",
        "| ▓▓▓▓▓▓▓\\",
        "| ▓▓      ",
        "| ▓▓__/ ▓▓",
        "|  \\▓▓▓▓▓▓",
        " \\_____/",
    ],
    "F": [
        "_______ ",
        "|       \\",
        "| ▓▓▓▓▓▓▓\\",
        "| ▓▓      ",
        "| ▓▓▓▓▓▓▓\\",
        "| ▓▓      ",
        "| ▓▓      ",
        "| ▓▓      ",
        " \\▓▓▓▓▓▓▓",
    ],
    "G": [
        " ______ ",
        "/      \\",
        "|  ▓▓▓▓▓▓\\",
        "| ▓▓__/ ▓▓",
        "| ▓▓      ",
        "| ▓▓   ▓▓▓",
        "| ▓▓__/ ▓▓",
        "|  \\▓▓▓▓▓▓",
        " \\_____/",
    ],
    "H": [
        "|       |",
        "| ▓▓    ▓▓",
        "| ▓▓    ▓▓",
        "| ▓▓▓▓▓▓▓▓\\",
        "| ▓▓    ▓▓",
        "| ▓▓    ▓▓",
        "| ▓▓    ▓▓",
        "| ▓▓    ▓▓",
        " \\▓▓▓▓▓▓▓",
    ],
    "I": [
        " __ ",
        "|  \\",
        "| ▓▓",
        "| ▓▓",
        "| ▓▓",
        "| ▓▓",
        "| ▓▓",
        "| __/",
        " \\_/",
    ],
    "J": [
        "    ___ ",
        "   |   \\",
        "   |  ▓▓",
        "   |  ▓▓",
        "   |  ▓▓",
        "   |  ▓▓",
        " ▓▓__/ ▓▓",
        "|  \\▓▓▓▓▓",
        " \\_____/",
    ],
    "K": [
        "|      /",
        "| ▓▓  / ",
        "| ▓▓ /  ",
        "| ▓▓▓▓  ",
        "| ▓▓▓▓  ",
        "| ▓▓ \\  ",
        "| ▓▓  \\ ",
        "| ▓▓   \\",
        " \\▓▓▓▓▓▓",
    ],
    "L": [
        " __    ",
        "|  \\   ",
        "| ▓▓   ",
        "| ▓▓   ",
        "| ▓▓   ",
        "| ▓▓   ",
        "| ▓▓___",
        "|  \\▓▓ \\",
        " \\▓▓▓▓▓▓",
    ],
    "M": [
        "|\\      /|",
        "| \\    / ▓▓",
        "|  \\  /  ▓▓",
        "|   \\/   ▓▓",
        "|   /\\   ▓▓",
        "|  /  \\  ▓▓",
        "| /    \\ ▓▓",
        "|/      \\▓▓",
        " \\▓▓▓▓▓▓▓▓",
    ],
    "N": [
        "|\\     |",
        "| \\    ▓▓",
        "|  \\   ▓▓",
        "|   \\  ▓▓",
        "|    \\ ▓▓",
        "| ▓▓  \\▓▓",
        "| ▓▓   \\▓",
        "| ▓▓    \\",
        " \\▓▓▓▓▓▓",
    ],
    "O": [
        " ______ ",
        "/      \\",
        "|  ▓▓▓▓▓▓\\",
        "| ▓▓__/ ▓▓",
        "| ▓▓    ▓▓",
        "| ▓▓    ▓▓",
        "| ▓▓__/ ▓▓",
        "|  \\▓▓▓▓▓▓",
        " \\_____/",
    ],
    "P": [
        "_______ ",
        "|       \\",
        "| ▓▓▓▓▓▓▓\\",
        "| ▓▓__/ ▓▓",
        "| ▓▓▓▓▓▓▓\\",
        "| ▓▓      ",
        "| ▓▓      ",
        "| ▓▓      ",
        " \\▓▓▓▓▓▓▓",
    ],
    "Q": [
        " ______ ",
        "/      \\",
        "|  ▓▓▓▓▓▓\\",
        "| ▓▓__/ ▓▓",
        "| ▓▓    ▓▓",
        "| ▓▓   ▓▓▓",
        "| ▓▓__/ ▓▓",
        "|  \\▓▓▓▓▓▓",
        " \\_____/\\",
    ],
    "R": [
        "_______ ",
        "|       \\",
        "| ▓▓▓▓▓▓▓\\",
        "| ▓▓__/ ▓▓",
        "| ▓▓▓▓▓▓▓\\",
        "| ▓▓\\   ▓▓",
        "| ▓▓ \\  ▓▓",
        "| ▓▓  \\ ▓▓",
        " \\▓▓▓▓▓▓▓",
    ],
    "S": [
        " ______ ",
        "/      \\",
        "|  ▓▓▓▓▓▓\\",
        "| ▓▓__/   ",
        " \\▓▓▓▓▓▓\\",
        " /  ___/ ▓▓",
        "| ▓▓__/ ▓▓",
        "|  \\▓▓▓▓▓▓",
        " \\_____/",
    ],
    "T": [
        "_______ ",
        "\\▓▓▓▓▓▓▓\\",
        "    | ▓▓  ",
        "    | ▓▓  ",
        "    | ▓▓  ",
        "    | ▓▓  ",
        "    | ▓▓  ",
        "    | __/ ",
        "     \\_/ ",
    ],
    "U": [
        "|       |",
        "| ▓▓    ▓▓",
        "| ▓▓    ▓▓",
        "| ▓▓    ▓▓",
        "| ▓▓    ▓▓",
        "| ▓▓    ▓▓",
        "| ▓▓__/ ▓▓",
        "|  \\▓▓▓▓▓▓",
        " \\_____/",
    ],
    "V": [
        "|      /",
        "| ▓▓  / ",
        "| ▓▓ /  ",
        "| ▓▓/   ",
        " \\▓▓\\  ",
        "  \\▓▓\\ ",
        "   \\▓▓\\",
        "    \\▓▓\\",
        "     \\▓▓",
    ],
    "W": [
        "|\\      /|",
        "| \\    / ▓▓",
        "|  \\  /  ▓▓",
        "|   \\/   ▓▓",
        "|   /\\   ▓▓",
        "|  /  \\  ▓▓",
        "| /    \\ ▓▓",
        "|/      \\▓▓",
        " \\▓▓▓▓▓▓▓▓",
    ],
    "X": [
        "\\      /",
        " \\ ▓▓  / ",
        "  \\▓▓ /  ",
        "   \\▓▓/  ",
        "   /▓▓/  ",
        "  / ▓▓\\  ",
        " /  ▓▓ \\ ",
        "/   ▓▓  \\",
        "\\▓▓▓▓▓▓▓▓",
    ],
    "Y": [
        "\\      /",
        " \\ ▓▓  / ",
        "  \\▓▓ /  ",
        "   \\▓▓/  ",
        "   | ▓▓  ",
        "   | ▓▓  ",
        "   | ▓▓  ",
        "   | __/ ",
        "    \\_/  ",
    ],
    "Z": [
        "_______ ",
        "\\▓▓▓▓▓▓▓\\",
        "     / ▓▓ ",
        "    /  ▓▓ ",
        "   /   ▓▓ ",
        "  /    ▓▓ ",
        " /     ▓▓ ",
        "/     ▓▓  ",
        "\\▓▓▓▓▓▓▓▓",
    ],
    " ": [
        "         ",
        "         ",
        "         ",
        "         ",
        "         ",
        "         ",
        "         ",
        "         ",
        "         ",
    ],
    "-": [
        "         ",
        "         ",
        "         ",
        " _______ ",
        "         ",
        "         ",
        "         ",
        "         ",
        "         ",
    ],
    "_": [
        "         ",
        "         ",
        "         ",
        "         ",
        "         ",
        "         ",
        "         ",
        "         ",
        " _______ ",
    ],
    ".": [
        "   ",
        "   ",
        "   ",
        "   ",
        "   ",
        "   ",
        "   ",
        " ▓▓ ",
        " \\▓▓",
    ],
    "!": [
        " __ ",
        "|  \\",
        "| ▓▓",
        "| ▓▓",
        "| ▓▓",
        "| ▓▓",
        "|    ",
        "| ▓▓",
        " \\▓▓",
    ],
}


def render_dos(text: str, font: dict[str, list[str]] | None = None) -> list[str]:
    """Render text as DOS/ANSI-Shadow 9-row FIGlet art."""
    if font is None:
        font = DOS_FONT
    lines = [""] * 9
    for ch in text.upper():
        glyph = font.get(ch, font.get("?", font[" "]))
        width = max(len(row) for row in glyph)
        for i in range(9):
            row = glyph[i]
            lines[i] += row + " " * (width - len(row))
    return [ln.rstrip() for ln in lines]


def render_go_literal(text: str) -> str:
    """Render text and emit a Go string literal for the lines."""
    lines = render_dos(text)
    q = "`"
    out = "var boiLogoDOS = []string{\n"
    for ln in lines:
        safe = ln.replace("\\", "\\\\")
        out += f'\t{q}{safe}{q},\n'
    out += "}"
    return out


def box_lines(lines: list[str], padding: int = 2) -> list[str]:
    """Wrap lines in a box frame."""
    max_width = max(len(ln) for ln in lines)
    inner = max_width + padding * 2
    top = "╔" + "═" * inner + "╗"
    bottom = "╚" + "═" * inner + "╝"
    result = [top]
    for _ in range(padding):
        result.append("║" + " " * inner + "║")
    for ln in lines:
        lp = padding
        rp = inner - len(ln) - padding
        result.append("║" + " " * lp + ln + " " * rp + "║")
    for _ in range(padding):
        result.append("║" + " " * inner + "║")
    result.append(bottom)
    return result


def ansi_fg(r: int, g: int, b: int) -> str:
    return f"\x1b[38;2;{r};{g};{b}m"


ANSI_RESET = "\x1b[0m"


def gradient(lines: list[str]) -> list[str]:
    """Apply purple-cyan gradient across each line."""
    out = []
    for ln in lines:
        width = max(len(ln), 1)
        colored = ""
        for i, ch in enumerate(ln):
            t = i / (width - 1) if width > 1 else 0
            rr = int(108 + (167 - 108) * t)
            gg = int(99 + (139 - 99) * t)
            bb = int(255 + (250 - 255) * t)
            if ch == " ":
                colored += ch
            else:
                colored += ansi_fg(rr, gg, bb) + ch
        out.append(colored + ANSI_RESET)
    return out


def list_glyphs(font: dict[str, list[str]] | None = None) -> str:
    """Return a visual table of all glyphs."""
    if font is None:
        font = DOS_FONT
    out_lines = []
    for ch in sorted(font.keys()):
        if ch in (" ", "-", "_", ".", "!"):
            continue
        out_lines.append(f"\n--- {ch} ---")
        out_lines.extend(font[ch])
    return "\n".join(out_lines)


def main() -> None:
    parser = argparse.ArgumentParser(description="DOS/ANSI-Shadow FIGlet generator")
    parser.add_argument("--text", default="BOI CLI", help="Text to render")
    parser.add_argument("--go-literal", action="store_true", help="Emit Go string array")
    parser.add_argument("--no-color", action="store_true", help="Disable ANSI gradient")
    parser.add_argument("--no-frame", action="store_true", help="Skip box frame")
    parser.add_argument("--list-glyphs", action="store_true", help="Show all glyph designs")
    parser.add_argument("--padding", type=int, default=2, help="Box padding")
    parser.add_argument("--output", help="Save plain text to file")
    args = parser.parse_args()

    if args.list_glyphs:
        print(list_glyphs())
        return

    lines = render_dos(args.text)

    if args.go_literal:
        print(render_go_literal(args.text))
        return

    if not args.no_color:
        lines = gradient(lines)

    if args.no_frame:
        output = "\n".join(lines)
    else:
        output = "\n".join(box_lines(lines, args.padding))

    print(output)

    if args.output:
        os.makedirs(os.path.dirname(args.output) or ".", exist_ok=True)
        plain = "\n".join(render_dos(args.text))
        with open(args.output, "w", encoding="utf-8") as f:
            f.write(plain)
        print(f"\nSaved plain text to: {args.output}")


if __name__ == "__main__":
    main()
