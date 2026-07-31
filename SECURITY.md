# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 1.x.x   | :white_check_mark: |
| 0.x.x   | :white_check_mark: |

## Reporting a Vulnerability

**Do not open a public issue.** Instead, report vulnerabilities privately to:

**boi-family@proton.me**

Please include:
- A description of the vulnerability
- Steps to reproduce
- Affected versions
- Any potential mitigations you've identified

We will respond within 72 hours with:
- Confirmation of receipt
- An initial assessment
- A timeline for resolution

## Process

1. You report privately via email.
2. We acknowledge within 72 hours.
3. We investigate and develop a fix.
4. We release a patch and publish a security advisory on GitHub.
5. We credit you in the advisory (unless you prefer to remain anonymous).

## Scope

Security issues include but are not limited to:
- Remote code execution via skills or agent commands
- Command injection through `boi run`
- Information disclosure through Phantom DB
- File system access beyond configured workspace boundaries
- API key or environment variable leaks

## Out of Scope

- Issues that require physical access to the machine
- Social engineering attacks
- Issues in third-party LLM providers (OpenAI, Anthropic, etc.)

Thank you for helping keep BOI CLI and its users safe.
