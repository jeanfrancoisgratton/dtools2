
# ERROR CODES (Tight Allocation)

This revision **tightens** code ranges by component so we have plenty of room for upcoming areas
(images, containers, networks, volumes, etc.). Codes remain **3 digits** and are grouped in contiguous
blocks. Each block has **sub‑ranges** when different POSIX exit semantics are helpful.

> Pretty codes = human‑readable 3‑digit codes shown in TTY/logs.  
> POSIX codes = process exit statuses (sysexits).

---

## Global layout (by component)

| Range      | Component                         | Notes / Sub‑ranges                                                                                                                                                                  |
|-----------:|-----------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **100–149** | **CORE / CLI usage & args**       | 100–129 usage/validation → `EX_USAGE`; 130–149 data errors → `EX_DATAERR`                                                                                                           |
| **200–259** | **REST / transport & HTTP**       | 200–229 transport/TLS → `EX_TEMPFAIL`; 230–259 HTTP/non‑2xx/build → `EX_UNAVAILABLE`                                                                                                |
| **400–499** | **AUTH**                          | 400–429 protocol/authz (probe/basic/bearer) → `EX_NOPERM`; 430–459 token/realm JSON → `EX_DATAERR`; 460–479 dockerConfig IO/JSON → `EX_CONFIG`; 480–499 whoami/logout → `EX_CONFIG` |
| **500–599** | **IMAGES** *(reserved)*           | 500–529 client/protocol; 530–559 JSON/data; 560–599 FS/cache                                                                                                                        |
| **600–699** | **CONTAINERS** *(reserved)*       | 600–629 client/protocol; 630–659 JSON/data; 660–699 runtime/FS                                                                                                                      |
| **700–739** | **NETWORKS** *(reserved)*         | 700–719 client/protocol; 720–739 JSON/data                                                                                                                                          |
| **740–779** | **VOLUMES** *(reserved)*          | 740–759 client/protocol; 760–779 JSON/data/FS                                                                                                                                       |
| **800–899** | **FUTURE FEATURES** *(reserved)*  | Leave empty for now                                                                                                                                                                 |
| **900–949** | **ORCHESTRATION / GLUE**          | 900–919 flow/unsupported → `EX_SOFTWARE`; 920–949 integration                                                                                                                       |
| **950–999** | **INTERNAL / RESERVED**           | Keep free                                                                                                                                                                           |

> Reserve **300–399** for a future component (if needed).

---
