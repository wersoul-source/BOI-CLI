# STABILIZATION PLAN — ทำให้ BOI Agent Suit เสถียรและใช้งานได้จริง

- **สถานะ:** ข้อเสนอ ยังไม่ได้ลงมือแก้ รอเจ้าของโครงการอนุมัติ
- **ที่มา:** [`REPOSITORY_AUDIT_REPORT.md`](REPOSITORY_AUDIT_REPORT.md) (2026-11-20, commit `a05923f`)
- **กรอบ:** ยึดสถาปัตยกรรมที่ออกแบบไว้ทั้งหมด — 6 Blocks, ADR-006 (กฎ 4 ข้อ, สถานะ `NATIVE`/`ASSISTED`/`UNVERIFIED`/`BLOCKED`), D1–D7, F1–F9, E1–E9 ไม่เพิ่ม Block ไม่แตะ Trusted Foundation ไม่ลอกแพลตฟอร์มอื่น
- **กติกาการทำ:** หนึ่งงานหนึ่ง commit ที่ย้อนได้; ทุก PR ตอบ Gap · Composition · Evidence; gate ครบก่อน push (`gofmt`, `go vet`, `staticcheck`, `go test -race`, `go build`); ไม่ push เข้า `master` ตรง

## นิยามของ "เสถียร ใช้งานได้จริง"

ผู้ใช้สั่งงานกับไฟล์จริงบนโมเดลที่ **ไม่สมบูรณ์** แล้วได้หนึ่งในสองอย่างเสมอ: งานเสร็จถูกต้องพร้อมหลักฐาน หรือหยุดพร้อมเหตุผลและทางไปต่อ ไม่มีกรณีที่ตอบผิดแล้วบอกว่าเสร็จ ไม่มีกรณีที่เงียบ

วัดด้วยเกณฑ์ในชั้น S3 ไม่ใช่ความรู้สึก

---

## ชั้น S1 — แก้ทันที (ก่อน release ถัดไป)

เป้าหมาย: ปิดข้อบกพร่องของงานที่ merge แล้ว และทำให้ release เชื่อถือได้ ทั้งชั้นนี้ไม่เปลี่ยนสถาปัตยกรรม

| ID | งาน | Finding | ไฟล์ที่แตะ | เทสต์ที่ต้องมี / เกณฑ์ผ่าน | E |
|---|---|---|---|---|---|
| S1.1 | merge PR #31 | M11 | — | `HANDOFF.md` บน master ตรงกับสถานะจริง | — |
| S1.2 | ผูก reasoning กับ Provider ที่สร้าง: เก็บ `{provider, text}` ต่อ tool turn แล้วส่งกลับเฉพาะเมื่อ Provider ที่กำลังเรียกตรงกัน; เมื่อทิ้งให้บันทึก `route.changed` พร้อมเหตุผล (ตรงกับ F4) | M1 | `internal/runtime/agent/service.go` (map `reasoning`, `nativeObservationMessages`) | เทสต์ zoo ใหม่: cluster `[reasoning-replay ล่มหลัง turn แรก, native]` → request ที่ไปถึง Provider ตัวที่สองไม่มี `reasoning_content`; งานเสร็จ | E3 |
| S1.3 | เอาข้อความ error ออกจาก event: `route.changed` เก็บ `route`, `error_class`, `http_status` เท่านั้น; ข้อความเต็มอยู่ใน notice ของผู้ใช้ตามเดิม | M2 | `internal/runtime/agent/event.go` (`routeChangedPayload`), `service.go` (`noteFallback`) | ขยาย `TestEventsNeverCarryArgumentsOutputOrErrorText` ให้ฝัง secret ใน error ของ Provider แล้วตรวจว่าไม่อยู่ใน `events.jsonl`; แก้ `TestRouteChangeIsAnEvent` ให้ตรวจ status แทนข้อความ | E2, E6 |
| S1.4 | ข้อความ error ของ CLI บอกสาเหตุและทางแก้: `CommandError.Error()` ต่อ `Cause`; กรณีไม่มี Provider บอกให้รัน `boi setup` หรือ `boi quickstart` | M3 | `internal/transport/cli/automation.go`, `agent.go` | เทสต์: `boi ask` ไม่มี Provider → ข้อความมีสาเหตุและคำสั่งแก้; JSON mode ไม่เปลี่ยนรูป | E1 |
| S1.5 | `boi doctor` คืน exit ≠ 0 เมื่อข้อที่จำเป็นไม่ผ่าน (แยกข้อจำเป็นกับข้อแนะนำ) | M10 | `internal/transport/cli/doctor.go` | เทสต์ exit code ทั้งสองกรณี | E1 |
| S1.6 | Release เท่ากับ CI: ให้ `release.yml` เรียก job เดียวกับ `ci.yml` (reusable workflow) ก่อน goreleaser | M4 | `.github/workflows/release.yml`, `ci.yml` | tag ทดสอบบน fork หรือ `workflow_dispatch` แบบ dry-run แสดงว่ารันครบทุกขั้น | — |
| S1.7 | Vulnerability scan: เพิ่ม `govulncheck ./...` ใน CI และ `dependabot.yml` สำหรับ gomod + actions | M5, L14 | `.github/workflows/ci.yml`, `.github/dependabot.yml` | job ผ่านบน master | — |
| S1.8 | ตรวจ scheme ของ base URL: ปฏิเสธ `http://` เมื่อ endpoint ไม่ใช่ `local` เว้นแต่ผู้ใช้ตั้งค่ายอมรับเอง; แสดงใน `boi env` | M8 | `internal/service/provider/factory/factory.go` | เทสต์ตาราง: local http ผ่าน, remote http ถูกปฏิเสธพร้อมข้อความ, https ผ่าน | E7 |
| S1.9 | เก็บงานเล็ก: สิทธิ์ไฟล์ memory เป็น 0600 (L7); เช็ก error ใน `extractTarGz` และจำกัด mode (L1); `permissions: contents: read` และ `concurrency` ใน CI (L4); เปลี่ยนคำ Chimera/Phantom (L9) | L1, L4, L7, L9 | ตามตาราง audit | เทสต์เดิมผ่าน + เทสต์สิทธิ์ไฟล์ | — |

**จบ S1 เมื่อ:** งาน S1.1–S1.8 merge ครบ, CI เขียว, แล้วออก release ได้ (ดู "จุดที่ติด" ข้อ 5 เรื่องเลขเวอร์ชัน)

---

### สถานะ S1 (อัปเดต 2026-11-21)

| ID | สถานะ | หมายเหตุ |
|---|---|---|
| S1.1 | done | PR #31 merge แล้วที่ `8aace34` (รายงานสองไฟล์ไม่ได้ไปด้วย จึงมากับ PR ของ S1) |
| S1.2 | review | `TestReasoningDoesNotCrossProviders`, `TestForeignReasoningIsNotSentAndIsReported` — ยืนยันว่าล้มเมื่อถอดการแก้ |
| S1.3 | review | `TestRouteChangeIsAnEvent`, `TestRouteChangedNeverCarriesTheEndpointsWords` — ยืนยันว่าล้มเมื่อถอดการแก้ |
| S1.4 | review | `TestCommandErrorCarriesItsCauseOnce`; รันด้วยมือแล้วเห็นสาเหตุและทางแก้ |
| S1.5 | review | `TestDoctorVerdict`, `TestDoctorFailsOutsideAWorkspace`; รันด้วยมือได้ exit 5 |
| S1.6 | review | `release.yml` เรียก `ci.yml`; **ยังไม่ได้พิสูจน์บน tag จริง** จนกว่าจะมีการ push tag ครั้งถัดไป |
| S1.7 | review | job `govulncheck` + `dependabot.yml`; ผลจริงดูจาก CI ของ PR |
| S1.8 | review | `TestAKeyIsNotSentInClearToAPublicHost` (8 กรณี); ไม่ได้แสดงใน `boi env` ตามที่วางแผนไว้ — เหตุผลอยู่ใน log และในข้อความ error เมื่อไม่เหลือ Provider |
| S1.9 | review (บางส่วน) | L1, L7 เสร็จ (L4 ทำใน S1.7); **L9 ยังไม่ทำ**: "Chimera" และ "Phantom DB" อยู่บนหน้า splash ของ TUI ด้วย จึงเป็นชื่อของผลิตภัณฑ์ ต้องให้เจ้าของตัดสินว่าจะเปลี่ยนหรือเก็บ |

## ชั้น S2 — ทำให้ใช้งานได้จริงบนโมเดลอ่อน

เป้าหมาย: ปิด H1 และ H2 ด้วยเส้นทางที่สั้นที่สุดบนสถาปัตยกรรมเดิม เลือกเฉพาะงานของระยะ 2–3 ที่จำเป็น งานที่เหลือของแผนใหญ่ไม่ถูกยกเลิก แค่มาทีหลัง

### สถานะ S2 (อัปเดต 2026-11-21)

| ID | สถานะ | หมายเหตุ |
|---|---|---|
| S2.1 | review | `internal/core/recipe.go` + `TestModelZooRecipes`; Runtime ยังไม่อ่าน Recipe (S2.4) จึงยังไม่มีอะไรเปลี่ยนสำหรับผู้ใช้ |
| S2.2–S2.7 | todo | S2.2 (approval floor) ถัดไป |

### ลำดับ

| ID | งาน (ตรงกับแผนใหญ่) | Finding | Block | เกณฑ์ผ่าน | E |
|---|---|---|---|---|---|
| S2.1 | **`Recipe` ใน Core** (P2.1, P2.2): `ComposeRecipe(profile, endpoint facts)` คืนสถานะ 4 แบบต่อ capability พร้อมหลักฐาน, `tool_protocol`, `approval_floor`, budgets | H2 | Core | เทสต์ตาราง: โปรไฟล์ของทุก trait ใน zoo ได้ Recipe ที่คาดไว้; `ASSISTED` ไม่มีทางออกมาเป็น `NATIVE` | E6 |
| S2.2 | **Approval floor** (P2.3, D1/D2): probe ความปลอดภัยตก → เส้นทาง Tool เปิด แต่ทุกคำสั่งต้องอนุมัติผ่าน Broker; non-interactive ปฏิเสธและบันทึกเหตุผล; เส้นทางเป็น `ASSISTED` ได้ก็ต่อเมื่อเทสต์บังคับผ่าน | H2 | Runtime (Broker) | เทสต์บังคับ 3 ข้อของ ADR-006: ทุกคำสั่งถึงการอนุมัติ, `boi ask` ถูกปฏิเสธ, retry/failover หนีไม่ได้; zoo `corrupt-token` ได้ `tools=true (assisted: approval-floor)` | E7 |
| S2.3 | **ตัวตรวจ grounding ของคำตอบ** (ใหม่ ภายใต้ P2.6): เมื่องานอ่านข้อมูลจาก Tool แล้วคำตอบอ้างค่าที่ไม่อยู่ใน observation ให้ติด notice "คำตอบนี้ไม่ตรงกับสิ่งที่อ่านมา" และไม่รายงานว่า verified; ต่อยอด `Ungrounded` ที่มีอยู่ใน Engine | **H1** | Runtime (Verifier) | zoo `corrupt-token`: ผลไม่ใช่ `completed` แบบเงียบ มี notice และ event `verification.recorded{passed:false}`; trait อื่นไม่มี false positive (รัน 10 รอบ) | E6 |
| S2.4 | **ขับสวิตช์จาก Recipe** (P2.5): Service อ่าน Recipe แทน `AgentEnvironment`; กันออกจาก Router เฉพาะ "ไม่มีโปรไฟล์" กับ "completion ล้ม"; เติม `recipe_version` ใน event | H2 | Core → Runtime | เทสต์ acceptance เดิมผ่านครบ; `boi env` แสดงสถานะ 4 แบบ | E6 |
| S2.5 | **Dialect ตาม Recipe** (ต่อยอด P1.3): เปิด dialect parser เฉพาะ endpoint ที่ probe พบ dialect นั้น และข้าม block ใน code fence | M6 | Service, Core | เทสต์: โมเดล native ที่ยกตัวอย่าง `<tool_call>` ใน code fence ไม่เกิด tool call; zoo dialect 3 แบบยังผ่าน | E7 |
| S2.6 | **Recipe ต่อ endpoint ใน Router** (P2.4, D3): failover ใช้ Recipe ของ Provider ใหม่และตรวจอนุมัติซ้ำ; ปล่อย `model.switched` | M1 (ปิดถาวร) | Runtime, Core | zoo: สลับ native → dialect กลางงาน งานเสร็จหรือหยุดพร้อมเหตุผล; สิทธิ์ไม่เพิ่ม | E3 |
| S2.7 | **Resume ขั้นต่ำ** (P3.1–P3.3, P3.5): เก็บบทสนทนาแบบ opt-in, resume จาก checkpoint + event log, ไม่ทำ side effect ซ้ำโดยไม่อนุมัติ | M9 | Agent Folder, Runtime | เทสต์ Crash Recovery: ฆ่า task กลางทาง resume แล้วไฟล์ไม่ถูกเขียนซ้ำ | E4 |

**ตัดออกจากชั้นนี้ (ทำทีหลังได้โดยไม่กระทบเป้าหมาย):** P1.1/P1.2/P1.6 (จัดโครงสร้าง adapter, content parts), P2.8 (โหมดสิทธิ์), P3.4 (undo), P3.6–P3.7 (compaction, circuit breaker), ระยะ 4 ทั้งหมด (Extension, hook, OS sandbox), ระยะ 5

**จบ S2 เมื่อ:** ในเทสต์ zoo ไม่มีแถวใดที่ "ตอบผิดแล้วบอกว่าเสร็จ" และไม่มีโมเดลที่ completion ได้แต่ถูกปิด Tool โดยไม่มีทางเลือกแบบอนุมัติ

---

## ชั้น S3 — พิสูจน์ความเสถียร

เป้าหมาย: ปิด H3 ด้วยหลักฐาน ไม่ประกาศว่าเสถียรจนกว่าผ่านทุกข้อ

| ID | งาน | เกณฑ์ผ่าน (กำหนดล่วงหน้า ตามกฎข้อ 3) |
|---|---|---|
| S3.1 | **Record/replay** (P0.7): helper อัดบทสนทนาจริงเป็น fixture แล้วเล่นซ้ำใน CI | ผลรัน `boi provider qualify maxplus` จริง 1 ชุดเล่นซ้ำได้ใน CI |
| S3.2 | **Acceptance กับ Provider จริง** ตามชุด D6: โมเดลเล็กบนเครื่อง, API ตรง, Maxplus, DeepSeek ตรง — เจ้าของรันด้วยสคริปต์ที่ผมเตรียม (อ่าน key ด้วย `read -s`) | ชุดงาน 10 งาน (อ่าน, ค้นหา, แก้ไฟล์, รันเทสต์, งานหลายขั้น) ต่อ Provider: สำเร็จถูกต้อง ≥ 8/10 และ **0 งานที่ตอบผิดแล้วรายงานว่าเสร็จ**; งานที่ไม่สำเร็จต้องหยุดพร้อมเหตุผล |
| S3.3 | **Soak test บน zoo**: รันชุดงานซ้ำ 200 รอบพร้อม `-race` รวม trait ที่ล่มและช้า | 0 panic, 0 data race, 0 goroutine ค้าง, ผลตรง baseline ทุกรอบ |
| S3.4 | **Coverage ต่อ Block**: ยกผิวสัมผัสผู้ใช้ | `transport/cli` ≥ 55%, `runtime/llm` ≥ 75%, `platform/update` ≥ 60%; gate รวมขยับจาก 55% เป็น 65% |
| S3.5 | **แพลตฟอร์มจริง**: Windows (PowerShell) และ Android (Termux) รัน `boi version`, `doctor`, งาน 1 งานผ่าน TUI และ `boi ask` | เจ้าของรันและบันทึกผลใน `docs/operations` |
| S3.6 | **เซ็น release** (M7): เซ็น `checksums.txt` และให้ `boi upgrade` ตรวจลายเซ็น | upgrade ปฏิเสธ archive ที่ checksum ถูกแก้แม้ `checksums.txt` ถูกแก้ด้วย |
| S3.7 | **ตรวจ Experience Contract**: เทสต์ต่อแถว E1–E9 ที่ S1–S2 แตะ | E1, E3, E4, E6, E7 มีเทสต์ผ่าน; แถวที่ยังไม่มีระบุว่า `UNVERIFIED` ในเอกสาร |

**จบ S3 เมื่อ:** ทุกแถวผ่าน และผลถูกบันทึกในรายงาน acceptance ก่อนจึงเรียกว่า "เสถียร" ได้

---

## จุดที่ติด — ต้องให้หัวหน้าตัดสินหรือทำเอง

| # | เรื่อง | ทำไมผมทำเองไม่ได้ | ข้อเสนอของผม | บล็อกงานไหน |
|---|---|---|---|---|
| 1 | **F9: ให้ BOI เก็บบทสนทนาลงดิสก์ได้ไหม** | เปลี่ยนสิ่งที่เก็บในเครื่องผู้ใช้ ซึ่ง `TaskRecorder` ห้ามไว้ และกฎข้อ 6 ของแผนให้ถามก่อน | อนุมัติแบบ **ปิดเป็นค่าเริ่มต้น ผู้ใช้เปิดเอง** มีการปิดข้อมูลลับและลบอัตโนมัติหลัง 7 วัน | S2.7 |
| 2 | **ผลรัน maxplus จริง (P0.7)** | sandbox เข้า host ไม่ได้ และผมต้องไม่เห็น key | หัวหน้ารัน `boi provider qualify maxplus` 2 ครั้ง แล้วส่งผลที่พิมพ์ออกมา (ไม่มี key ในผล) — ครั้งที่สองบอกได้ว่า token เพี้ยนเกิดทุกครั้งหรือเป็นบางครั้ง | S3.1 และการออกแบบ S2.3 |
| 3 | **ทดสอบ Provider จริง / Windows / Android** | host และเครื่องอยู่นอก sandbox | ผมเตรียมสคริปต์และแบบบันทึกผล หัวหน้ารัน | S3.2, S3.5 |
| 4 | **PR #31 ค้าง** | ผมไม่ merge โดยไม่มีคำสั่ง | สั่ง merge | S1.1 |
| 5 | **เลขเวอร์ชันและ tag ถัดไป** | เลขเวอร์ชันเป็นการตัดสินของเจ้าของ และ session นี้ push tag ไม่ได้ (HTTP 403) | ออก **v2.8.0** หลังจบ S1 (มีฟีเจอร์ใหม่ ไม่ทำลายของเดิม); เก็บ **v3.0.0** ไว้หลัง S3 | release |
| 6 | **กุญแจเซ็น release** | ต้องสร้างและเก็บ private key ใน secret ของ repo ซึ่งผมต้องไม่ถือ | หัวหน้าสร้างคู่กุญแจ ใส่ private key ใน GitHub Secrets ผมเขียนส่วนตรวจ | S3.6 |
| 7 | **ลบ branch เก่า 18 อัน** | เป็นการลบที่ย้อนยาก | ลบได้ทั้งหมดถ้า tag ของแต่ละรุ่นมีอยู่แล้ว ให้หัวหน้ายืนยันก่อน | ไม่บล็อก |

## จุดที่ติดทางเทคนิค — พร้อมทางที่ผมแนะนำ

1. **ตัวตรวจ grounding (S2.3) เสี่ยง false positive** — คำตอบที่สรุปความไม่ได้ลอกคำตรงๆ อาจถูกติดธงผิด
   **ทางที่แนะนำ:** เริ่มแคบ ตรวจเฉพาะค่าที่งานขอให้ "ตอบตามที่อ่าน" (ตัวเลข, identifier, บรรทัดที่ยกมา) และเป็น notice ไม่ใช่การบล็อก วัด false positive บน zoo ก่อนขยาย
2. **Approval floor อาจทำให้ใช้งานน่ารำคาญ** — โมเดลที่ตก probe ต้องอนุมัติทุกคำสั่ง รวมการอ่านไฟล์
   **ทางที่แนะนำ:** ทำตาม D1 ตรงตัวก่อน (อนุมัติทุกคำสั่ง) แล้วใช้ผล S3.2 ตัดสินว่าจะผ่อนให้คำสั่งอ่านใน workspace หรือไม่ การผ่อนเป็นการเปลี่ยนนโยบายความปลอดภัย ต้องให้หัวหน้าอนุมัติ
3. **ไม่มี sandbox ระดับ OS จนกว่าจะถึงระยะ 4** — `process.run` บนโมเดลที่ตก probe ความปลอดภัยพึ่งการอนุมัติของคนอย่างเดียว
   **ทางที่แนะนำ:** ใน S2.2 ให้ `process.run` ของ endpoint ที่อยู่ใต้ approval floor แสดงคำสั่งเต็มและ working directory ในหน้าอนุมัติเสมอ และไม่จำการอนุมัติข้ามคำสั่ง
4. **Resume ต้องการบทสนทนา** — ถ้า F9 ถูกปฏิเสธ S2.7 ทำได้แค่ "เริ่มงานใหม่โดยรู้ว่าไฟล์ไหนถูกแก้ไปแล้ว"
   **ทางที่แนะนำ:** ถ้าปฏิเสธ F9 ให้ลด S2.7 เป็น "รายงานสถานะงานที่ค้าง + กันเขียนซ้ำ" ซึ่งทำได้จาก event log ที่มีอยู่

---

## ลำดับที่แนะนำและขนาดงาน

| ลำดับ | งาน | ขนาด (จำนวน PR โดยประมาณ) |
|---|---|---|
| 1 | S1.1 → S1.2 → S1.3 → S1.4/S1.5 | 3 |
| 2 | S1.6 → S1.7 → S1.8 → S1.9 | 3 |
| — | **ออก release (v2.8.0 ถ้าอนุมัติ)** | — |
| 3 | S2.1 → S2.2 → S2.4 | 3 |
| 4 | S2.3 → S2.5 → S2.6 | 3 |
| 5 | S3.1 → S3.3 → S3.4 (ทำคู่กับชั้น S2 ได้) | 3 |
| 6 | S2.7 (หลังตัดสิน F9) | 2 |
| 7 | S3.2, S3.5 (เจ้าของรัน) → S3.6 → S3.7 | 2 |

ขนาดเป็นการประมาณจากขอบเขตไฟล์ ไม่ใช่คำสัญญาเรื่องเวลา

## ความครอบคลุมของ finding

| Finding | งานที่ปิด |
|---|---|
| H1 | S2.3 |
| H2 | S2.1, S2.2, S2.4 |
| H3 | S3.1, S3.2, S3.5 |
| M1 | S1.2 (แก้), S2.6 (ปิดถาวร) |
| M2 | S1.3 |
| M3 | S1.4 |
| M4 | S1.6 |
| M5 | S1.7 |
| M6 | S2.5 |
| M7 | S3.6 |
| M8 | S1.8 |
| M9 | S2.7 |
| M10 | S1.5 |
| M11 | S1.1 |
| L1, L4, L7, L9, L14 | S1.9, S1.7 |
| L12 | S3.4 |
| L5 | แผนใหญ่ P3.2 (ไม่จำเป็นต่อเป้าหมายนี้) |
| L6 | แผนใหญ่ P1.1 + แยก `runWith` เมื่อแตะใน S2.4 |
| L3 | ยอมรับตาม D7 จนถึง P4.6; ลดความเสี่ยงด้วยทางเทคนิคข้อ 3 |
| L2, L8, L10, L11, L13 | ยอมรับไว้ก่อน: ผลกระทบต่ำ ไม่กระทบเป้าหมาย "เสถียร ใช้งานได้จริง"; L10 รอหัวหน้ายืนยัน |
