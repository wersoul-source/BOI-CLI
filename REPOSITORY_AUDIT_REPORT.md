# REPOSITORY AUDIT REPORT — BOI Agent Suite (BOI-CLI)

- **วันที่ตรวจ:** 2026-11-20
- **Commit ที่ตรวจ:** `origin/master` = `a05923f` (2026-11-03) และ branch `ccr-ee8da262-0ibcmg` = `8aace34` (ต่างจาก master แค่เอกสาร 2 ไฟล์ของ PR #31)
- **ประเภท:** READ-ONLY ไม่มีการแก้ source, commit, push, merge, ติดตั้ง dependency หรือสร้าง release
- **สัญลักษณ์หลักฐาน:** **[V]** ตรวจจากไฟล์หรือคำสั่งที่รันจริงในรอบนี้ · **NOT VERIFIED** ตรวจไม่ได้ในรอบนี้ พร้อมเหตุผล
- แผนแก้ไขอยู่ในไฟล์คู่กัน: [`STABILIZATION_PLAN.md`](STABILIZATION_PLAN.md)

---

## 1. Executive Summary

BOI-CLI เป็นโค้ด Go ขนาดกลาง (~18.8k บรรทัด + เทสต์ ~10.5k) ที่ **วินัยทางวิศวกรรมดี**: เทสต์ทั้งหมดผ่านพร้อม race detector, coverage รวม 68.3% (เกณฑ์ CI 55%), ไม่พบ secret ใน tree หรือ git history, ขอบเขตชั้นของ 6 Blocks ถูกบังคับด้วยเทสต์ และ finding ทั้ง 8 ข้อของ audit รอบ 2026-11-01 ยังมีเทสต์ regression ครบ

**ไม่พบปัญหาระดับ CRITICAL** แต่มี **HIGH 3 ข้อ** ที่ทำให้ยังเรียกว่า "ใช้งานได้จริงบนโมเดลอ่อน" ไม่ได้:

1. โมเดลที่ token เพี้ยนตอบผิดแล้วระบบรายงานว่างานเสร็จ ไม่มีอะไรจับได้
2. Provider จริงของเจ้าของโครงการ (maxplus) ยังถูกปิด Tool ทั้งหมดด้วยกลไกเดิม ซึ่งขัดกับหลักการ ADR-006 ที่อนุมัติแล้ว
3. โค้ด 3,014 บรรทัดที่เพิ่มหลัง v2.7.1 ยังไม่เคยรันกับ Provider จริงเลย ทดสอบกับ endpoint จำลองเท่านั้น

มี MEDIUM 11 ข้อ และ LOW 14 ข้อ ในนั้น 2 ข้อเป็นข้อบกพร่องของงานที่เพิ่ง merge (PR #29, #30): `reasoning_content` อาจถูกส่งข้าม Provider ตอน failover และ `events.jsonl` เก็บข้อความ error ของ endpoint ทั้งที่เอกสารสัญญาว่าไม่เก็บ

**คำตัดสิน:** พร้อมออกเป็น **patch/minor release สำหรับผู้ใช้กลุ่มเดิม** ได้หลังแก้ MEDIUM 4 ข้อแรก **ยังไม่พร้อม** ประกาศเป็น v3.0.0 หรือ production สำหรับโมเดลอ่อน

| หมวด | คะแนน |
|---|---|
| Architecture | 82 |
| Code Quality | 78 |
| Security | 74 |
| Agent & CLI | 66 |
| Git & CI/CD | 72 |
| Production Readiness | 55 |

---

## 2. Repository Overview

| รายการ | ค่า | หลักฐาน |
|---|---|---|
| ภาษา / toolchain | Go 1.24.2 (เครื่องตรวจใช้ go1.24.7) | `go.mod`, `boi version` [V] |
| ไฟล์ Go | 217 (เทสต์ 83) | `find` [V] |
| บรรทัดโค้ด / เทสต์ | 18,837 / 10,483 | `wc -l` [V] |
| Dependency ตรง | 7 ตัว (cobra, bubbletea, bubbles, lipgloss, uniseg, x/sys, yaml.v3) | `go.mod` [V] |
| Commit บน master | 127 | `git rev-list --count` [V] |
| Tag ล่าสุด | `v2.7.1` | `git tag` [V] |
| โค้ดที่ยังไม่ release | 14 commits, 35 ไฟล์, +3,014 / −38 | `git diff --stat v2.7.1 origin/master` [V] |
| PR ที่เปิดอยู่ | 1 (PR #31) | GitHub API [V] |
| Working tree | สะอาดก่อนเขียนรายงาน | `git status` [V] |

โครงสร้างหลัก: `cmd/boi` (entry point), `internal/{core, service, equipment, runtime, agentfolder, subagent}` (6 Blocks), `internal/transport` (CLI/TUI), `internal/app` (composition root), `internal/platform`, `internal/acceptance`, `internal/architecture` (เทสต์บังคับชั้น), `docs/`, `scripts/`, `packaging/winget`

---

## 3. Architecture Assessment — 82/100

**เกณฑ์:** ขอบเขตชั้นถูกบังคับจริงไหม · โครงสร้างตรงกับวัตถุประสงค์ไหม · dead/duplicate code · ความตรงกันของเอกสารกับโค้ด

**จุดแข็ง**
- ขอบเขตของ Blocks ถูกบังคับด้วย `internal/architecture/deps_test.go` และเทสต์นี้ผ่าน [V]
- เอกสารสถาปัตยกรรมครบและเป็นปัจจุบัน: ADR-006, `ARCHITECTURE_IV.md` (สัญญา 4 ฉบับ, F1–F9, E1–E9), แผน 7 ระยะพร้อม status board
- งานที่เพิ่มหลัง v2.7.1 วางถูก Block: dialect parser อยู่ใน Service, event envelope อยู่ใน Runtime, event log อยู่ใน Agent Folder

**จุดอ่อน**
- **สถาปัตยกรรมที่ออกแบบไว้ยังสร้างไม่ถึงครึ่ง:** เหลือ 37 จาก ~50 ขั้น; `Recipe` (แกนของ ADR-006) ยังไม่มีในโค้ด; `ComposeEnvironment` (`internal/core/qualification.go:148`) ยังเป็นสวิตช์เปิดปิดแบบเดิม [V]
- ไฟล์รวมศูนย์ใหญ่: `runtime/agent/service.go` 841 บรรทัด, `engine.go` 793, `transport/tui/app.go` 676 [V] — `runWith` ใน service.go ทำหลายหน้าที่ (prompt, decider, event, fallback, recorder)
- OpenAI adapter ยังทำ 3 หน้าที่ (transport, protocol, dialect) ตามที่ P1.1 ระบุว่าต้องแยก
- คำเรียกเก่าค้างในผลิตภัณฑ์: "Chimera Architecture", "Phantom DB" (`transport/cli/root.go:17`, `version.go:13`, `memory.go:16,72`) ไม่ตรงกับ 6 Blocks [V]
- **Dead code: NOT VERIFIED** — `deadcode` และ `staticcheck` ไม่มีในเครื่องรอบนี้และกติกาห้ามติดตั้ง (CI รัน staticcheck ผ่านบน `8aace34` เมื่อ 2026-11-03)
- **Duplicate code: NOT VERIFIED** — ไม่ได้รันเครื่องมือวัด; จากการอ่านพบแพตเทิร์นซ้ำเล็กน้อยใน helper ของเทสต์ (`newService` กับ `runZooTaskIn`)

---

## 4. Code Quality Assessment — 78/100

**เกณฑ์:** ผลเทสต์จริง · coverage · static analysis · การจัดการ error · ความอ่านง่าย

| ตรวจ | ผล | หลักฐาน |
|---|---|---|
| `gofmt -l .` | ผ่าน (ไม่มีไฟล์ค้าง) | รันจริง [V] |
| `go vet ./...` | ผ่าน | รันจริง [V] |
| `go test -race -count=1 ./...` | **ผ่านทุกแพ็กเกจ** (exit 0) | รันจริง [V] |
| Coverage รวม | **68.3%** | `go tool cover -func` [V] |
| `go build ./cmd/boi` | ผ่าน | รันจริง [V] |
| staticcheck | NOT VERIFIED ในเครื่อง | เครื่องมือไม่มี |
| `panic(` ในโค้ดจริง | 0 | `grep` [V] |
| TODO/FIXME ในโค้ดจริง | 0 | `grep` [V] |
| error ที่ถูกทิ้ง (`_ =`, `, _ :=`) | 78 จุด | `grep` [V] (ไม่ได้ไล่ทีละจุด) |

**Coverage รายแพ็กเกจที่ต่ำกว่าเกณฑ์ที่ควรเป็น** [V]

| แพ็กเกจ | Coverage | หมายเหตุ |
|---|---|---|
| `cmd/boi` | 0.0% | entry point |
| `internal/block` | 15.8% | |
| `internal/transport/cli` | 32.5% | ผิวสัมผัสหลักของผู้ใช้ |
| `internal/platform/update` | 41.6% | โค้ดที่แทนที่ binary ตัวเอง |
| `internal/transport/tui`, `tui/setup` | 54.3% | |
| `internal/runtime/llm` | 57.8% | Router/failover |
| `internal/equipment/tools/mcp` | 60.6% | |

แพ็กเกจแกนอยู่ในเกณฑ์ดี: `runtime/agent` 83.4%, `core` 80.1%, `app` 86.5%, `adapters` 86.5%, `dialect` 92.5%, `childenv` 100%

**Bug / logic error ที่ยืนยันจากโค้ด** — ดู M1, M2, M3, M6 ในหัวข้อ 8

---

## 5. Security Assessment — 74/100

**เกณฑ์:** secret exposure · ขอบเขตการรันคำสั่งและไฟล์ · supply chain · การอัปเดตตัวเอง · ข้อมูลที่เก็บลงดิสก์

**จุดแข็ง**
- ไม่พบ key ใน tree และใน git history ทุก branch (regex `sk-…`, `AKIA…`, `ghp_…`, `AIza…`, private key) [V]
- `.env` อยู่ใน `.gitignore`; ไฟล์ที่ track มีแค่ `.env.example` [V]
- Child process ทุกตัวได้ env ที่กรอง `PSC_*` ออกแล้ว (`internal/platform/childenv`, coverage 100%); MCP server ใช้ตัวกรองเดียวกัน (`equipment/tools/mcp/client.go:110`) [V]
- เทสต์ regression ของ A1–A8 (audit 2026-11-01) ยังอยู่ครบและผ่าน [V]
- self-update ตรวจ SHA-256, จำกัดขนาด archive, กัน path traversal (`platform/update/update.go:305`) [V]
- ไม่มี `InsecureSkipVerify` [V]
- tool call ที่กู้จาก markup ยังผ่าน Broker และการอนุมัติ (`TestRecoveredToolCallStillNeedsApproval`) [V]

**จุดอ่อน** — รายละเอียดในหัวข้อ 8: M1, M2, M5, M7, M8, L1, L2, L3, L7

**NOT VERIFIED**
- ช่องโหว่ของ dependency: `govulncheck` ไม่มีในเครื่อง และฐานข้อมูลช่องโหว่เข้าไม่ถึงจาก sandbox (รอบก่อนได้ HTTP 403)
- การ bypass deny-list ของ `process.run` ด้วยคำสั่งที่อำพราง: โค้ดระบุเองว่าไม่ใช่ขอบเขตความปลอดภัย (`process/sandbox.go:10-13`) ไม่ได้ทดสอบเจาะ
- พฤติกรรมบน Windows (PowerShell path ที่ `process/executor.go:177`)

---

## 6. Agent & CLI Assessment — 66/100

**เกณฑ์:** entry point ทำงานจริงไหม · lifecycle และ Tool · Provider/routing · config · session · ความพร้อม production

**CLI ที่รันจริงใน workspace แยก** [V]

| คำสั่ง | ผล | ข้อสังเกต |
|---|---|---|
| `boi --help` | แสดง 20 คำสั่ง | มีคำว่า Chimera/Phantom (L9) |
| `boi version` | `v2.7.1`, exit 0 | ตรงกับ tag ทั้งที่ master นำหน้าไป 14 commits |
| `boi doctor` | 3/9 ผ่าน, **exit 0** | สคริปต์แยกไม่ออกว่าพัง (M10) |
| `boi env` | แสดง Blocks และเหตุผลที่ Tool ไม่ถูกเสนอ | ชัดเจนดี |
| `boi ask "hello"` (ไม่มี Provider) | `Error: load providers`, exit 5 | ไม่บอกสาเหตุและวิธีแก้ (M3) |
| `boi nosuchcmd` | error, exit 1 | ถูกต้อง |

คำสั่งอ่านอย่างเดียว (`doctor`/`env`) สร้างโฟลเดอร์ `.boi` ใน workspace (L11) [V]

**Agent lifecycle และ Tool**
- Engine เป็น state machine มี limit (12 ขั้น / 8 tool calls / 2 นาที), มี Broker, Verifier, Recoverer; coverage 83.4% [V]
- Model zoo 11 แบบรันผ่าน Service จริง: 9 แบบทำงานเสร็จถูกต้อง, 1 แบบ (`corrupt-token`) เสร็จแต่ตอบผิด (H1), 1 แบบ (`down`) ถูกกันออกอย่างถูกต้อง [V]

**Provider และ routing**
- Router มี retry + backoff + failover ตามชนิด error (`runtime/llm/router.go:92-135`) แต่ **ไม่มี circuit breaker และ replica affinity** (อยู่ในแผน P3.7) [V]
- Adapter: OpenAI-compatible, Anthropic, Google; dialect 3 แบบ; reasoning replay
- **NOT VERIFIED:** การทำงานกับ Provider จริงทุกราย (host ถูกบล็อกจาก sandbox)

**Config**
- Provider อ่านจาก env `PSC_1..PSC_n` (`service/provider/factory/factory.go:47-63`); `.env` เขียนด้วยสิทธิ์ 0600 (`tui/setup/env.go:83,95`) [V]
- ไม่ตรวจ scheme ของ base URL (M8)

**Session**
- บทสนทนาอยู่ในหน่วยความจำเท่านั้น (`runtime/agent/service.go:554`); ปิดโปรแกรมแล้วหาย; ไม่มี resume/fork (M9, รอการตัดสิน F9) [V]
- มี checkpoint + manifest + `events.jsonl` ต่อ task แต่ **ยังไม่มีโค้ดใดอ่านกลับมาใช้**

---

## 7. Git & CI/CD Status — 72/100

**เกณฑ์:** สถานะ repo · ความเข้มของ CI · ความเข้มของ release · ความสะอาดของ branch/PR

| รายการ | สถานะ | หลักฐาน |
|---|---|---|
| Branch ปัจจุบัน | `ccr-ee8da262-0ibcmg`, สะอาด | [V] |
| Merge conflict | ไม่มี | PR #31 `mergeable_state: clean` [V] |
| **PR #31** | **OPEN, ยังไม่ merge, ค้าง 17 วัน** (เปิด 2026-11-03), CI ผ่านครบ 10 job บน `8aace34` | GitHub API [V] |
| ผลของ PR #31 ค้าง | `HANDOFF.md` บน master ยังบอกว่า "Phase 0 กำลังทำ" และไม่มีรายการงานที่ merge แล้ว | `git show origin/master:HANDOFF.md` เทียบ branch [V] |
| Issue ที่ค้าง | NOT VERIFIED | ไม่ได้เรียกดู issue list |
| Branch ค้างบน remote | 18 อัน (`agent-suite-v1.5` … `v2.6`) | `git branch -r` [V] |

**CI (`.github/workflows/ci.yml`)** — เข้ม: gofmt, vet, race + coverage gate 55%, staticcheck (pin เวอร์ชัน), acceptance บน binary จริง, cross-compile 4 เป้า [V]

**Release (`.github/workflows/release.yml`)** — อ่อนกว่า CI: รันแค่ `go test`, `go vet`, `go build` (M4) [V]

**ข้อสังเกตอื่น:** L4 (pin action ด้วย tag, รันซ้ำ 2 รอบ, ไม่มี `permissions:`), L13 (goreleaser ออกเฉพาะ windows/linux), M5 (ไม่มี vulnerability scan)

**Release readiness:** master นำหน้า `v2.7.1` อยู่ 14 commits; `CHANGELOG.md` มีหัวข้อ "Unreleased" ครบ; ยังไม่ได้กำหนดเลขเวอร์ชันถัดไป

---

## 8. Critical Findings

**CRITICAL: ไม่พบ**

### HIGH

**H1 — โมเดลที่ token เพี้ยนตอบผิด แต่งานถูกรายงานว่าเสร็จ**
- **ที่:** `internal/acceptance/modelzoo_test.go:61` (baseline ที่บันทึกไว้), `internal/acceptance/modelzoo/modelzoo.go` (trait `CorruptToken`)
- **สาเหตุ:** ไม่มีขั้นตรวจว่าคำตอบสุดท้ายตรงกับหลักฐานจาก Tool; Verifier ตรวจแค่ผลของ Tool
- **ผลกระทบ:** ผู้ใช้ได้คำตอบผิดพร้อมสถานะ `completed` ขัด E6 ("ข้อจำกัดต้องบอกตรงๆ") นี่คือพฤติกรรมที่เคยเจอกับ maxplus จริง (`OBSERVATION_public`)
- **แก้:** P2.3 + ตัวตรวจ grounding ของคำตอบเทียบ observation (ดูแผน S2.3)
- **ลำดับ:** 1 · **หลักฐาน:** เทสต์ zoo ผ่านด้วย baseline `marker=false` สำหรับ trait นี้ [V]

**H2 — Provider จริงของเจ้าของ (maxplus) ยังถูกปิด Tool ทั้งหมด**
- **ที่:** `internal/core/qualification.go:148-170` (`ComposeEnvironment`)
- **สาเหตุ:** `trusted := completion ∧ tool_observation ∧ authority`; probe ตกข้อเดียวปิดทุก Tool; Recipe และ approval floor (D1) ยังไม่มีในโค้ด
- **ผลกระทบ:** หลักการที่อนุมัติแล้ว ("เติม ไม่ตัด") ยังไม่มีผลกับผู้ใช้จริง; โมเดลอ่อนยังใช้ Tool ไม่ได้
- **แก้:** P2.1 → P2.3 → P2.5 (ดูแผน S2.1–S2.2)
- **ลำดับ:** 2 · **หลักฐาน:** โค้ด + zoo baseline `tools=false` สำหรับ `corrupt-token` [V]

**H3 — โค้ดหลัง v2.7.1 ไม่เคยรันกับ Provider จริง**
- **ที่:** ทั้ง diff `v2.7.1..master` (+3,014 บรรทัด): dialect parser, reasoning replay, fallback, event log
- **สาเหตุ:** sandbox เข้า host ของ Provider ไม่ได้; P0.7 (record/replay) ยังรอผลรันจากเจ้าของ
- **ผลกระทบ:** พฤติกรรมจริงของ DSML/reasoning ของ DeepSeek อาจต่างจาก fixture (เคยเกิดแล้ว: fixture DSML ของ P0.6 ผิดรูป)
- **แก้:** รัน acceptance จริงตามแผน S3.2 ก่อน tag
- **ลำดับ:** 3 · **หลักฐาน:** `HANDOFF.md` (PR #31) และ PR #28–#30 ระบุเองว่าวัดบน endpoint จำลอง [V]

### MEDIUM

| ID | เรื่อง | ที่ | สาเหตุ → ผลกระทบ | แก้ |
|---|---|---|---|---|
| **M1** | `reasoning_content` อาจถูกส่งข้าม Provider ตอน failover | `runtime/agent/service.go:464, 753` | map `reasoning` ผูกกับ run ไม่ผูกกับ Provider → เมื่อ Router สลับ Provider กลางงาน ฟิลด์ของ Provider A ถูกส่งให้ B; gateway ที่เข้มงวดอาจตอบ 400; CHANGELOG อ้างว่า "ส่งเฉพาะ endpoint ที่สร้าง" ซึ่งไม่จริงในกรณีนี้ | เก็บชื่อ Provider คู่กับ reasoning แล้วส่งเฉพาะเมื่อ Provider ตรงกัน + เทสต์ zoo failover (F4) |
| **M2** | `events.jsonl` เก็บข้อความ error ของ endpoint | `runtime/agent/event.go:200-205`, `service.go:379` | `route.changed` ใส่ `err.Error()` (ตัด 200 ตัวอักษร) → ขัดกับ `ARCHITECTURE_IV.md` §5.4 "ไม่เก็บข้อความ error"; body ของ gateway อาจมีข้อมูลที่ไม่ควรเก็บ | เก็บแค่ route + error class + HTTP status; ขยาย `TestEventsNeverCarry…` ให้ครอบ route.changed |
| **M3** | ข้อความ error ของ `boi ask` ทิ้งสาเหตุจริง | `transport/cli/automation.go:37-45`, `agent.go:59` | `CommandError.Error()` คืน `Message` แล้วทิ้ง `Cause` → ผู้ใช้เห็นแค่ `Error: load providers` ขัด E1 | ต่อ cause และคำแนะนำ (`boi setup`) ในข้อความ |
| **M4** | Release ตรวจอ่อนกว่า CI | `.github/workflows/release.yml:17-19` | ไม่มี race, staticcheck, gofmt, coverage, acceptance → tag จาก commit ที่ CI แดงก็ออก binary ได้ | ให้ release เรียก job เดียวกับ CI |
| **M5** | ไม่มี vulnerability scan | `.github/` | ไม่มี govulncheck/dependabot → ช่องโหว่ของ dependency ไม่ถูกแจ้ง | เพิ่ม govulncheck ใน CI + dependabot |
| **M6** | Dialect parser อาจรัน Tool เมื่อโมเดลแค่ยกตัวอย่าง markup | `service/provider/dialect/dialect.go:66`, `adapters/openai.go:176` | กู้ทุก block ที่ถูกรูปและตรงชื่อ Tool แม้อยู่ใน code fence → เรียก Tool ที่ผู้ใช้ไม่ได้ขอ (ยังผ่านอนุมัติสำหรับงานเขียน/รันคำสั่ง แต่ read ผ่านอัตโนมัติ) | ข้าม block ที่อยู่ใน code fence; เปิดใช้ต่อ endpoint ตาม Recipe (`tool_protocol: dialect:<name>`) |
| **M7** | self-update เชื่อ checksum จากที่เดียวกับ binary | `platform/update/update.go:129-135` | ไม่มีลายเซ็น → ถ้า release ถูกแก้ ทั้งไฟล์และ checksum ถูกแก้พร้อมกันได้ | เซ็น `checksums.txt` (cosign/minisign) แล้วตรวจด้วย public key ที่ฝังใน binary |
| **M8** | ไม่ตรวจ scheme ของ base URL | `service/provider/factory/factory.go:71-88` | `http://` ไปยัง host ที่ไม่ใช่เครื่องตัวเองส่ง API key แบบไม่เข้ารหัส | ปฏิเสธหรือเตือน `http://` เมื่อ endpoint ไม่ใช่ `local` |
| **M9** | Session ไม่ถูกเก็บ ไม่มี resume | `runtime/agent/service.go:554` | บทสนทนาอยู่ในหน่วยความจำ → งานยาวที่พังต้องเริ่มใหม่ (E4 ยังไม่มี) | รอ F9 แล้วทำ P3.1–P3.5 |
| **M10** | `boi doctor` exit 0 แม้ตรวจไม่ผ่าน | `transport/cli/doctor.go:30-135` | ไม่คืน error → ใช้ใน script/CI ไม่ได้ | exit ≠ 0 เมื่อมีข้อที่จำเป็นไม่ผ่าน |
| **M11** | PR #31 ค้าง ทำให้ handoff บน master ล้าสมัย | PR #31 | ยังไม่ merge 17 วัน → เอเจนต์ที่เริ่มจาก master อ่านสถานะผิด | merge PR #31 |

### LOW

| ID | เรื่อง | ที่ |
|---|---|---|
| L1 | `extractTarGz` ใช้ `header.Mode` ตรงๆ และไม่เช็ก error ของ `MkdirAll`/`Chmod` | `platform/update/update.go:310-327` |
| L2 | uninstall เขียน batch ชื่อคงที่ใน temp dir ด้วยสิทธิ์ 0644 | `transport/cli/uninstall.go:67-76` |
| L3 | `process.run` เป็น deny-list ไม่ใช่ sandbox ระดับ OS (ยอมรับแล้วใน D7) | `equipment/tools/process/sandbox.go:10-36` |
| L4 | Actions pin ด้วย tag; CI รัน 2 รอบ (`push`+`pull_request`); ไม่มี `permissions:` | `.github/workflows/ci.yml:3` |
| L5 | งานของ SubAgent ไม่ถูกบันทึก event | `runtime/agent/delegation.go` |
| L6 | ไฟล์ใหญ่ รวมหลายหน้าที่ | `service.go` (841), `engine.go` (793) |
| L7 | ไฟล์ memory เขียนด้วยสิทธิ์ 0644 | `equipment/memory/store.go:63` |
| L8 | MCP เปิด process ใหม่ทุก request | `equipment/tools/mcp/client.go:109` |
| L9 | ชื่อเก่า Chimera/Phantom ในผลิตภัณฑ์ | `transport/cli/root.go:17`, `version.go:13`, `memory.go:16` |
| L10 | Branch เก่า 18 อันบน remote | `git branch -r` |
| L11 | คำสั่งอ่านอย่างเดียวสร้าง `.boi` | สังเกตจากการรัน `doctor`/`env` |
| L12 | Coverage ต่ำใน `cli` 32.5%, `update` 41.6%, `llm` 57.8%, `cmd/boi` 0% | ผล coverage |
| L13 | goreleaser ไม่ออก binary ของ darwin/android | `.goreleaser.yml:7-8` |
| L14 | `golang.org/x/text v0.3.8` (indirect) เก่า; ผลต่อช่องโหว่ NOT VERIFIED | `go.mod` |

---

## 9. Technical Debt

1. **ช่องว่างระหว่างสถาปัตยกรรมที่ออกแบบกับที่สร้าง** — 37 ขั้นที่เหลือ; `Recipe`, approval floor, model switch, resume, extension, sandbox ระดับ OS ยังไม่มี
2. **ข้อมูลที่เก็บแต่ยังไม่มีใครใช้** — `events.jsonl`, checkpoint, `parent_agent_id` ยังไม่มีผู้อ่าน
3. **ไฟล์รวมศูนย์** — `service.go`, `engine.go`, `openai.go`, `tui/app.go`
4. **เทสต์ที่ผิวสัมผัสผู้ใช้บาง** — CLI 32.5%, TUI 54.3%, entry point 0%
5. **ชื่อและเอกสารค้างรุ่น** — Chimera/Phantom; `HANDOFF.md` บน master; `boi version` ยังเป็น 2.7.1
6. **Router พื้นฐาน** — ไม่มี circuit breaker, affinity, per-endpoint Recipe (D3)
7. **Release pipeline** — ไม่เท่ากับ CI, ไม่เซ็น, ไม่ scan
8. **หนี้ด้านหลักฐาน** — ไม่มีผลกับ Provider จริง, Windows, Android หลัง v2.7.1

---

## 10. Recommended Action Plan

แผนเต็มพร้อมไฟล์ที่แตะ เทสต์ และเกณฑ์ผ่านอยู่ใน [`STABILIZATION_PLAN.md`](STABILIZATION_PLAN.md) สรุปสั้น:

| ชั้น | เป้าหมาย | งาน |
|---|---|---|
| **S1 แก้ทันที** | ปิดข้อบกพร่องของงานที่ merge แล้ว และรู release | M11, M1, M2, M3, M4, M5, M8, M10 |
| **S2 ใช้งานได้จริง** | โมเดลอ่อนได้ Tool กลับอย่างปลอดภัย และคำตอบผิดถูกจับได้ | H2 (P2.1→P2.3→P2.5), H1 (grounding), M6, M9 ส่วนที่จำเป็น |
| **S3 พิสูจน์ความเสถียร** | มีหลักฐานจริงก่อนเรียกว่าเสถียร | H3 (acceptance จริง), P0.7, soak test, coverage ต่อ Block, M7 |

---

## 11. Production Readiness — 55/100

**เกณฑ์:** หลักฐานการใช้งานจริง · การกู้คืน · ความปลอดภัยของ pipeline · ความชัดเจนของ error · การสังเกตการณ์

| ด้าน | สถานะ | เหตุผล |
|---|---|---|
| Build / เทสต์ | พร้อม | ผ่านทั้งหมดในรอบนี้ [V] |
| ผู้ใช้กลุ่มเดิม (โมเดลที่เรียก Tool แบบมาตรฐาน) | พร้อมแบบมีเงื่อนไข | พฤติกรรมเดิมไม่เปลี่ยน; ต้องแก้ M1 ก่อนถ้าใช้หลาย Provider |
| โมเดลอ่อน / gateway | **ไม่พร้อม** | H1, H2 |
| หลักฐานกับ Provider จริง | **ไม่มี** | H3 |
| Windows / Android | NOT VERIFIED | CI แค่ compile |
| กู้คืนงานที่พัง | ไม่มี | M9 |
| Release pipeline | ต้องปรับ | M4, M5, M7 |
| การสังเกตการณ์ | พื้นฐานมีแล้ว | event log มี แต่ยังไม่มีผู้อ่าน |

---

## 12. Final Verdict

**Repository อยู่ในสภาพดีสำหรับโครงการที่อยู่ระหว่างรื้อสถาปัตยกรรม** — พื้นฐานวิศวกรรมแข็งแรง ไม่มีปัญหาระดับ CRITICAL และความเสี่ยงที่เหลือส่วนใหญ่ถูกระบุไว้ในแผนอยู่แล้ว แต่ **ยังไม่ถึงเป้าหมายของตัวเอง**: โมเดลอ่อนยังใช้งานได้ไม่จริง และงานใหม่ยังไม่มีหลักฐานจากการใช้งานจริง

### สิ่งที่ทำได้ดี
- เทสต์ผ่านครบพร้อม race, coverage 68.3%, แพ็กเกจแกน 80%+
- ไม่มี secret รั่ว; child process ถูกกรอง env; finding เก่าปิดครบพร้อมเทสต์
- ขอบเขต Block ถูกบังคับด้วยเทสต์; เอกสารสถาปัตยกรรมและแผนครบ
- ความซื่อสัตย์ของสถานะ: สิ่งที่ได้รับความช่วยเหลือถูกระบุว่า assisted และมีเทสต์บังคับ
- Model zoo ทำให้ทดสอบโมเดลไม่สมบูรณ์ได้ใน CI

### สิ่งที่มีปัญหา
- คำตอบผิดจาก token เพี้ยนไม่ถูกจับ (H1)
- maxplus ยังไม่ได้ Tool (H2); `Recipe` ยังไม่มี
- ไม่มีหลักฐานกับ Provider จริงหลัง v2.7.1 (H3)
- ข้อบกพร่อง 2 ข้อในงานที่เพิ่ง merge (M1, M2)
- ข้อความ error ของ CLI ไม่บอกสาเหตุ (M3); release อ่อนกว่า CI (M4)

### สิ่งที่ควรแก้ทันที
1. merge PR #31 (M11)
2. M1 — ผูก reasoning กับ Provider
3. M2 — เอาข้อความ error ออกจาก event
4. M3 — ให้ `boi ask` บอกสาเหตุและวิธีแก้
5. M4 + M5 — ยกระดับ release ให้เท่า CI และเพิ่ม vulnerability scan

### สิ่งที่ควรพัฒนาใน Version ถัดไป
- Recipe + approval floor (P2.1–P2.5) ให้โมเดลอ่อนได้ Tool กลับ
- ตัวตรวจ grounding ของคำตอบ (H1)
- Resume งานที่พัง (P3.1–P3.5) หลังตัดสิน F9
- เซ็น release (M7), sandbox ระดับ OS (D7/P4.6)
- แยก adapter สามชั้น (P1.1) และลดขนาด `service.go`

### Repository พร้อม Release หรือไม่
- **Patch/minor (เช่น v2.8.0) สำหรับผู้ใช้กลุ่มเดิม:** **พร้อมแบบมีเงื่อนไข** — หลังแก้ M1, M2, M3, M4 และรัน acceptance กับ Provider จริงอย่างน้อย 1 ราย
- **v3.0.0 / "เสถียรสำหรับทุกโมเดล":** **ยังไม่พร้อม** — ต้องปิด H1, H2, H3 ก่อน

---

## ภาคผนวก: ขอบเขตที่ไม่ได้ตรวจ (NOT VERIFIED)

| เรื่อง | เหตุผล |
|---|---|
| staticcheck, deadcode, govulncheck ในเครื่อง | เครื่องมือไม่มีใน container รอบนี้ และกติกาห้ามติดตั้ง |
| Provider จริงทุกราย | host ถูกบล็อกจาก sandbox |
| Windows, macOS, Android ขณะรันจริง | มีแค่ compile check ใน CI |
| TUI แบบโต้ตอบ | ไม่ได้รันในเทอร์มินัลจริง |
| การเจาะ deny-list ของ `process.run` | ไม่ได้ทดสอบเจาะ |
| GitHub Issues, branch protection, secret ของ Actions | ไม่ได้เรียกดู |
| 78 จุดที่ทิ้ง error | นับแล้ว ไม่ได้ไล่ทีละจุด |
| ผล CI ของ `master` ล่าสุดโดยตรง | อ้างจากผลของ PR #31 (`8aace34`) ซึ่ง base คือ master เดียวกัน |
