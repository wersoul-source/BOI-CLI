# BOI CLI Definition Worksheet

เอกสารนี้เป็นพื้นที่เทียบนิยามระหว่าง **บ๋อย** กับ **หัวหน้าเอ** ก่อนปิด
CLI Product Contract อย่างเป็นทางการ

- คอลัมน์ “นิยามของบ๋อย” เป็นข้อเสนอจากโครงสร้างและพฤติกรรมที่พบในโปรแกรมปัจจุบัน
- คอลัมน์ “นิยามของหัวหน้าเอ” เว้นว่างไว้ให้หัวหน้าเอกำหนดเจตนาที่แท้จริง
- ข้อความในเอกสารนี้ยังไม่ถือเป็นคำสั่งให้เปลี่ยนชื่อ ค่าปริยาย พฤติกรรม หรือสิทธิ์ของ CLI
- หลังเติมนิยามของหัวหน้าเอแล้ว จึงค่อยตัดสินสถานะ `Keep / Change / Remove / Add`

## 1. นิยามองค์ประกอบหลัก

| องค์ประกอบ | นิยามของบ๋อย | นิยามของหัวหน้าเอ |
|---|---|---|
| BOI CLI | ประตูหลักสำหรับเข้าถึง Agent Runtime ของ BOI ผ่าน TUI และคำสั่งแบบ non-interactive ภายใต้ workspace ที่กำหนด | |
| TUI | พื้นที่ทำงานหลักสำหรับสนทนา สังเกตสถานะ และอนุมัติการกระทำของ Agent แบบโต้ตอบ | |
| Agent | วงจรตัดสินใจแบบมีขอบเขต ทำงานตามลำดับ Observe → Decide → Authorize → Act → Verify → Recover โดยไม่มีสิทธิ์อนุมัติตัวเอง | |
| Agent Service | จุดเข้าใช้งาน Agent เพียงชุดเดียวที่ CLI และ TUI ใช้ร่วมกัน เพื่อไม่ให้ policy และพฤติกรรมแยกคนละทาง | |
| Persona | ชุดตัวตน วิธีคิด น้ำเสียง และค่าการใช้โมเดลของ Agent ไม่ใช่สิทธิ์ในการใช้เครื่องมือ | |
| Provider | ช่องทางเชื่อมต่อโมเดล AI ที่ Router สามารถ retry หรือ failover ตามประเภทข้อผิดพลาดและงบที่กำหนด | |
| Skill | ความรู้และขั้นตอนเฉพาะด้านที่เพิ่มวิธีทำงานให้ Agent แต่ไม่เพิ่มสิทธิ์เข้าถึงระบบโดยอัตโนมัติ | |
| Memory | บริบทข้ามรอบการทำงานที่บันทึกและค้นคืนได้ โดยต้องแยก “ข้อมูลที่จำ” ออกจาก “คำสั่งที่มีอำนาจ” | |
| Weight Engine | กลไกอธิบายคะแนนความเกี่ยวข้องหรือความสำคัญของ Memory เพื่อให้ตรวจสอบเหตุผลการเลือกข้อมูลได้ | |
| Workspace | ขอบเขตโปรเจกต์ที่ BOI CLI กำลังให้บริการและเป็นฐานของ path, config, personas, skills และ memory | |
| Workspace Sandbox | ขอบเขตตรวจสอบ path และ symlink ภายใน workspace ไม่ใช่ container หรือ OS sandbox | |
| Capability Broker | เจ้าของทะเบียนเครื่องมือและ policy ฝั่ง host ทำหน้าที่แปลงข้อเสนอจากโมเดลเป็น Tool Call ที่มี risk, approval, timeout และ preview ที่เชื่อถือได้ | |
| Approval | การอนุญาตการกระทำหนึ่งครั้งต่อ Tool Call ที่แสดงจริง ผูกด้วย fingerprint มีวันหมดอายุ และไม่ใช้ Enter เป็นการอนุมัติ | |
| MCP | ช่องทางเพิ่มเครื่องมือภายนอกแบบ opt-in ซึ่งต้องผ่าน Capability Broker และจัดเป็น external risk เสมอ | |
| Subagent | Agent ย่อยที่รับงาน งบ และสิทธิ์แบบแยกขอบเขต ปัจจุบันยังปิดไว้จนกว่าจะมี isolation และ acceptance contract | |

## 2. นิยามคำสั่งระดับผลิตภัณฑ์

| ID | คำสั่ง | พฤติกรรมที่พบในปัจจุบัน | นิยามของบ๋อย | ขอบเขต/ความเสี่ยงที่บ๋อยเสนอ | นิยามของหัวหน้าเอ |
|---:|---|---|---|---|---|
| C01 | `boi` | เปิด TUI เมื่อไม่ระบุคำสั่ง | เปิด Workspace Console หลักสำหรับสนทนา ดูสถานะ และอนุมัติงานของ Agent | Interactive; การเปลี่ยนแปลงต้องผ่าน Approval Panel | |
| C02 | `boi ask QUERY` | ส่งงานให้ Agent Service พร้อม persona, memory, provider และ step limit | ช่องทางถามหรือมอบหมายงานแบบ non-interactive สำหรับผู้ใช้และ automation | Read ทำได้ตาม policy; write/process/external ต้องไม่ auto-approve | |
| C03 | `boi run COMMAND` | รัน shell command โดยใช้ workspace เป็น working directory และตรวจ deny patterns | ช่องทางรันคำสั่งที่ผู้ใช้ระบุเองโดยตรง ไม่ใช่คำสั่งที่ Agent เลือก | Execute risk; ควรแยกให้ชัดว่าเป็น user-authorized direct command แต่ยังไม่ใช่ OS sandbox | |
| C04 | `boi init` | สร้าง `.boi/config.yaml`, skills และ memory ใน directory ปัจจุบัน | ประกาศ directory ปัจจุบันให้เป็น BOI Workspace และสร้างโครงขั้นต่ำแบบปลอดภัย | Change risk; `--force` ต้องระบุสิ่งที่จะถูกเขียนทับให้ชัด | |
| C05 | `boi setup` | เปิด TUI wizard ตั้งค่า provider; `--refresh` ยังใช้ embedded registry | ขั้นตอน onboarding และตรวจความพร้อมของ Provider Supply Chain | Secret-handling; ห้ามแสดง API key และต้องบอกตำแหน่งจัดเก็บ | |
| C06 | `boi config` | แสดงสรุป config หรือ YAML ด้วย `--all` | อ่าน effective configuration พร้อมบอกแหล่งที่มาและลำดับ precedence | Read-only; secrets ต้อง masked เสมอ | |
| C07 | `boi doctor` | ตรวจ Go, workspace, config, personas, skills, memory, providers, binary และ OS | ตรวจสุขภาพระบบและให้คำแนะนำแก้ไขโดยไม่เปลี่ยน state | Read-only diagnostic; สถานะเตือนต้องแยกจากสถานะล้มเหลว | |
| C08 | `boi provider` | เป็น namespace ของคำสั่ง provider | ศูนย์จัดการ Provider Supply Chain และ active routing preference | ไม่มี side effect หากเรียก namespace อย่างเดียว | |
| C09 | `boi provider list` | แสดง provider จากตัวแปร `PSC_*` | แสดง provider ที่พร้อมใช้งาน ลำดับ routing และสถานะโดยไม่เปิดเผย secret | Read-only; ห้ามพิมพ์ credential | |
| C10 | `boi provider switch NAME` | ตรวจชื่อ provider แล้วบันทึกลง config | เปลี่ยน provider ที่ต้องการให้เป็นค่าเริ่มต้นของ workspace | Change config; ต้องบอกว่ามีผลทันทีหรือหลัง restart | |
| C11 | `boi model NAME` | บันทึกชื่อ model ลง config | เลือก model เริ่มต้นภายใต้ provider ปัจจุบัน | Change config; ควร validate compatibility ก่อนบันทึก | |
| C12 | `boi persona` | เป็น namespace ของ persona | ศูนย์จัดการตัวตนและรูปแบบการคิดของ Agent | Persona ไม่ควรเปลี่ยน tool authority | |
| C13 | `boi persona list` | แสดง persona และทำเครื่องหมายตัวที่ active | แสดง persona ที่โหลดได้ แหล่งที่มา และตัวที่ใช้งานอยู่ | Read-only | |
| C14 | `boi persona switch NAME` | เปลี่ยน persona ใน config | เปลี่ยนตัวตนเริ่มต้นของ Agent สำหรับ workspace | Change config; ไม่เปลี่ยน provider/model โดยเงียบ | |
| C15 | `boi persona init` | คัดลอก persona ค่าเริ่มต้นโดยไม่ทับไฟล์เดิม | ติดตั้ง persona มาตรฐานลง workspace เพื่อให้แก้ไขต่อได้ | Change files; default ต้องไม่ overwrite | |
| C16 | `boi persona wizard` | ให้เลือก persona แบบ interactive และอาจปรับ provider/model | onboarding สำหรับเลือกสไตล์การทำงานของ Agent | Change config; ทุก field ที่จะเปลี่ยนควรมี summary ก่อนยืนยัน | |
| C17 | `boi skill` | เป็น namespace ของ skill | ศูนย์จัดการความสามารถเชิงความรู้และ workflow ของ Agent | Skill ไม่ได้รับ capability เพิ่มอัตโนมัติ | |
| C18 | `boi skill list` | โหลดและแสดง skills ใน `.boi/skills` | แสดง skill ที่ Agent มองเห็น พร้อมสถานะ valid/invalid และ requirements | Read-only | |
| C19 | `boi skill init` | สร้างตัวอย่าง `git` และ `web` หากยังไม่มี | ติดตั้ง starter skills ที่ปลอดภัยและตรวจสอบได้ | Change files; ไม่ overwrite | |
| C20 | `boi skill show NAME` | แสดง metadata และ prompt ของ skill | ตรวจสอบเนื้อหา แหล่งที่มา และ requirements ของ skill ก่อนใช้ | Read-only; เนื้อหา skill ถือเป็น untrusted instructions จนผ่าน policy | |
| C21 | `boi memory` | เป็น namespace ของ Phantom DB | ศูนย์จัดการบริบทข้าม session ภายใน workspace | ต้องแยก read, write, delete และ export ให้ชัด | |
| C22 | `boi memory search QUERY` | ค้นสูงสุด 10 รายการพร้อม score | ค้นหลักฐานหรือบริบทที่เกี่ยวข้อง โดยผลค้นหาไม่มี authority เหนือ user/system | Read-only | |
| C23 | `boi memory stats` | แสดงสถิติและตำแหน่งจัดเก็บ | แสดงสุขภาพ ปริมาณ ขนาด และชนิดข้อมูลใน memory store | Read-only; ไม่ควรเปิดเผยเนื้อหาความลับ | |
| C24 | `boi memory save TYPE KEY CONTENT` | บันทึก MemoryEntry ใหม่ด้วย score เริ่มต้น | บันทึกความจำแบบ explicit ที่ผู้ใช้ระบุ พร้อม provenance และเวลา | Change data; ควรมี validation และ duplicate policy | |
| C25 | `boi memory repomap` | สแกนและสรุปโครงสร้าง repository | สร้างภาพรวม workspace เพื่อช่วย Agent ทำความเข้าใจโครงสร้าง | Read-only scan; ต้องมี ignore และขนาดจำกัด | |
| C26 | `boi memory init` | สร้าง `.boi/memory.md` จาก template | สร้างไฟล์ Project Memory ที่มนุษย์อ่านและแก้ไขได้ | Change file; ต้องไม่ overwrite โดยไม่ยืนยัน | |
| C27 | `boi weight` | เป็น namespace ของ Weight Engine | ศูนย์อธิบาย ranking policy ของ memory/context | ไม่มี side effect | |
| C28 | `boi weight explain PATTERN` | ค้น memory หนึ่งรายการแล้วแสดงองค์ประกอบคะแนน | อธิบายว่าทำไม Memory จึงถูกจัดลำดับ เพื่อให้ตรวจสอบและปรับ policy ได้ | Read-only; ต้องแสดงสูตร/ปัจจัยที่มีผล | |
| C29 | `boi upgrade` | ตรวจ release ล่าสุด ดาวน์โหลด แทน binary และ restart; `--check` ตรวจอย่างเดียว | workflow อัปเดตตัวโปรแกรมที่ตรวจ version, artifact และความน่าเชื่อถือก่อนแทน binary | Critical change; ควร verify checksum/signature และรองรับ rollback | |
| C30 | `boi uninstall` | ยืนยันก่อนลบ data และตั้งเวลาลบ binary บน Windows | ถอนเฉพาะส่วนที่ผู้ใช้เลือก พร้อมแสดงรายการผลกระทบและทางกู้คืน | Destructive; default ควรเก็บ data หรือขอเลือกอย่างชัดเจน | |
| C31 | `boi version` | แสดง version, Go build และ architecture label | แสดงข้อมูล build ที่เสถียรสำหรับคนและ automation | Read-only; ควรมี machine-readable mode ในอนาคต | |

## 3. ประเด็นที่ต้องปิดหลังหัวหน้าเอเติมนิยาม

| Contract dimension | ข้อเสนอของบ๋อย | ข้อกำหนดของหัวหน้าเอ |
|---|---|---|
| Command lifecycle | ระบุคำสั่งที่ Keep, Change, Remove และ Add | |
| Arguments and flags | กำหนดชื่อ ชนิด required/default และ compatibility promise | |
| Input streams | ระบุว่าแต่ละคำสั่งรับ argv, stdin, file หรือ TTY prompt อย่างไร | |
| Output streams | แยก normal output ไป stdout และ diagnostic ไป stderr | |
| Exit codes | กำหนดรหัส success, invalid input, denied, cancelled, unavailable และ internal error | |
| Machine-readable output | เลือกคำสั่งที่ต้องมี `--json` และกำหนด schema/version | |
| Config precedence | เสนอ `flags > environment > workspace config > user config > defaults` | |
| Destructive actions | กำหนด preview, confirmation, `--force`, dry-run และ recovery policy | |
| TUI/CLI parity | ใช้ Agent Service และ policy ชุดเดียวกัน แต่ rendering และ approval UX ต่างกันได้ | |
| Compatibility | ไม่เปลี่ยนชื่อ ค่าปริยาย หรือ output shape โดยไม่มี migration/deprecation plan | |

## 4. ช่องสรุปทิศทางจากหัวหน้าเอ

| หัวข้อ | คำตอบของหัวหน้าเอ |
|---|---|
| Outcome หลักของ BOI CLI | |
| ผู้ใช้หลัก | |
| งานหลัก 3 อันดับแรก | |
| สิ่งที่ BOI CLI ต้องไม่ทำ | |
| คำสั่งที่ต้องเพิ่ม | |
| คำสั่งที่ควรถอดออก | |
| รูปแบบ automation/JSON ที่ต้องรองรับ | |
| ขอบเขต security ที่ต้องการ | |
| นิยามคำว่า “พร้อมใช้งานจริง” | |
