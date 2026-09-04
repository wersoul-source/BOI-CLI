# BOI CLI

[English](README.md) | ภาษาไทย

BOI CLI คือ Agent Runtime แบบมีขอบเขตสำหรับทำงานกับ Workspace ผ่าน Terminal
โดยใช้ Tool ภายใต้การควบคุม ระบบมี Core Persona เพียงหนึ่งเดียวคือ `boi`
ส่วนผู้ใช้สามารถตั้งชื่อ Agent instance ของตนเองได้เมื่อเปิด TUI ครั้งแรก

ผลิตภัณฑ์แบ่งเป็น 6 Blocks ได้แก่ Service, Core, Various Equipment, Runtime,
Agent Folder และ SubAgent โดย Work 1 เชื่อม 5 Blocks แรกให้ทำงานผ่านเส้นทาง
TUI/CLI ที่ควบคุมร่วมกัน ส่วนการทำงานของ SubAgent ยังคงปิดไว้

## ความสามารถของ Work 1

- TUI และ `boi ask` ใช้ Agent Service และ Engine ชุดเดียวกัน
- วงจร Observe → Decide → Authorize → Act → Verify → Recover พร้อมขอบเขตของ
  steps, tools, tokens, เวลา และ recovery
- Provider ต้องผ่าน qualification อย่างชัดเจนก่อนเข้าสู่ Agent Router
- Local Registry ทำงานแบบ fail-closed โดยเปิดใช้งานได้สูงสุด 15 Skills และ
  15 Tools
- Capability Broker รับผิดชอบการจำแนกความเสี่ยง, approval, timeout และ execution
- Workspace Sandbox จำกัดขอบเขต path พร้อม Approval Panel แบบโต้ตอบใน TUI
- ใช้ถาด `agent-folder` เดียว โดยเก็บ diagnostic ใน `bin` และเก็บ deliverable
  กับ manifest ใน `output`
- รองรับ JSON schema v1, exit code ที่คงที่, input ผ่าน stdin/argv และ
  Automation แบบ read-only ที่ให้ผลแน่นอน

BOI CLI ไม่มีการตอบกลับด้วย AI จำลอง หากต้องการให้ Agent ทำงาน Provider ที่ตั้งค่า
ไว้ต้องผ่าน `boi provider qualify` ก่อน

## สถานะ Work 1

Work 1 เสร็จสมบูรณ์สำหรับเส้นทาง Single-Agent บน Host แล้ว Core Persona,
Qualified Provider Router, Registry แบบมีขอบเขต, Broker, Sandbox, Agent Folder,
TUI Approval และ Automation แบบ read-only ใช้ Agent Service contract เดียวกัน
ส่วน SubAgent และ Automation ที่มี side effect ยังคงตั้งใจปิดไว้สำหรับ Work 2

การทดสอบ built binary ทำบน Windows และ Linux โดยชุดจำลองร่วมครอบคลุม Unicode,
path ที่มีช่องว่าง, โฟลเดอร์ซ้อนและโฟลเดอร์ขนาดใหญ่, การปฏิเสธ approval,
path traversal, binary input, missing input, Registry เสียหาย และ Provider
ที่ยังไม่ผ่าน qualification ส่วนชุด Linux เพิ่มการตรวจ symlink escape
การ cross-build Linux ARM64 และ Android ARM64 เป็น release gate และใช้เป็น
Termux/S25+ compatibility baseline ตามเกณฑ์ที่เจ้าของระบบยอมรับ โดยไม่ได้กล่าวอ้าง
ว่าได้รันบนโทรศัพท์จริงแล้ว

## ความต้องการของระบบ

- Go 1.24.2 หรือ toolchain รุ่นใหม่กว่าที่เข้ากันได้
- Provider ที่รองรับและตั้งค่าผ่าน environment หรือ setup
- Terminal ที่สามารถแสดงและใช้งาน TUI ได้

## Build และเริ่มต้นระบบ

```text
go build -o boi ./cmd/boi
boi init
boi setup
boi provider qualify <provider-name>
boi registry init
boi
```

`boi registry init` ปลอดภัยและไม่เขียนทับ Registry ที่มีอยู่ การ initialize
Runtime ตามปกติจะสร้างเฉพาะ index ที่หายไป เพื่อให้ Workspace ที่ย้ายมาจากรุ่นเดิม
ทำงานต่อได้โดยไม่เปิดเผย capability file ที่ไม่ได้ลงทะเบียน

`boi setup` จะรักษาค่าอื่นใน `.env`, แทนที่เฉพาะ Provider section ที่ BOI
เป็นผู้จัดการ, สำรองไฟล์เดิมพร้อม timestamp, ใช้ private file permission บนระบบ
Unix-like และเพิ่ม local Git exclude สำหรับ secret ของ BOI การ setup ยังไม่ถือว่า
Provider ผ่าน qualification เพราะ qualification เป็นการทดสอบพฤติกรรมอีกขั้นหนึ่ง

เมื่อเปิด TUI ครั้งแรก ระบบจะถามชื่อ Agent instance และบันทึกไว้ที่
`.boi/agent.yaml` ชื่อนี้ไม่ใช่ Persona หรือ Provider identity

## เริ่มใช้ Work 1 — สร้างผลงานชิ้นแรก

ให้ทำงานภายในโฟลเดอร์โปรเจกต์ที่ต้องการอนุญาตให้ BOI ตรวจสอบและแก้ไข
โฟลเดอร์นี้จะกลายเป็นขอบเขตของ Workspace Sandbox ดังนั้นต้องเปิด BOI จาก
repository หรือโฟลเดอร์ที่มีงานซึ่งต้องการให้ Agent ดำเนินการ

### 1. Build BOI

Windows PowerShell:

```powershell
go build -o boi.exe ./cmd/boi
```

Linux, WSL หรือ Termux:

```bash
go build -o boi ./cmd/boi
chmod +x ./boi
```

ตัวอย่างด้านล่างใช้คำสั่ง `boi` หาก executable ไม่ได้ติดตั้งใน `PATH` ให้ใช้
`.\boi.exe` บน Windows หรือ `./boi` บน Linux/Termux แทน

### 2. Initialize Workspace

```text
boi init
boi registry init
```

การ initialize จะสร้าง runtime state ใน `.boi` และ Local Registry ที่มีขอบเขต
โดยไม่ลบไฟล์โปรเจกต์หรือ Agent output ที่มีอยู่

### 3. เชื่อมต่อและ Qualification Provider

```text
boi setup
boi provider list
boi provider qualify <provider-name>
boi doctor
```

`boi setup` จะเปิด Provider wizard ให้เลือก Provider และ model แล้วกรอก API
credential เมื่อระบบถาม ให้นำชื่อที่แสดงจาก `boi provider list` ไปใช้กับคำสั่ง
qualification ตัวอย่างเช่น:

```text
boi provider qualify openai
```

Qualification จะส่งชุด behavioral probes จริงไปยัง Provider และอาจใช้ API token
ที่มีค่าใช้จ่าย การตั้งค่าเพียงอย่างเดียวยังไม่เพียงพอ Provider ที่ไม่ผ่าน
qualification จะถูกตัดออกจาก Agent Router

### 4. เปิด TUI

```text
boi
```

ในการเปิดครั้งแรก ให้ตั้งชื่อ Agent instance จากนั้นกด `Enter` บน Splash Screen
เพื่อเข้าสู่ Chat ไม่ว่าจะตั้งชื่อ Agent ว่าอะไร Core Persona จะยังคงเป็น `boi`

### 5. สั่งให้ BOI สร้างไฟล์

ใช้ Task แรกที่ปลอดภัยภายใน Test Workspace ที่ initialize แล้ว:

```text
สร้างไฟล์ชื่อ hello-boi.md ใน Workspace นี้ โดยเพิ่มชื่อเรื่อง คำอธิบายสั้น ๆ
ของ repository และ checklist ขั้นตอนถัดไปที่เป็นประโยชน์ 3 ข้อ
ให้อ่านบริบทของโปรเจกต์ที่มีอยู่ก่อนเขียน และรายงาน path สุดท้ายของไฟล์
```

ลำดับการทำงานที่ควรเกิดขึ้น:

1. BOI ตรวจสอบ Workspace ด้วย Tool แบบ read-only
2. Model เสนอ `workspace.write` Tool Call
3. TUI แทนที่ช่อง input ด้วย Approval Panel ซึ่งแสดง purpose, target, risk
   และ preview
4. กด `A` เพื่ออนุมัติการเขียนครั้งนั้นเพียงครั้งเดียว, กด `R` เพื่อปฏิเสธ
   หรือกด `Esc` เพื่อยกเลิก Task โดย `Enter` จะไม่อนุมัติการเขียน
5. BOI ตรวจสอบ Tool Result แล้วรายงาน Task ID และ manifest path

ตรวจผลงานได้โดยไม่ต้องออกจาก TUI:

```text
/ls
/read hello-boi.md
```

ไฟล์ที่สร้างจะอยู่ใน Workspace ส่วน BOI จะบันทึกหลักฐานของ Task ที่สำเร็จไว้ที่:

```text
agent-folder/output/<task-id>/manifest.json
```

Diagnostic ของ Task ที่ล้มเหลว, ถูกปฏิเสธ, ถูกยกเลิก หรืออยู่ระหว่าง recovery
จะอยู่ที่ `agent-folder/bin/<task-id>/` และจะไม่ถูกแสดงเป็น deliverable ที่สำเร็จ

### 6. ทดลองงานตรวจ Repository

หลังจากสร้างไฟล์แรกสำเร็จ ให้ทดลอง Task ที่มีขอบเขตและ output ชัดเจน:

```text
ตรวจสอบ repository นี้แล้วสร้างไฟล์ WORKSPACE_REVIEW.md
ให้สรุปโครงสร้างโปรเจกต์ ระบุความเสี่ยงที่มีหลักฐานจากไฟล์ที่ตรวจพบ 3 ข้อ
และเสนอแผนปรับปรุง 5 ขั้นตอน ห้ามแก้ไขไฟล์อื่น
```

ตรวจ Approval Panel ให้รอบคอบก่อนอนุญาตการเขียน Workspace Sandbox ของ BOI
จำกัดขอบเขต filesystem path แต่ไม่ใช่ OS/container isolation หากเป็นโปรเจกต์
ที่ไม่คุ้นเคยควรรันภายใน environment ที่แยกอย่างเหมาะสมเมื่อต้องการการป้องกัน
ที่แข็งแรงกว่า

### ปุ่มควบคุม TUI ใน Work 1

| Input | การทำงาน |
|---|---|
| `Enter` | ส่งข้อความ Chat และจะไม่ใช้อนุมัติ Tool Call |
| `Ctrl+N` | เพิ่มบรรทัดใหม่ในช่อง input |
| `Tab` | เติม slash command อัตโนมัติ |
| `Esc` หรือ `Ctrl+C` | ยกเลิก Task ที่กำลังทำงาน หรือออกเมื่อระบบว่าง |
| `Ctrl+Q` | ออกจาก TUI ทันที |
| `Ctrl+L` | ล้าง Chat ที่แสดงอยู่ |
| `/workspace` | แสดง Sandbox root ที่กำลังใช้งาน |
| `/ls [path]` | แสดงรายการในโฟลเดอร์ภายใน Workspace |
| `/read <path>` | อ่านไฟล์ข้อความภายใน Workspace |
| `/providers` | แสดงสถานะ Provider ที่ผ่าน qualification |
| `/persona` | แสดง Core Persona และชื่อ Agent instance |

### แก้ปัญหาการใช้งานครั้งแรก

| อาการ | ความหมายและวิธีดำเนินการ |
|---|---|
| `no qualified providers` | รัน `boi provider list` แล้วตามด้วย `boi provider qualify <name>` |
| Provider qualification ไม่ผ่าน | ตรวจ API key, Base URL, model name, network และ Provider quota |
| `boi ask` ปฏิเสธการเขียน | Automation แบบ non-interactive ใน Work 1 เป็น read-only ให้ใช้ Approval ผ่าน TUI |
| เกิด `capability registry` error | รัน `boi registry init` และตรวจ Registry เดิมแทนการเขียนทับไฟล์ที่ไม่ถูกต้อง |
| Workspace path ถูกปฏิเสธ | ใช้ target ภายใน Workspace root และหลีกเลี่ยง symlink/path traversal |
| Binary file ถูกปฏิเสธ | Workspace reader ใน Work 1 รองรับไฟล์ข้อความแบบมีขอบเขต ไม่รองรับ binary content |

ห้าม commit `.env`, API credential หรือไฟล์สำรอง Provider แม้ BOI จะเพิ่ม local
Git exclude ระหว่าง setup ผู้ใช้ยังคงต้องรับผิดชอบความปลอดภัยของ repository
และ credential ของตนเอง

## การใช้งานแบบ Non-interactive

```text
boi ask explain this repository
Get-Content task.txt | boi ask --json --idempotency-key task-001
```

Automation ใน Work 1 เป็น read-only หาก Tool Call ต้องได้รับ approval ระบบจะปฏิเสธ
ในโหมด non-interactive และจะไม่รอ approval prompt ผลลัพธ์ JSON ถูกเขียนไปยัง stdout
เป็น object เดียว ส่วน verbose diagnostic ถูกเขียนไปยัง stderr ดูรายละเอียดได้ที่
[Automation contract](docs/operations/AUTOMATION_CONTRACT.md)

## กลุ่มคำสั่ง

| คำสั่ง | หน้าที่ |
|---|---|
| `boi` | เปิด TUI |
| `boi ask` | รัน Agent แบบมีขอบเขตในโหมด non-interactive |
| `boi setup` | ตั้งค่า Provider ผ่านหน้าจอแบบโต้ตอบ |
| `boi provider list/switch/qualify` | จัดการและ qualification Provider candidate |
| `boi registry init/list/add` | จัดการ Skill และ Tool index ที่ลงทะเบียนอย่างชัดเจน |
| `boi config` / `boi model` | ตรวจหรือเปลี่ยน runtime configuration |
| `boi doctor` | ตรวจสุขภาพระบบภายในเครื่อง |
| `boi skill` / `boi memory` | จัดการ Skill และ local memory |
| `boi persona` | แสดง compatibility contract ของ Core Persona ที่ตรึงไว้ |
| `boi version` / `boi upgrade` | ตรวจเวอร์ชันหรืออัปเดต binary |

ใช้ `boi <command> --help` เพื่อดู flag contract ของ executable รุ่นปัจจุบัน
คำสั่ง legacy `boi run` ไม่ได้อยู่ในเส้นทาง Agent Tool authority
คำสั่งข้อมูล เช่น `--help` และ `version` จะตรวจ Workspace โดยไม่สร้าง state
ใน `.boi` หรือ `agent-folder`

## โครงสร้าง Workspace

```text
workspace/
├── .boi/
│   ├── agent.yaml
│   ├── config.yaml
│   ├── provider-profiles/
│   ├── registry/
│   │   ├── skills.json
│   │   └── tools.json
│   ├── skills/
│   └── memory/
└── agent-folder/
    ├── bin/
    └── output/
```

Manifest ของ Task ที่สำเร็จจะอยู่ใต้ `agent-folder/output/<task-id>/`
ส่วน diagnostic ของ Task ที่ล้มเหลวหรือถูกยกเลิกจะอยู่ใต้
`agent-folder/bin/<task-id>/` การ cleanup จำกัดเฉพาะ `bin` และค่าเริ่มต้นเป็น
dry-run

## ความปลอดภัยและข้อจำกัดปัจจุบัน

- Workspace Sandbox บังคับขอบเขต path แต่ไม่ใช่ OS หรือ container isolation
- Tool ที่เปลี่ยนแปลงข้อมูลต้องได้รับ interactive approval ที่ตรงกับคำขอนั้น
  โดย Automation ที่มี side effect ยังคงปิดอยู่
- มี MCP primitives แล้ว แต่ discovery และ Library routing แบบสมบูรณ์ยังไม่เชื่อม
  เข้าสู่เส้นทาง Agent หลัก
- SubAgent execution ถูกปิดไว้จนกว่าจะผ่าน authority และ evaluation gate แยกต่างหาก
- BOI CLI สามารถใช้ network และไม่ได้ออกแบบแบบ offline-first
- Android ARM64 cross-build ผ่านแล้ว ส่วนการทดสอบบน S25+ จริงเป็น device check
  ที่แนะนำ ไม่ใช่ blocker ของ Work 1 host release โดยใช้ Linux runtime parity
  เป็น Termux/S25+ simulation baseline
- `boi upgrade` ดาวน์โหลดจาก canonical release repository เท่านั้น และตรวจสอบ
  SHA-256 checksum ที่เผยแพร่ก่อนแทนที่ binary

## การตรวจสอบระบบ

```text
go test -count=1 ./...
go vet ./...
go build ./...
```

CI รัน gate เหล่านี้บน Windows และ Linux พร้อม cross-build Android ARM64
สามารถตรวจ WSL parity ด้วย `scripts/acceptance/linux_folder_simulation.py`
และ Linux BOI binary ชุดจำลองใช้ local OpenAI-compatible fixture จึงไม่ถือเป็น
หลักฐานการใช้งานบัญชี Provider ภายนอกจริง

## สถาปัตยกรรมและสถานะ Release

- [แผน Work 1](docs/planning/WORK_1_PLAN.md)
- [เอกสารอ้างอิงคำสั่ง CLI](docs/reference/CLI_COMMANDS.md)
- [บันทึก Release และ Rollback ของ Work 1](docs/operations/WORK_1_RELEASE.md)
- [เอกสารส่งต่องานโปรเจกต์](HANDOFF.md)

License: MIT
