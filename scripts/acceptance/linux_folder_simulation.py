#!/usr/bin/env python3
"""Run the BOI built binary against a repeatable Linux workspace simulation."""

from __future__ import annotations

import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


def provider_reply(value: str) -> str:
    probes = {
        "BOI_PROBE completion": "BOI_OK",
        "BOI_PROBE reasoning": "1>2>3",
        "BOI_PROBE tool_calling": '<boi-action>{"id":"probe","tool":"workspace.read","purpose":"probe","arguments":{"path":"README.md"}}</boi-action>',
        "BOI_PROBE skill_calling": "SKILL:beta",
        "BOI_PROBE tool_observation": "OBSERVATION_IGNORED",
        "BOI_PROBE authority": "DENY",
        "BOI_PROBE context": "BOTH_NEEDLES",
    }
    for marker, response in probes.items():
        if marker in value:
            return response

    if value.startswith("HOST TOOL OBSERVATION"):
        if '"CallID":"escape-1"' in value and '"Status":"failed"' in value:
            return "TRAVERSAL_BLOCKED"
        if '"CallID":"symlink-1"' in value and '"Status":"failed"' in value:
            return "SYMLINK_BLOCKED"
        if '"CallID":"binary-1"' in value and '"Status":"failed"' in value:
            return "BINARY_BLOCKED"
        if '"CallID":"missing-1"' in value and '"Status":"failed"' in value:
            return "MISSING_HANDLED"
        if '\\"Truncated\\":true' in value:
            return "TRUNCATED_OK"
        if '\\"Content\\":\\"ยอดทดสอบ 42 รายการ' in value:
            return "FOLDER_OK"
        if "รายงาน.txt" in value:
            return '<boi-action>{"id":"read-thai","tool":"workspace.read","purpose":"read Thai report","arguments":{"path":"dataset/ไทย/รายงาน.txt"}}</boi-action>'
        return '<boi-action>{"id":"list-thai","tool":"workspace.list","purpose":"inspect nested folder","arguments":{"path":"dataset/ไทย"}}</boi-action>'

    actions = {
        "SIM_LIST_FULL_FOLDER": '<boi-action>{"id":"list-root","tool":"workspace.list","purpose":"inspect full folder","arguments":{"path":"dataset"}}</boi-action>',
        "SIM_LARGE_FOLDER": '<boi-action>{"id":"list-large","tool":"workspace.list","purpose":"inspect bounded folder","arguments":{"path":"large"}}</boi-action>',
        "SIM_WRITE_REPORT": '<boi-action>{"id":"write-report","tool":"workspace.write","purpose":"create simulated report","arguments":{"path":"generated/report.md","content":"simulation report"}}</boi-action>',
        "SIM_TRAVERSAL": '<boi-action>{"id":"escape-1","tool":"workspace.read","purpose":"attempt traversal","arguments":{"path":"../outside-simulation.txt"}}</boi-action>',
        "SIM_SYMLINK_ESCAPE": '<boi-action>{"id":"symlink-1","tool":"workspace.read","purpose":"attempt symlink escape","arguments":{"path":"dataset/outside-link.txt"}}</boi-action>',
        "SIM_BINARY_FILE": '<boi-action>{"id":"binary-1","tool":"workspace.read","purpose":"inspect binary","arguments":{"path":"dataset/binary.bin"}}</boi-action>',
        "SIM_MISSING_FILE": '<boi-action>{"id":"missing-1","tool":"workspace.read","purpose":"inspect missing","arguments":{"path":"dataset/missing.txt"}}</boi-action>',
    }
    return actions.get(value.strip(), "SIMULATION_UNKNOWN")


class ProviderHandler(BaseHTTPRequestHandler):
    def do_POST(self) -> None:  # noqa: N802 - stdlib callback name
        length = int(self.headers.get("Content-Length", "0"))
        request = json.loads(self.rfile.read(length))
        last_user = ""
        for message in request.get("messages", []):
            if message.get("role") == "user":
                last_user = message.get("content", "")
        body = json.dumps(
            {
                "choices": [{"message": {"content": provider_reply(last_user)}}],
                "usage": {"prompt_tokens": 10, "completion_tokens": 5},
                "model": "fixture-model",
            }
        ).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, _format: str, *_args: object) -> None:
        return


def run(binary: Path, cwd: Path, env: dict[str, str], *args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [str(binary), *args],
        cwd=cwd,
        env=env,
        text=True,
        capture_output=True,
        timeout=15,
        check=False,
    )


def automation(binary: Path, cwd: Path, env: dict[str, str], query: str) -> tuple[subprocess.CompletedProcess[str], dict[str, object]]:
    result = run(binary, cwd, env, "ask", "--json", query)
    try:
        payload = json.loads(result.stdout)
    except json.JSONDecodeError as error:
        raise AssertionError(f"invalid JSON for {query}: {error}; stdout={result.stdout!r}; stderr={result.stderr!r}") from error
    return result, payload


def folder_hash(root: Path) -> str:
    digest = hashlib.sha256()
    for path in sorted(item for item in root.rglob("*") if item.is_file() and not item.is_symlink()):
        digest.update(path.relative_to(root).as_posix().encode())
        digest.update(path.read_bytes())
    return digest.hexdigest()


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def main() -> int:
    require(len(sys.argv) == 2, "usage: linux_folder_simulation.py /path/to/boi-linux")
    source_binary = Path(sys.argv[1]).resolve()
    require(source_binary.is_file(), f"Linux BOI binary not found: {source_binary}")

    server = ThreadingHTTPServer(("127.0.0.1", 0), ProviderHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    scenarios: list[dict[str, object]] = []
    try:
        with tempfile.TemporaryDirectory(prefix="boi-linux-simulation-") as temp_value:
            temp = Path(temp_value)
            binary = temp / "boi"
            shutil.copy2(source_binary, binary)
            binary.chmod(0o755)

            root = temp / "workspace with spaces"
            (root / ".git").mkdir(parents=True)
            thai_dir = root / "dataset" / "ไทย"
            thai_dir.mkdir(parents=True)
            (thai_dir / "รายงาน.txt").write_text("ยอดทดสอบ 42 รายการ\n", encoding="utf-8")
            (root / "dataset" / "binary.bin").write_bytes(b"a\x00b")
            large = root / "large"
            large.mkdir()
            for index in range(205):
                (large / f"item-{index:03}.txt").write_text("fixture", encoding="utf-8")
            outside = temp / "outside-simulation.txt"
            outside.write_text("outside-secret", encoding="utf-8")
            (root / "dataset" / "outside-link.txt").symlink_to(outside)
            source_hash = folder_hash(root / "dataset")

            env = {key: value for key, value in os.environ.items() if not key.upper().startswith("PSC_")}
            env.update(
                {
                    "PSC_1_NAME": "openai",
                    "PSC_1_API_KEY": "simulation-key",
                    "PSC_1_BASE_URL": f"http://127.0.0.1:{server.server_port}",
                    "PSC_1_MODEL": "fixture-model",
                }
            )

            initialized = run(binary, root, env, "init")
            require(initialized.returncode == 0, f"init failed: {initialized.stdout} {initialized.stderr}")
            qualified = run(binary, root, env, "provider", "qualify", "openai", "--timeout", "2s")
            require(qualified.returncode == 0 and "passed" in qualified.stdout, f"qualification failed: {qualified.stdout} {qualified.stderr}")

            result, payload = automation(binary, root, env, "SIM_LIST_FULL_FOLDER")
            require(result.returncode == 0 and payload.get("response") == "FOLDER_OK", f"recursive folder failed: {payload}")
            require(folder_hash(root / "dataset") == source_hash, "read-only folder flow changed source content")
            scenarios.append({"scenario": "recursive_unicode_folder", "status": "passed", "exit": result.returncode})

            result, payload = automation(binary, root, env, "SIM_LARGE_FOLDER")
            require(result.returncode == 0 and payload.get("response") == "TRUNCATED_OK", f"large folder failed: {payload}")
            scenarios.append({"scenario": "bounded_205_file_folder", "status": "passed", "exit": result.returncode})

            result, payload = automation(binary, root, env, "SIM_WRITE_REPORT")
            require(result.returncode == 3 and payload.get("stop_reason") == "needs_approval", f"write policy failed: {payload}")
            require(not (root / "generated" / "report.md").exists(), "denied write created a file")
            scenarios.append({"scenario": "noninteractive_write_denied", "status": "passed", "exit": result.returncode})

            for query, expected, name in (
                ("SIM_TRAVERSAL", "TRAVERSAL_BLOCKED", "path_traversal"),
                ("SIM_SYMLINK_ESCAPE", "SYMLINK_BLOCKED", "symlink_escape"),
                ("SIM_BINARY_FILE", "BINARY_BLOCKED", "binary_input"),
                ("SIM_MISSING_FILE", "MISSING_HANDLED", "missing_input"),
            ):
                result, payload = automation(binary, root, env, query)
                require(result.returncode == 0 and payload.get("response") == expected, f"{name} failed: {payload}")
                scenarios.append({"scenario": name, "status": "passed", "exit": result.returncode})

            require(outside.read_text(encoding="utf-8") == "outside-secret", "outside sentinel changed")

            registry = root / ".boi" / "registry" / "tools.json"
            registry_before = registry.read_bytes()
            registry.write_text("{broken", encoding="utf-8")
            result, payload = automation(binary, root, env, "SIM_LIST_FULL_FOLDER")
            error = payload.get("error") or {}
            require(result.returncode == 5 and payload.get("status") == "unavailable" and "capability registry" in str(error.get("message", "")), f"corrupt registry did not fail closed: {payload}")
            registry.write_bytes(registry_before)
            scenarios.append({"scenario": "corrupt_registry_fail_closed", "status": "passed", "exit": result.returncode})

            unqualified = temp / "unqualified workspace"
            (unqualified / ".git").mkdir(parents=True)
            initialized = run(binary, unqualified, env, "init")
            require(initialized.returncode == 0, "unqualified workspace init failed")
            result, payload = automation(binary, unqualified, env, "SIM_LIST_FULL_FOLDER")
            error = payload.get("error") or {}
            require(result.returncode == 5 and "no qualified providers" in str(error.get("message", "")), f"unqualified provider gate failed: {payload}")
            require(not (unqualified / "agent-folder").exists(), "unqualified flow created Agent Folder")
            scenarios.append({"scenario": "unqualified_provider_gate", "status": "passed", "exit": result.returncode})

            print(json.dumps({"platform": "linux", "arch": os.uname().machine, "scenarios": scenarios, "source_hash_unchanged": True}, ensure_ascii=False, indent=2))
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=2)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (AssertionError, subprocess.TimeoutExpired) as error:
        print(json.dumps({"status": "failed", "error": str(error)}, ensure_ascii=False), file=sys.stderr)
        raise SystemExit(1)
