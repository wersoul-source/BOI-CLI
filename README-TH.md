# BOI Agent Suite

[English](README.md) | [ภาษาไทย](README-TH.md)

BOI คือ Agent สำหรับทำงานใน workspace ผ่าน terminal ที่มีขอบเขตชัดเจน มันอ่านโปรเจกต์
เสนอการกระทำ ขออนุญาตก่อนเปลี่ยนแปลงอะไร ตรวจสอบสิ่งที่ทำแล้ว และทิ้ง manifest ของผลงานไว้

v1.5 วางโครงใหม่ทั้งหมดบนหลักเดียว: **แกนตายตัว + Block ที่ถอดประกอบได้**
แกนของ Agent คงเดิมเสมอ ส่วนความสามารถใหม่เสียบเข้ามาได้โดยไม่ต้องแก้แกน
หลังจากนั้นระบบคุยกับแต่ละโมเดลแบบ native เมื่อโมเดลพิสูจน์ได้ว่าใช้เป็น อ่านและค้นไฟล์ได้ในจำนวนรอบที่น้อยลง และประกอบ Agent จาก Block pack ต่อโปรเจกต์ได้ (Tool ในตัว, MCP server, SubAgent แบบเสริม) และ `boi env` รายงานตามสิ่งที่โหลดจริง

มี release สำหรับ Linux และ Windows (amd64, arm64) ส่วน macOS และ Android คอมไพล์ได้แต่ยังไม่ใช่เป้าหมาย release

## เริ่มเร็ว

**1. ติดตั้ง** ไม่ต้องมีอย่างอื่น เปิด terminal ใหม่หลังติดตั้งเพื่อให้เรียก `boi` ได้

Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/wersoul-source/BOI-CLI/master/scripts/install.sh | bash
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/wersoul-source/BOI-CLI/master/scripts/install.ps1 | iex
```

ยังไม่มี release สำหรับเครื่องคุณ? ดู [build จากซอร์ส](#build-จากซอร์ส)

**2. เตรียมให้พร้อม** ในโฟลเดอร์ของโปรเจกต์ที่อยากให้ Agent ทำงาน:

```text
boi quickstart
```

คำสั่งนี้หาโมเดลให้และตรวจว่าโมเดลทำอะไรได้บ้าง:

- โมเดลที่รันในเครื่อง (Ollama, LM Studio, llama.cpp) จะถูกตรวจเจอและเสนอเป็นรายชื่อ ไม่ต้องพิมพ์อะไร ไม่ต้องมี API key
- หรือเลือก provider แบบเสียเงิน (OpenAI, Anthropic, Cloudflare ฯลฯ) แล้วใส่ API key
- ค่าที่ตั้งจะเก็บใน `.env` ของโปรเจกต์ และไม่ถูกส่งเข้า Git

ยังไม่มีโมเดล? วิธีเริ่มฟรีที่เร็วที่สุดคือ [Ollama](https://ollama.com): ติดตั้ง รัน `ollama pull qwen2.5-coder:7b` แล้วสั่ง `boi quickstart`

**3. ใช้งาน**

```text
boi                                 คุยและทำงานกับโปรเจกต์นี้
boi ask "main.go ทำอะไร?"             ถามหนึ่งคำถาม อ่านอย่างเดียว
```

ทุกการแก้ไฟล์และทุกคำสั่งจะถามคุณก่อนเสมอ พร้อมแสดงสิ่งที่จะทำ
ติดปัญหา? `boi doctor` ตรวจการตั้งค่า และ `boi env` แสดงว่า Agent ใช้อะไรได้บ้าง รายละเอียดอยู่ใน [GET_STARTED.md](GET_STARTED.md)

## สถาปัตยกรรม

```text
            ┌──────────────── แกนตายตัว ─────────────────┐
            │  Core      ตัวตน, การคัดกรอง, Persona     │
            │  Runtime   engine, Broker, การอนุมัติ, LLM │
            └──────────────────────┬─────────────────────┘
                                   │ ports (block/port, llm.Provider, ...)
      ┌──────────────┬─────────────┼──────────────┬──────────────┐
  Equipment       Service      Agent Folder     SubAgent       (ของคุณ)
  tools, skills,  providers,   bin / output     agent.delegate (เสริม)
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

ชุดสำเร็จรูป: ดูด้วย `boi env presets` แล้วเลือก `boi env use minimal` (อ่านอย่างเดียว) หรือ `boi env use coding` (อ่าน เขียน รันคำสั่ง โดยต้องขออนุญาต) แล้วดูผลด้วย `boi env`

pack ในตัว: `workspace-tools` (list, read, search, write), `workspace-read` (list, read, search), `process-tools` (`process.run`) และ `subagent` (`agent.delegate`: ส่งงานย่อยให้ผู้ช่วยที่ใช้ได้เฉพาะ Tool ที่อนุญาต ดู [ADR-003](docs/architecture/ADR-003-subagent.md))
รายการที่มี `skills` จะเสียบโฟลเดอร์ Skill (ไฟล์ `*.skill.md` อยู่ใต้ `.boi`) เช่น `{"id": "team-skills", "required": false, "skills": {"dir": "skills/team"}}` Skill เป็นแค่ข้อความ ไม่เพิ่ม Tool และไม่ต้อง trust โมเดลเป็นคนตัดสินว่าจะโหลดหรือไม่ และ Skill ทำงานได้ทั้งมีและไม่มี Tool ที่มันอ้างถึง ดู [ADR-004](docs/architecture/ADR-004-skill-packs.md)
รายการที่มี `mcp` จะเสียบ tool ของ MCP server ในเครื่องเข้ามา ทุกตัวต้องขออนุญาตก่อนใช้ และไม่ต้อง build BOI ใหม่
แต่ MCP คือคำสั่งที่รันบนเครื่องคุณ จึงเริ่มทำงานก็ต่อเมื่อคุณตรวจไฟล์แล้วสั่ง `boi env trust` ถ้าไฟล์ถูกแก้ภายหลัง trust จะถูกถอนอัตโนมัติ

```json
{"id": "github", "required": false, "mcp": {"command": "github-mcp-server", "args": ["stdio"]}}
```
รายละเอียดที่ [ADR-002](docs/architecture/ADR-002-block-packs.md)

### สองทางในการเสียบความสามารถใหม่

| ทาง | ใช้เมื่อ | วิธี |
|---|---|---|
| ตอนคอมไพล์ | Tool ที่เขียนด้วย Go | ทำตาม `port.Tool` แล้วเพิ่มใน pack ของ `app.BuiltinPacks` และใน `app.DefaultCapabilityIndexes` โดยไม่ต้องแก้ Broker |
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

## Build จากซอร์ส

ต้องมี Go 1.24.2 ขึ้นไป

```bash
git clone https://github.com/wersoul-source/BOI-CLI.git
cd BOI-CLI
go build -trimpath -o boi ./cmd/boi      # Windows: go build -o boi.exe ./cmd/boi
sudo install boi /usr/local/bin/         # หรือใช้ ./boi ก็ได้
```

จากนั้นสั่ง `boi quickstart` ในโปรเจกต์ ขั้นตอนที่คำสั่งนั้นทำให้ ถ้าอยากทำทีละขั้น:

```bash
boi init                        # สร้าง state ใน .boi (ไม่ลบของเดิม)
boi setup                       # เลือก Provider (wizard)
boi provider qualify <name>     # ทดสอบพฤติกรรมจริง (โมเดลแบบเสียเงินจะเสีย token)
boi doctor                      # ตรวจสุขภาพ
boi                             # เปิด TUI
```

เปิด TUI ครั้งแรก ระบบจะถามชื่อ Agent ของคุณ Core Persona ยังเป็น `boi` เสมอ ชื่อนี้เป็นของ Agent ตัวนั้นเท่านั้น

Provider ที่ยังไม่ผ่าน `boi provider qualify` จะไม่ถูกใช้งาน และ BOI จะไม่ตอบแบบจำลองแทน
ผล qualify ยังตัดสินว่าจะใช้ native tool calling หรือไม่ (`native_tool_schema`)
ผล qualify ของ v1.5 ใช้กับ v1.6 ขึ้นไปไม่ได้ และผลทุกชุดหมดอายุใน 30 วัน ถ้า `boi env` บอกว่าไม่มีโมเดลที่ผ่าน ให้ qualify ใหม่

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
| `boi env` | Agent ประกอบจาก Block อะไรบ้าง สถานะ active, loaded หรือ declared พร้อมเหตุผล |
| `boi env presets\|use\|trust` | ดูและใช้ preset, trust MCP server ใน environment ของ workspace |
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
- SubAgent เป็นตัวเสริม ไม่อยู่ใน preset ใด และยังไม่ผ่านด่าน benchmark
- MCP server รันเป็น process ในเครื่อง (stdio เท่านั้น) และเริ่มโดยไม่มีตัวแปร `PSC_*` นอกจากนั้นไม่ได้ถูกแยกสิทธิ์เพิ่ม
- BOI ต้องใช้เครือข่ายเพื่อคุยกับ Provider ไม่ได้ออกแบบมาให้ทำงาน offline
- ทดสอบครบวงจรแล้วเฉพาะ Linux แพลตฟอร์มอื่นตรวจแค่ว่าคอมไพล์ได้

## ร่วมพัฒนา

อ่าน [BLOCK_ARCHITECTURE.md](docs/architecture/BLOCK_ARCHITECTURE.md) ก่อนเพิ่มแพ็กเกจ
และ [CONTRIBUTING.md](CONTRIBUTING.md) สำหรับขั้นตอนการทำงาน
ประวัติและบันทึกส่งต่องาน: [HANDOFF.md](HANDOFF.md)

License: MIT
