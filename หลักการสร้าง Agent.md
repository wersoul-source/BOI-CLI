รายงานการวิเคราะห์สถาปัตยกรรม เฟรมเวิร์ก และแนวทางการพัฒนา Agent CLI

1. บทนำ: Agent CLI คืออะไร? (Overview & Definition)

Agent CLI (Command Line Interface) คือ เครื่องมือหรือระบบปัญญาประดิษฐ์ที่ทำงานผ่านทางเทอร์มินัล (Terminal / Shell) โดยมีความสามารถในการรับคำสั่งภาษาธรรมชาติ (Natural Language) แปลงคำสั่งเป็นขั้นตอนการทำงาน (Action Plan) และดำเนินการสั่งงานระบบปฏิบัติการ ไฟล์ หรือ API ต่างๆ ได้อย่างต่อเนื่องด้วยตนเอง (Autonomous / Semi-autonomous)

แตกต่างจาก CLI ทั่วไปที่เป็นเพียงโปรแกรมแบบ Deterministic (สั่ง A ได้ผล A ตามกฎที่เขียนไว้ตายตัว) Agent CLI มีสมองหลักเป็น Large Language Model (LLM) ซึ่งสามารถ:

Perceive (รับรู้): อ่านไฟล์ Context, โครงสร้าง Directory, Terminal Output, Log, และสถานะของระบบ

Reason (คิดและวางแผน): ใช้เทคนิคการคิดแบบ ReAct (Reasoning + Acting), Planning, หรือ Chain-of-Thought เพื่อแยกแยะงานใหญ่เป็นงานย่อย

Act (ลงมือทำ): สั่งรัน Bash Command, แก้ไขไฟล์ (Diff/Patch), เรียกใช้ External Tools หรือ MCP (Model Context Protocol)

Reflect & Self-Correct (ประเมินและแก้ไข): อ่าน Error/Stdout/Stderr ที่คืนค่ามา และปรับแผนการแก้ไขจนกว่างานจะสำเร็จ

2. สถาปัตยกรรมและหลักการทำงานพื้นฐาน (Core Architecture & Principles)

การออกแบบ Agent CLI ที่มีประสิทธิภาพประกอบด้วย 5 แกนหลัก ดังนี้:

┌─────────────────────────────────────────────────────────┐
│                      User Prompt                        │
└───────────────────────────┬─────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────┐
│               Agent CLI Engine (Loop)                   │
│  ┌──────────────┐   ┌──────────────┐   ┌─────────────┐  │
│  │   Planner    │──>│  Tool Call   │──>│ Execution   │  │
│  └──────────────┘   └──────────────┘   └─────────────┘  │
│         ▲                                     │         │
│         └────────── Observation Log ──────────┘         │
└───────────────────────────┬─────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────┐
│  Environment: Terminal / File System / Git / MCP Server │
└─────────────────────────────────────────────────────────┘


2.1 ReAct Loop (Reasoning + Acting)

Thought Step: Agent อ่าน Prompt + Context แล้วสร้างความคิดสั้นๆ ว่าขั้นตอนต่อไปต้องทำอะไร

Action Step: Agent เลือก Tool หรือ Shell Command ที่ต้องการรัน

Observation Step: รับค่า Result/Error จากการรัน นำกลับไปสอดใส่เข้าสู่ Context Window เพื่อคิดก้าวต่อไป

2.2 Tool Calling & Code Execution Paradigm

มี 2 รูปแบบหลักในการสั่งงาน:

JSON/Function Calling: LLM คืนค่าเป็น Schema สั่งให้ CLI รัน Tool ที่กำหนด (เช่น read_file, write_file, execute_bash)

Code Agent (Python/Bash-as-Tool): LLM เขียนโค้ด Python หรือ Shell script โดยตรงเพื่อประมวลผล เช่น แนวคิดของ Hugging Face smolagents หรือ OpenInterpreter ซึ่งลดข้อจำกัดของ JSON Schema

2.3 Context Window & Memory Management

Task Tree / Task Graph: แบ่งงานเป็นสับเซตเล็กๆ เพื่อเก็บเฉพาะ Log ของขั้นตอนปัจจุบัน

Prompt Caching / Token Caching: เก็บส่วน System Prompt และ Repository Map ไว้ใน Cache

Diff-Based Editing: แทนที่จะส่งไฟล์เต็มทั้งไฟล์ เปลี่ยนเป็นการส่ง Unified Diff หรือ Block Line Edit เพื่อประหยัด Token

2.4 Human-in-the-Loop (HITL) & Sandboxing

Safe / Dry-Run Mode: ให้ผู้ใช้ตรวจสอบ Bash Command หรือ Git Commit ก่อนรันจริง

Sandboxing: รันคำสั่งอันตรายผ่าน Docker, E2B, หรือ WASM Container

2.5 Standardized Protocols (MCP & Agent Skills)

Model Context Protocol (MCP): โปรโตคอลมาตรฐานจาก Anthropic ในการเชื่อมต่อ LLM กับภายนอก (File, Database, Github, CLI)

Agent Skills Standard: มาตรฐานการนิยามโฟลเดอร์ทักษะ (SKILL.md) เพื่อให้ Agent สามารถเรียกใช้ Script หรือ CLI สั่งการภายนอกได้

3. เจาะลึก 6 เฟรมเวิร์กและ Agent CLI สำคัญเพิ่มเติม

3.1 Agent Zero (Dynamic & Organic Execution Architecture)

จุดเด่นหลัก: สถาปัตยกรรมแบบ "Computer as a Tool" ไร้กรอบบังคับตายตัว (No Hardcoded Rails) ตัวระบบได้รับการออกแบบมาให้ใช้ระบบปฏิบัติการเป็นเครื่องมือหลัก

กลไกการทำงาน:

Agent Zero ไม่เน้นการทำ Pre-defined Tools แต่เน้นให้ LLM เขียนโค้ด Python และสั่ง Exec ผ่าน Terminal เพื่อสร้าง Tool ขึ้นมาใหม่เองตามสถานการณ์

มีระบบ Adaptive Memory & Context Engineering เก็บประวัติวิธีแก้ปัญหาไว้ใช้ซ้ำ

รองรับการทำงานแบบ Multi-Agent Cooperation และการควบคุมผ่าน Terminal/UI

มีแนวทางการคุม Workflow ด้วย State Machine (Plan -> Build -> Diff -> QA -> Approval -> Apply -> Docs)

3.2 Hermes Agent (Nous Research & Self-Evolution Ecosystem)

จุดเด่นหลัก: Open-source Agent Framework จาก Nous Research ที่โดดเด่นด้วยระบบ Self-Evolution & Closed Learning Loop

กลไกการทำงาน:

ใช้เฟรมเวิร์ก DSPy + GEPA (Genetic-Pareto Prompt Evolution) ในการปรับแต่ง System Prompts, Tool Descriptions และ Code ของตัวเองโดยอัตโนมัติจาก Execution Traces

มีระบบรันได้หลากหลาก Environment (6 Terminal Backends: Local, Docker, SSH, Singularity, Modal, Daytona)

เชื่อมต่อกับอินเทอร์เฟซภายนอกได้มากกว่า 20 แพลตฟอร์ม รวมถึงระบบ Cron Scheduling และ MCP Integration

3.3 Grok Build & Grok CLI (xAI Engine)

จุดเด่นหลัก: AI Agent CLI และ TUI สำหรับสายพัฒนาที่ดึงประสิทธิภาพจากโมเดล Grok 4.5 ของ xAI

กลไกการทำงาน:

โดดเด่นด้าน Native Tool Calling Speed และ Context Window ขนาดใหญ่ (สูงสุด 500k tokens)

รองรับการทำงานทั้งผ่าน Fullscreen Interactive TUI (เปิดให้จัดการไฟล์ โค้ด และ Shell) และ Headless Mode สำหรับ Scripting

รองรับ Agent Client Protocol (ACP) เพื่อสร้างการเชื่อมต่อระหว่าง CLI กับเครื่องมือภายนอกหรือ IDE

3.4 Meta Llama Stack (Standardized Agent Infrastructure)

จุดเด่นหลัก: สถาปัตยกรรมโครงสร้างพื้นฐานระดับมาตรฐานจาก Meta สำหรับสร้าง Agentic Applications

กลไกการทำงาน:

สร้าง Unified Agentic APIs ซ้อนอยู่บน Open-Weight Models (Llama 3.x) ทำหน้าที่นามธรรม (Abstract) Layer สำหรับ Inference, Memory, Tool Calling และ Web Search Grounding

รองรับ Multi-agent Orchestration ช่วยให้สามารถประกอบสร้าง Agent CLI หลายโปรไฟล์ทำงานร่วมกันแบบกระจายศูนย์

3.5 OpenClaw (Personal AI Assistant & Gateway Architecture)

จุดเด่นหลัก: Autonomous Personal Agent Framework แบบ Open-source ที่เน้นการรัน Daemon บนเครื่อง Local

กลไกการทำงาน:

ทำงานผ่านสถาปัตยกรรม Gateway Service (ติดตั้งผ่าน openclaw onboard) รันเป็น Background Daemon (Launchd / Systemd)

มีการประมวลผลอินเทอร์เฟซแบบ Live Canvas (A2UI) รองรับการส่งคำสั่งผ่าน Messaging Channels (Discord, Slack, Telegram) และรัน Shell / Browser Tools ได้จากระยะไกล

3.6 Coder / AI Coding Agents Ecosystem (Claude Code, GitHub Copilot CLI, OpenCode, Cline)

จุดเด่นหลัก: กลุ่มเครื่องมือ Terminal-Native AI Coding Agents สำหรับนักพัฒนา

กลไกการทำงาน:

Claude Code / Copilot CLI: เน้นการทำงานร่วมกับ Repository Context, Git History และการรัน Test Execution โดยตรงใน Shell

OpenCode / Pi Agent: เฟรมเวิร์กสาย Open-source ในการสร้าง Terminal Agent ที่มีความประหยัด High-throughput และสลับ Provider LLM ได้อิสระ

Cline CLI / Kilo: เน้นระบบ Plan & Act Mode, Checkpoint/Undo File Edits และการรัน Background Terminal Services

4. สรุปประเภทและกลุ่มของ Agent CLI ในปัจจุบัน

กลุ่มประเภท (Category)

ตัวอย่างโครงการ / เฟรมเวิร์ก

ลักษณะเฉพาะ

Terminal-Native Coding Agents

Claude Code, Aider, GitHub Copilot CLI, OpenCode, Grok Build, Cline

เน้นงานเขียนโค้ด, Multi-file edits, Git integration, Test & Debugging

Autonomous / General-Purpose Agents

Agent Zero, Hermes Agent, OpenClaw, OpenInterpreter

เน้นใช้ OS/Terminal เป็นเครื่องมือหลักสร้าง Tool เอง, มี Gateway Daemon

Agent Building Frameworks

Hugging Face smolagents, Meta Llama Stack, LangGraph, AutoGen, CrewAI

เป็น SDK/Library ให้เดฟนำไปเขียนสร้าง Agent CLI ของตัวเอง

5. รวบรวม 20 แหล่งข้อมูล เฟรมเวิร์ก และ Repository สำคัญ

#

ชื่อโครงการ / เอกสาร

แหล่งที่มา (Platform)

คำอธิบาย / จุดเด่น

1

Hugging Face smolagents

GitHub

Light-weight framework ที่ใช้แนวคิด "Think in Code" ให้ Agent เขียน Python รันบน CLI

2

Agent Zero

GitHub

Dynamic Agent framework ที่ใช้คอมพิวเตอร์และ OS เป็น Tool หลักแบบครบวงจร

3

Nous Research Hermes Agent

GitHub

Agent ที่เรียนรู้และพัฒนาตัวเองได้ผ่าน Closed Loop Learning และ DSPy + GEPA

4

OpenClaw

GitHub

Personal AI Assistant CLI & Gateway Daemon รองรับ Multi-channel และ Live Canvas

5

xAI Grok Build & grok-cli

GitHub / xAI Docs

Terminal Agent ที่ขับเคลื่อนด้วย Grok 4.5 พร้อมระบบ Interactive TUI และ ACP

6

Meta Llama Stack

Meta Developer Portal

โครงสร้างพื้นฐานและ Unified API มาตรฐานสำหรับการสร้าง Agent บน Llama

7

GitHub Copilot CLI

GitHub

Agent CLI อย่างเป็นทางการจาก GitHub เน้นการทำงานร่วมกับ Repository และ MCP

8

Aider

GitHub

Open-source AI pair programming agent ใน Terminal ที่มีความน่าเชื่อถือสูง

9

Claude Code

Anthropic Docs

Agentic CLI Tool จาก Anthropic เน้นการจัดการสถาปัตยกรรมโค้ดขนาดใหญ่

10

OpenInterpreter

GitHub

Open-source code execution agent บน Terminal แบบไร้ข้อจำกัด

11

OpenCode / Anomaly

GitHub

Terminal Agent ที่เน้นความเร็ว รองรับหลาย LLM Provider และเป็น Open-Source

12

Goose

GitHub

AI Agent ภายใต้ Linux Foundation Agentic AI Foundation (AAIF)

13

Plandex

GitHub

Engine ในการวางแผนและสร้างโค้ดแบบ Multi-step ผ่าน Command Line Interface

14

Microsoft AutoGen

GitHub

Multi-agent framework ยอดนิยมจาก Microsoft สำหรับสร้างระบบ Agent แบบสนทนา

15

CrewAI

GitHub

Multi-agent orchestration framework ยอดฮิตพร้อม CLI scaffolding

16

LangChain Agent Infrastructure

GitHub

เครื่องมือมาตรฐานในการสร้าง Agent, Tool binding, และ CLI interfaces

17

Model Context Protocol (MCP)

Official Protocol Site

มาตรฐานการเชื่อมต่อ Context และ Tools ระหว่าง CLI Client กับ Server

18

Cline CLI

GitHub

Autonomous Coding Agent ที่รองรับทั้ง IDE Extension และ CLI Assistant

19

Hugging Face Agent Skills

GitHub

Repository มาตรฐานสำหรับกำหนด Agent Skill และสเปกการเรียกใช้ CLI Tools

20

Awesome AI Coding Tools

GitHub

คลังรวมเครื่องมือพัฒนา AI coding agent และ IDE/CLI integrations

6. ผังโครงสร้างสถาปัตยกรรมระบบ Agent CLI (System Architecture Diagrams)

6.1 ผังโครงสร้างองค์ประกอบหลักของระบบ (Detailed Component Architecture)

┌─────────────────────────────────────────────────────────────────────────────────┐
│                                USER / TERMINAL                                  │
│                 (Natural Language Commands / Flag Arguments)                    │
└────────────────────────────────────────┬────────────────────────────────────────┘
                                         │ User Input Event
                                         ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                            1. INTERFACE & TUI LAYER                             │
│  - Terminal Render Engine (Rich / Bubbletea / Ink)                              │
│  - Input Sanitizer & Command Parser                                             │
│  - Session Manager & Output Stream Handler (Stdout/Stderr)                      │
└────────────────────────────────────────┬────────────────────────────────────────┘
                                         │ Structured Prompt
                                         ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                        2. ORCHESTRATION & REASONING CORE                        │
│  ┌──────────────────────────┐  ┌──────────────────────────┐  ┌───────────────┐  │
│  │      Planner / CoT       │  │  Dynamic Prompt Engine   │  │ Self-Healing  │  │
│  │ (Task Tree Decomposition)│  │(System Spec + MCP Schema)│  │ (Error Loop)  │  │
│  └──────────────────────────┘  └──────────────────────────┘  └───────────────┘  │
└──────────────┬─────────────────────────┬─────────────────────────┬──────────────┘
               │                         │                         │
               ▼                         ▼                         ▼
┌──────────────────────────┐  ┌─────────────────────┐  ┌──────────────────────────┐
│  3. CONTEXT & MEMORY     │  │   4. MODEL ROUTER   │  │   5. TOOL REGISTRY       │
│  - Short-Term Execution  │  │  - Primary LLM API  │  │  - Shell Exec Tool       │
│  - Repo Map / AST Index  │  │  - Fallback Models  │  │  - File Diff Engine      │
│  - Token Optimization    │  │  - Caching Layer    │  │  - MCP Client Connector  │
└──────────────────────────┘  └─────────────────────┘  └──────────────────────────┘
               │                         │                         │
               └─────────────────────────┼─────────────────────────┘
                                         ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                     6. SECURITY & GUARDRAIL SANDBOX LAYER                       │
│  - Human-in-the-Loop Approval Interceptor                                       │
│  - Command Whitelist & Blacklist (e.g. Reject `rm -rf /`)                       │
│  - Sandbox Isolator (Docker / WASM / E2B Container / Restricted Shell)          │
└────────────────────────────────────────┬────────────────────────────────────────┘
                                         │ Safe Action Request
                                         ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                          7. HOST ENVIRONMENT RUNTIME                            │
│  - Local Shell (Bash/Zsh)   - File System Operations   - Git / VCS Control      │
└─────────────────────────────────────────────────────────────────────────────────┘


6.2 ผังลำดับการทำงานและแก้ปัญหาอัตโนมัติ (Execution & Self-Correction Flow Diagram)

graph TD
    A[User Input Command] --> B[CLI Input Sanitizer & Parser]
    B --> C[Load Repo Map & Environment Context]
    C --> D[System Prompt Builder + MCP Tools Registry]
    D --> E[LLM Inference / Reasoning Step]
    
    E --> F{LLM Output Type?}
    F -->|Final Text Answer| G[Stream Output to Terminal TUI]
    F -->|Tool Call Request| H[Guardrail & Security Checker]
    
    H --> I{Action Requires Approval?}
    I -->|Yes| J[Prompt User for Approval y/n]
    J -->|Approved| K[Execute Tool / Shell Command]
    J -->|Denied| L[Feedback User Rejection to Context]
    I -->|No - Safe Action| K
    
    K --> M{Execution Status}
    M -->|Success - Exit 0| N[Capture Stdout & Update History]
    M -->|Error - Non-Zero Exit| O[Capture Error / Traceback]
    
    L --> E
    N --> E
    O --> P[Self-Healing Loop Engine]
    P -->|Construct Error Prompt| E


6.3 ผังโครงสร้างแบบ Multi-Agent Orchestration Diagram

สำหรับงานขนาดใหญ่ Agent CLI จะขยายสถาปัตยกรรมเป็น Multi-Agent Gateway ดังนี้:

                          ┌────────────────────────┐
                          │   Agent Gateway Daemon │
                          └───────────┬────────────┘
                                      │
                         ┌────────────┴────────────┐
                         ▼                         ▼
            ┌────────────────────────┐┌────────────────────────┐
            │    Orchestration /     ││    Context & Memory    │
            │     Master Agent       ││     Manager (Shared)   │
            └────────────┬───────────┘└────────────────────────┘
                         │
        ┌────────────────┼────────────────┬────────────────┐
        ▼                ▼                ▼                ▼
┌───────────────┐┌───────────────┐┌───────────────┐┌───────────────┐
│  Architect    ││  Coder Agent  ││  Test Runner  ││ Review/QA     │
│  Agent        ││  Code Edit)  ││  Agent        ││  Agent        │
└───────┬───────┘└───────┬───────┘└───────┬───────┘└───────┬───────┘
        │                │                │                │
        └────────────────┴────────┬───────┴────────────────┘
                                  ▼
                   ┌────────────────────────────┐
                   │ Secure Execution Sandbox   │
                   │  (Docker / Local Runtime)  │
                   └────────────────────────────┘


7. แนวทางและ Best Practices ในการสร้าง Agent CLI ของตนเอง

หากต้องการพัฒนา Agent CLI ขึ้นมาใช้งานเอง ควรคำนึงถึงปัจจัยสำคัญดังนี้:

1. การเลือก Tech Stack & TUI

ภาษาที่ใช้: Python (Rich / Typer / Prompt Toolkit) เหมาะสำหรับการสร้าง Prototype อย่างรวดเร็ว; Go (Bubbletea / Cobra) หรือ Rust เหมาะกับ CLI ที่ต้องการ Binary ขนาดเล็กและความเร็วสูง

UI/UX: แสดงผล Streaming output, Spinner สถานะการทำงาน, และการไฮไลท์สีโค้ด (Syntax Highlighting) เพื่อให้ผู้ใช้งานทราบว่า Agent กำลังอยู่ในขั้นตอนใด (Reasoning / Executing / Waiting for input)

2. การจัดทำ Prompt Engine & System Spec

กำหนด Constraints ชัดเจน เช่น ห้ามลบไฟล์ระบบ (rm -rf), ห้ามเปลี่ยนแปลงไฟล์นอก Workspace

กำหนดรูปแบบ Output สำหรับ Tool Call ให้แน่นอน เช่น การใช้ Structured JSON, XML tag (เช่น <tool>...</tool>), หรือ Python code block

3. การรับมือกับ ข้อผิดพลาด (Error Recovery Strategy)

เมื่อ Bash command ส่งคืนค่า non-zero exit code หรือเกิด Error Traceback ตัว Agent CLI ต้องไม่หยุดทำงาน แต่ต้องป้อน Error นั้นกลับไปให้ LLM เพื่อหาสาเหตุและรันคำสั่งแก้ไข (Self-Healing Loop)

กำหนด max_iterations (เช่น ไม่เกิน 10-15 รอบ) เพื่อป้องกันการวน Loop ไม่สิ้นสุดเมื่อเกิดข้อผิดพลาดรุนแรง

4. การจัดการความปลอดภัย (Security & Authorization)

ให้มีโหมด Approval Prompt เสมอ หากคำสั่งนั้นเป็นคำสั่งเขียน/ลบไฟล์ หรือรัน Shell Command ที่กระทบกับระบบภายนอก

ใช้ Env Var ในการเก็บ API Keys และหลีกเลี่ยงการเปิดเผย Token ใน Log Files