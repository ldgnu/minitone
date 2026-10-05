---
name: security-review
description: Use when auditing a codebase for security defects, before a release, or when reviewing code that handles untrusted input, secrets, subprocesses, network clients, file paths, or terminal output. Produces ranked, evidence-backed findings with file:line and a concrete reproduction, not generic advice.
---

# Security review

A review is only useful if every claim is **verified against the code that
ships**. No checklist output, no "consider validating input".

## Rules

1. **Every finding needs evidence**: `file:line`, the actual code, and a
   reason it is exploitable in *this* program — not in the abstract.
2. **Confirm before reporting.** Read the whole call path. Most false
   positives come from stopping at the first scary-looking line.
3. **Rank by real impact**, not by scanner severity.
4. **Say what is already fine.** A review that only lists problems gives no
   credit for the hardening that exists.
5. **Never invent a CVE** or claim a vulnerability class without showing the
   path that reaches it.

## Method

Work outward from trust boundaries. For each one, ask what an attacker
controls there.

```bash
# 1. Map the surface
rg -n 'exec\.Command|CommandContext|os\.Open|ReadFile|WriteFile|MkdirAll|http\.|net\.Dial|Unmarshal|filepath\.Join|\.\./' --type go

# 2. Every place untrusted data becomes an action
rg -n 'fmt\.Sprintf\(|Sprint' --type go | rg -v '_test'

# 3. What leaves the machine
rg -n 'fmt\.Print|log\.|Emit\(|Errorf' --type go

# 4. Trust boundaries
rg -n 'InsecureSkipVerify|TLSClientConfig|0600|0644|0755|0777' --type go
```

Then, per finding: **source → sink**. If there is no path from attacker-
controlled data to the sink, it is not a vulnerability.

## Classes worth checking every time

**Subprocess** — Go's `exec.Command` has no shell, so classic `; rm -rf` does
not apply. The real risks are:
- *Argument injection*: an untrusted URL/path/target starting with `-` becomes
  a flag. Fix: `--` separator, or `cmd.Args` validation.
- Path traversal into a different file than intended.
- Inherited environment (`PATH` hijack, `LD_PRELOAD`).
- Never `sh -c` with interpolated input.

**Terminal output** — TUIs print remote data to a terminal. Any attacker-
controlled string carrying `ESC ] … BEL` (OSC) or DCS sequences can rewrite
titles, change the clipboard, or in some terminals execute commands. **Strip
control characters from anything that came off the wire.** This is the single
most commonly missed class in TUI applications.

**Network** — TLS verification on (no `InsecureSkipVerify`), timeouts on every
client, response size limits, redirects bounded, user-agent identifiable.

**Secrets** — never logged, never in error strings, restrictive file modes
(`0600` files, `0700` dirs), and the directory holding them must not be
world-readable if it reveals more than the files do.

**Local state** — JSON files written from remote data: cap read size, ignore
unknown fields, and do not trust them for paths or permissions.

**Unbounded growth** — caches keyed on remote input (images, thumbnails,
results) are a memory-exhaustion DoS. Bound them.

**Concurrency** — data crossing goroutines without synchronisation; a
data race in a privileged path is a security bug, not just a correctness one.

## Reporting

```
### [SEVERITY] Title
**Where:** file:line (and the call path)
**What:** the concrete defect
**Why it matters:** attacker-controlled input reaches <sink> via <path>
**Evidence:** the code, and a repro command or test if you ran one
**Fix:** the smallest change that removes the path
```

Severity:

| | Meaning |
|---|---|
| **Critical** | Remote input reaches code execution or a secret, unauthenticated. |
| **High** | Remote input reads/writes local data, or a secret is exposed. |
| **Medium** | Needs a precondition (a crafted remote response, a local attacker), or leaks information. |
| **Low** | Hardening gap with no direct exploit path today. |

Close the report with: **what is already sound** (so it is not re-litigated),
and **not fixed / out of scope**.