# BOI Agent Suite v1.6

[English](README.md) | [ภาษาไทย](README-TH.md)

BOI คือ Agent สำหรับทำงานใน workspace ผ่าน terminal ที่มีขอบเขตชัดเจน มันอ่านโปรเจกต์
เสนอการกระทำ ขออนุญาตก่อนเปลี่ยนแปลงอะไร ตรวจสอบสิ่งที่ทำแล้ว และทิ้ง manifest ของผลงานไว้

v1.5 วางโครงใหม่ทั้งหมดบนหลักเดียว: **แกนตายตัว + Block ที่ถอดประกอบได้**
แกนของ Agent คงเดิมเสมอ ส่วนความสามารถใหม่เสียบเข้ามาได้โดยไม่ต้องแก้แกน
v1.6 ปรับวิธีคุยกับแต่ละโมเดล: ใช้ native tool calling เมื่อโมเดลพิสูจน์ได้ว่าใช้เป็น
เรียกได้หลาย Tool ต่อรอบ และคัดกรองโมเดลจากพฤติกรรมแทนการเทียบข้อความตรงตัว

Linux คือแพลตฟอร์มแรกที่รองรับ ส่วน Windows, macOS และ Android คอมไพล์ได้แต่ยังไม่ใช่เป้าหมาย release

## สถาปัตยกรรม

```text
            ┌──────────────── แกนตายตัว ─────────────────┐
            │  Core      ตัวตน, การคัดกรอง, Persona     │
            │  Runtime   engine, Broker, การอนุมัติ, LLM │
            └──────────────────────┬─────────────────────┘
                                   │ ports (block/port, llm.Provider, ...)
      ┌──────────────┬─────────────┼──────────────┬──────────────┐
  Equipment       Service      Agent Folder     SubAgent       (ของคุณ)
  tools, skills,  providers,   bin / output     ยังปิดอยู่
  memory, MCP     config       manifests
```

- **แกน** (`internal/core`, `internal/runtime`): ตายตัว กำหนดสัญญาที่ทุก Block ต้องทำตาม และไม่ import Block ใดเลย
- **Block** (`internal/equipment`, `internal/service`, `internal/agentfolder`, `internal/subagent`):
  ถอดประกอบได้ พึ่งได้แค่แกน ห้ามพึ่ง Block อื่น
- **จุดประกอบ** (`internal/app`): ที่เดียวที่ต่อ Block เข้ากับแกน ทั้ง TUI และ CLI สร้าง Agent ผ่าน `app.BuildAgent`

กฎเหล่านี้บังคับด้วย `internal/architecture/deps_test.go` ถ้ามีการแก้ที่ผิดกฎ `go test ./...` จะไม่ผ่าน
โครงสร้างเต็มอยู่ที่ [docs/architecture/BLOCK_ARCHITECTURE.md](docs/architecture/BLOCK_ARCHITECTURE.md)

### เลือก Block ให้ Agent

ไฟล์ `.boi/environment.json` บอกว่า Agent ของโปรเจกต์นี้ใช้ Block pack อะไร และตัวไหนจำเป็น
ถ้าไม่มีไฟล์ ระบบใช้ pack ในตัวทั้งหมด ถ้า pack ที่จำเป็นขาดหรือใช้ไม่ได้ ระบบจะหยุดก่อนเริ่มงานพร้อมบอกเหตุผล

```json
{"schema_version": 1, "name": "no-shell", "blocks": [{"id": "workspace-tools", "required": true}]}
```

pack ในตัว: `workspace-tools` (list, read, search, write) และ `process-tools` (`process.run`)
รายละเอียดที่ [ADR-002](docs/architecture/ADR-002-block-packs.md)

### สองทางในการเสียบความสามารถใหม่

| ทาง | ใช้เมื่อ | วิธี |
|---|---|---|
| ตอนคอมไพล์ | Tool ที่เขียนด้วย Go | ทำตาม `port.Tool` แล้วเพิ่มใน `app.BuiltinTools` และใน capability index โดยไม่ต้องแก้ Broker |
| ตอนรัน | อะไรก็ได้ที่อยู่นอก binary | เปิดเป็น MCP server แล้ว `equipment/tools/mcp` จะแปลงแต่ละ tool เป็น Tool ที่ต้องขออนุมัติ |

## งานหนึ่งงานทำงานอย่างไร

```text
Observe → Decide → Authorize → Act → Verify → Recover
```

1. Core เลือก Tool ไม่เกิน 15 และ Skill ไม่เกิน 15 สำหรับงานนั้น
2. Model เสนอการเรียก Tool ได้สูงสุด 5 ครั้งต่อรอบ ผ่าน native tool call ถ้า Provider ผ่านการคัดกรอง ไม่อย่างนั้นใช้ `<boi-action>`
   แต่กำหนดระดับความเสี่ยงหรือการอนุมัติเองไม่ได้ Broker เป็นคนกำหนดจาก spec ของ Tool
   ชุดที่เป็นการอ่านอัตโนมัติล้วนจะรันพร้อมกัน นอกนั้นรันตามลำดับ
3. การอ่านทำงานอัตโนมัติ การเขียน การรัน process และการเรียก MCP ต้องให้คุณอนุมัติตรงตัวใน TUI ส่วนโหมดไม่โต้ตอบจะปฏิเสธเสมอ
4. ผลจาก Tool ที่ยาวหรือเก่าจะถูกย่อใน context ของโมเดล แต่ต้นฉบับเต็มเก็บไว้ใน store ของงานนั้น
   โมเดลขออ่านส่วนใดก็ได้แบบตรงต้นฉบับด้วย `context.expand(ref, lines)`
5. Tool ตรวจผลของตัวเอง เช่น อ่านไฟล์ที่เพิ่งเขียนกลับมาเทียบ ก่อนจะนับว่าขั้นนั้นสำเร็จ
6. งานที่เสร็จจะอยู่ใน `agent-folder/output/<task-id>/` พร้อม manifest ส่วนงานที่ล้มเหลวหรือถูกยกเลิกอยู่ใน `agent-folder/bin/<task-id>/`

## เริ่มต้นใช้งาน (Linux)

ต้องมี Go 1.24.2 ขึ้นไป และ API key ของ Provider ที่รองรับ

```bash
git clone https://github.com/wersoul-source/BOI-CLI.git
cd BOI-CLI
go build -trimpath -o boi ./cmd/boi
sudo install boi /usr/local/bin/      # หรือใช้ ./boi ก็ได้
```

ในโปรเจกต์ที่ต้องการให้ Agent ทำงาน:

```bash
boi init                        # สร้าง state ใน .boi (ไม่ลบของเดิม)
boi registry init               # สร้าง index ของ Tool/Skill
boi setup                       # เลือก Provider และใส่ API key
boi provider qualify <name>     # ทดสอบพฤติกรรมจริง (เสีย token)
boi doctor                      # ตรวจสุขภาพ
boi                             # เปิด TUI
```

เปิด TUI ครั้งแรก ระบบจะถามชื่อ Agent ของคุณ Core Persona ยังเป็น `boi` เสมอ ชื่อนี้เป็นของ Agent ตัวนั้นเท่านั้น

Provider ที่ยังไม่ผ่าน `boi provider qualify` จะไม่ถูกใช้งาน และ BOI จะไม่ตอบแบบจำลองแทน
ผล qualify ยังตัดสินว่าจะใช้ native tool calling หรือไม่ (`native_tool_schema`)
ผล qualify ของ v1.5 ใช้กับ v1.6 ไม่ได้ ต้อง qualify ใหม่หลังอัปเกรด

### งานแรก

```text
สร้างไฟล์ hello-boi.md มีหัวเรื่อง คำอธิบายสั้นๆ ของ repo นี้
และขั้นตอนถัดไปที่มีประโยชน์ 3 ข้อ อ่านโปรเจกต์ก่อนแล้วรายงาน path
```

เมื่อ Agent เสนอการเขียน ช่องพิมพ์จะเปลี่ยนเป็นแผงอนุมัติ กด `A` เพื่ออนุมัติการเขียนนั้นครั้งเดียว
`R` เพื่อปฏิเสธ `Esc` เพื่อยกเลิก ส่วน `Enter` ไม่มีวันอนุมัติ

## ใช้งานแบบไม่โต้ตอบ

```bash
boi ask "อธิบาย repo นี้"
cat task.txt | boi ask --json --idempotency-key task-001
```

`--json` เขียน object เดียวที่มีเวอร์ชันออก stdout ส่วน diagnostics ออก stderr
โหมดนี้อ่านอย่างเดียว การเรียกที่ต้องอนุมัติจะถูกปฏิเสธทันทีโดยไม่รอ
Exit code: `0` สำเร็จ, `1` ภายใน, `2` input ผิด, `3` ถูกปฏิเสธ, `4` ยกเลิก, `5` ไม่พร้อม, `6` ตรวจสอบไม่ผ่าน
ดู [Automation contract](docs/operations/AUTOMATION_CONTRACT.md)

## คำสั่ง

| คำสั่ง | ใช้ทำอะไร |
|---|---|
| `boi` | เปิด TUI |
| `boi ask` | รัน Agent แบบไม่โต้ตอบ |
| `boi init` / `boi setup` | เตรียม workspace / ตั้งค่า Provider |
| `boi provider list\|switch\|qualify` | จัดการและทดสอบ Provider |
| `boi registry init\|list\|add` | จัดการ index ของ Tool และ Skill |
| `boi doctor` | ตรวจสุขภาพในเครื่อง |
| `boi skill` / `boi memory` | จัดการ Skill และ memory |
| `boi config` / `boi model` | ดูหรือเปลี่ยนการตั้งค่า |
| `boi version` / `boi upgrade` | ดูเวอร์ชัน / อัปเกรดแบบตรวจ checksum |

### ปุ่มใน TUI

| ปุ่ม | การทำงาน |
|---|---|
| `Enter` | ส่งข้อความ ไม่มีวันอนุมัติ Tool |
| `Ctrl+N` | ขึ้นบรรทัดใหม่ |
| `Tab` | เติม slash command |
| `Esc` / `Ctrl+C` | ยกเลิกงานที่ทำอยู่ ถ้าว่างคือออก |
| `Ctrl+Q` | ออก |
| `/ls [path]`, `/read <path>` | ดูไฟล์ใน workspace |
| `/workspace`, `/providers`, `/persona` | ดู root, สถานะ Provider, ตัวตน |

## โครงสร้าง workspace

```text
your-project/
├── .boi/
│   ├── agent.yaml             ชื่อ Agent
│   ├── config.yaml
│   ├── provider-profiles/     ผลการทดสอบ Provider
│   ├── registry/              tools.json, skills.json (active สูงสุด 15/15)
│   ├── skills/
│   └── memory/
└── agent-folder/
    ├── bin/                   draft, log, งานที่ล้มเหลวหรือยกเลิก
    └── output/                ผลงานและ manifest
```

### ใช้ provider ชุดเดียวกับทุกโปรเจกต์

ใส่ค่า provider ที่ใช้ร่วมกันไว้ใน `~/.config/boi/.env` (ใช้ key `PSC_*` แบบเดียวกับ `.env` ในโปรเจกต์)
ถ้าตั้งค่าซ้ำกัน ค่าใน `.env` ของโปรเจกต์จะชนะ และค่าที่ export ใน shell จะชนะทั้งคู่
ผล qualify จะถูกเก็บไว้ที่ `~/.config/boi/provider-profiles` จึง qualify ครั้งเดียวแล้วใช้ได้ทุกโปรเจกต์
ค่าใน `.env` ใส่เครื่องหมายคำพูดหรือขึ้นต้นด้วย `export` ได้

endpoint ใดก็ได้ที่ใช้รูปแบบเดียวกับ OpenAI ใช้กับ BOI ได้ โดยตั้ง `PSC_1_NAME=openai` และ `PSC_1_BASE_URL`
ที่ทดสอบผ่านแล้ว: Cloudflare Workers AI (`scripts/acceptance/cloudflare_smoke.sh`)

อย่า commit `.env` หรือ API key `boi setup` จะเพิ่ม Git exclude ในเครื่อง สำรองไฟล์เดิม และเขียนไฟล์แบบสิทธิ์ส่วนตัว

## ตรวจบน Linux

```bash
make smoke
```

รัน vet, race test, กฎสถาปัตยกรรม และการจำลอง 9 สถานการณ์ด้วย binary จริงกับ Provider ปลอมในเครื่อง
ถ้าต้องการทดสอบกับ Provider จริงด้วย:

```bash
PSC_1_NAME=openai PSC_1_API_KEY=... PSC_1_MODEL=... make smoke
```

CI รันด่าน Linux ชุดเดียวกัน บวก staticcheck และเกณฑ์ coverage ขั้นต่ำ
และตรวจว่าคอมไพล์ได้บน linux/arm64, windows, darwin, android

วัดอัตราสำเร็จ จำนวนครั้งที่เรียกโมเดล token และเวลา ด้วยชุดงานคงที่ และเทียบกับเวอร์ชันก่อน:

```bash
BASE_REF=<commit-หรือ-tag> bash scripts/acceptance/benchmark.sh
```

## ความปลอดภัยและข้อจำกัด

- Workspace sandbox บังคับขอบเขต path รวมถึง symlink แต่ไม่ใช่การแยกระดับ OS หรือ container
  ถ้าเป็นโปรเจกต์ที่ไม่คุ้นเคยให้รันใน VM ที่แยกไว้
- Deny-list ของคำสั่งเป็นแค่ด่านกรองเบื้องต้น ไม่ใช่การแยก process ด่านจริงคือการอนุมัติของ Broker และขอบเขต path
- SubAgent ยังปิดอยู่จนกว่าจะผ่านด่านประเมิน
- ลงทะเบียน MCP Tool ได้แล้ว แต่การค้นหา MCP อัตโนมัติยังไม่ได้ต่อเข้าเส้นทางหลัก
- BOI ต้องใช้เครือข่ายเพื่อคุยกับ Provider ไม่ได้ออกแบบมาให้ทำงาน offline
- ทดสอบครบวงจรแล้วเฉพาะ Linux แพลตฟอร์มอื่นตรวจแค่ว่าคอมไพล์ได้

## ร่วมพัฒนา

อ่าน [BLOCK_ARCHITECTURE.md](docs/architecture/BLOCK_ARCHITECTURE.md) ก่อนเพิ่มแพ็กเกจ
และ [CONTRIBUTING.md](CONTRIBUTING.md) สำหรับขั้นตอนการทำงาน
ประวัติและบันทึกส่งต่องาน: [HANDOFF.md](HANDOFF.md)

License: MIT
