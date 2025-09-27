
# ERROR CODES (Tight Allocation)

This revision **tightens** code ranges by component so we have plenty of room for upcoming areas
(images, containers, networks, volumes, etc.). Codes remain **3 digits** and are grouped in contiguous
blocks. Each block has **sub‑ranges** when different POSIX exit semantics are helpful.

> Pretty codes = human‑readable 3‑digit codes shown in TTY/logs.  
> POSIX codes = process exit statuses (sysexits).

---

## Global layout (by component)

| Range      | Component                         | Notes / Sub‑ranges                                               |
|-----------:|-----------------------------------|-------------------------------------------------------------------|
| **100–149** | **CORE / CLI usage & args**       | 100–129 usage/validation → `EX_USAGE`; 130–149 data errors → `EX_DATAERR` |
| **200–259** | **REST / transport & HTTP**       | 200–229 transport/TLS → `EX_TEMPFAIL`; 230–259 HTTP/non‑2xx/build → `EX_UNAVAILABLE` |
| **400–499** | **AUTH**                          | 400–429 protocol/authz (probe/basic/bearer) → `EX_NOPERM`; 430–459 token/realm JSON → `EX_DATAERR`; 460–479 dockerConfig IO/JSON → `EX_CONFIG`; 480–499 whoami/logout → `EX_CONFIG` |
| **500–599** | **IMAGES** *(reserved)*           | 500–529 client/protocol; 530–559 JSON/data; 560–599 FS/cache     |
| **600–699** | **CONTAINERS** *(reserved)*       | 600–629 client/protocol; 630–659 JSON/data; 660–699 runtime/FS   |
| **700–739** | **NETWORKS** *(reserved)*         | 700–719 client/protocol; 720–739 JSON/data                       |
| **740–779** | **VOLUMES** *(reserved)*          | 740–759 client/protocol; 760–779 JSON/data/FS                    |
| **800–899** | **FUTURE FEATURES** *(reserved)*  | Leave empty for now                                               |
| **900–949** | **ORCHESTRATION / GLUE**          | 900–919 flow/unsupported → `EX_SOFTWARE`; 920–949 integration     |
| **950–999** | **INTERNAL / RESERVED**           | Keep free                                                         |

> Reserve **300–399** for a future component (if needed).

---

## AUTH (400–499) — standardized

- **400–429 (protocol/authz → `EX_NOPERM`)**
  - `400` Probe `/v2/` failed (unauthorized / challenge missing / unsupported)
  - `401` Basic auth failed
  - `402` Bearer challenge missing/invalid (no realm)
  - `403` Bearer token fetch denied (401/403 from realm)
  - `404` Unsupported auth scheme
- **430–459 (token/realm JSON → `EX_DATAERR`)**
  - `430` Token response decode error
  - `431` Token missing in response
- **460–479 (dockerConfig IO/JSON → `EX_CONFIG`)**
  - `460` Read config.json failed
  - `461` Parse config.json failed
  - `462` Marshal auths failed
  - `463` Write temp config failed
  - `464` Rename temp config failed
  - `465` Invalid/empty registry key
- **480–499 (whoami/logout → `EX_CONFIG`)**
  - `480` WhoAmI: config parse/helper detection error
  - `481` Logout: entry not found (treat as success in CLI, but code exists for logs)
  - `482` Logout: marshal/write failed

> Map your existing auth codes into this range (see Migration section).

---

## REST (200–259) — standardized

- **200–229 (transport/TLS → `EX_TEMPFAIL`)**
  - `200` Invalid base URL / parse error
  - `201` Read CA file failed
  - `202` Append CA to pool failed
  - `203` Load client cert/key failed
  - `204` TLS client construction failed
- **230–259 (HTTP/build → `EX_UNAVAILABLE`)**
  - `230` Request build failed
  - `231` Non‑2xx HTTP status
  - `232` Response decode (JSON) failed
  - `233` Pipe/encoder error

---

## CORE / CLI (100–149) — standardized

- **100–129 (usage/validation → `EX_USAGE`)**
  - `100` Missing required positional/flag
  - `101` Invalid flag combination
  - `102` Unknown subcommand or bad args
- **130–149 (data errors → `EX_DATAERR`)**
  - `130` Invalid value format
  - `131` Unsupported scheme
  - `132` Invalid API version

---

## POSIX mapping (sysexits)

```
EX_OK=0, EX_USAGE=64, EX_DATAERR=65, EX_NOINPUT=66, EX_NOUSER=67, EX_NOHOST=68,
EX_UNAVAILABLE=69, EX_SOFTWARE=70, EX_OSERR=71, EX_OSFILE=72, EX_CANTCREAT=73,
EX_IOERR=74, EX_TEMPFAIL=75, EX_PROTOCOL=76, EX_NOPERM=77, EX_CONFIG=78
```

### Mapping table (by sub‑range)

- 100–129 → `EX_USAGE`
- 130–149 → `EX_DATAERR`
- 200–229 → `EX_TEMPFAIL`
- 230–259 → `EX_UNAVAILABLE`
- 400–429 → `EX_NOPERM`
- 430–459 → `EX_DATAERR`
- 460–479 → `EX_CONFIG`
- 480–499 → `EX_CONFIG`
- 500–599 *(images, TBD)* → choose per sub‑range
- 600–699 *(containers, TBD)* → choose per sub‑range
- 700–739 *(networks, TBD)* → choose per sub‑range
- 740–779 *(volumes, TBD)* → choose per sub‑range
- 900–919 → `EX_SOFTWARE`
- 920–949 → `EX_SOFTWARE`

> You can adjust sub‑range → POSIX mappings per component as they land.

---

## Migration plan

1. **Inventory existing codes** using ripgrep/grep:
   ```bash
   rg -n "CustomError\{[^}]*Code:\s*(\d{3})" -g '!vendor'
   ```
2. **Map old → new** using this table in PR (fill as you touch files):

   | Old | New | Notes |
   |----:|----:|-------|
   | 901 | 400 | Probe `/v2/` failed |
   | 904 | 401 | Basic failed |
   | 906 | 403 | Bearer realm denied / fetch failed |
   | 501 | 460 | Read config failed |
   | 502 | 461 | Parse config failed |
   | 503 | 462 | Marshal auths failed |
   | 504 | 463 | Write temp failed |
   | 505 | 464 | Rename temp failed |
   | 601 | 480 | WhoAmI parse/helper error |
   | 602 | 482 | Logout write failed |
   | 101 | 200 | Base URL / transport (if REST) **or** 100 (if CLI usage) — choose context |
   | ... | ... | Continue for each occurrence |

   > Keep the **log messages** stable; only renumber the codes. Do **not** re‑use retired numbers.

3. Update `exitcode.FromPretty` to the new sub‑range mapping (see code below).

---

## Helper: centralized POSIX mapping

```go
// src/exitcode/map.go
package exitcode

func FromPretty(code int) int {
    switch {
    case code>=100 && code<=129: return EX_USAGE
    case code>=130 && code<=149: return EX_DATAERR
    case code>=200 && code<=229: return EX_TEMPFAIL
    case code>=230 && code<=259: return EX_UNAVAILABLE
    case code>=400 && code<=429: return EX_NOPERM
    case code>=430 && code<=459: return EX_DATAERR
    case code>=460 && code<=479: return EX_CONFIG
    case code>=480 && code<=499: return EX_CONFIG
    case code>=500 && code<=599: return EX_SOFTWARE  // adjust as IMAGES lands
    case code>=600 && code<=699: return EX_SOFTWARE  // adjust as CONTAINERS lands
    case code>=700 && code<=739: return EX_SOFTWARE  // adjust as NETWORKS lands
    case code>=740 && code<=779: return EX_SOFTWARE  // adjust as VOLUMES lands
    case code>=900 && code<=949: return EX_SOFTWARE
    default: return EX_SOFTWARE
    }
}
```

---

## Usage notes

- Within a component, **pick the next free number in its sub‑range**.
- If you hit a cross‑cutting failure (e.g., JSON), use that component’s **JSON sub‑range** (e.g., AUTH 430–459).
- If a user error is detected in a command, return a `CustomError{Code: 100, ...}` and let `main` map to `EX_USAGE`.
- Don’t emit `os.Exit(1)` in subcommands; return errors and let top‑level handle mapping.

---

## Future components (reserved blocks)

- **500–599** IMAGES
- **600–699** CONTAINERS
- **700–739** NETWORKS
- **740–779** VOLUMES
- **800–899** FUTURE

Document their sub‑ranges similarly when implemented.
