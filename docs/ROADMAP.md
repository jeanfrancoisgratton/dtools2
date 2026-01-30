# ROADMAP, INCLUDED FEATURES

## Current version
```bash
# dtools -V
dtools version 2.30.00 (2026.01.11)
```

| Task                                                | Slated for  | Actual release | Comments                             |
|-----------------------------------------------------|-------------|----------------|--------------------------------------|
| [x] auth                                            | 0.10.00     |                |                                      | 
| [x] login                                           | 0.10.00     |                |                                      |
| [x] remote host connection                          | 0.10.00     |                |                                      |
| [x] http(s) thin client                             | 0.10.00     |                |                                      |
| [x] tab completion                                  | 0.10.00     |                |                                      |
| [x] image list                                      | 0.40.00     |                |                                      |
| [x] image pull                                      | 0.10.00     |                |                                      |
| [x] image push                                      | 0.20.00     |                |                                      |
| [x] container list                                  | 0.20.00     | 0.21.00        |                                      |
| [x] container info                                  | 0.20.00     | 0.30.00        |                                      |
| [x] container stop/start                            | 0.30.00     | 0.30.00        |                                      |
| [x] container pause/unpause                         | 0.30.00     | 0.30.00        |                                      |
| [x] container rm                                    | 0.20.00     | 0.50.00        |                                      |
| [x] container rename                                | 0.40.00     | 0.50.00        |                                      |
| [x] container restart                               | 0.55.00     |                | that function had been forgotten !   |
| [x] image rm                                        | 0.40.00     | 0.51.00        |                                      |
| [x] image tag                                       | 0.40.00     | 0.50.00        |                                      |
| [x] network create / rm                             | 0.60.00     |                |                                      |
| [x] network attach / detach                         | 0.60.00     |                |                                      |
| [x] network / volume ls                             | 0.60.00     |                |                                      |
| [x] volume prune / rm                               | 0.80.00     |                |                                      |
| [x] volume ls / create                              | 0.80.00     | 2.12.00        | create was actually added in 2.12.00 |
| [x] exec (shell)                                    | 0.90.00     |                |                                      |
| [x] system info                                     | 2.20.00     |                |                                      |
| [x] blacklist list [-a]                             | 0.20.00     | 0.30.00        | fixed regression in 0.50.00          |
| [x] blacklist rm                                    | 0.20.00     | 0.30.00        | fixed regression in 0.50.00          |
| [x] blacklist add                                   | 0.20.00     | 0.30.00        | fixed regression in 0.50.00          |
| [x] container kill / killall                        | 0.50.00     |                |                                      | 
| [x] `dockerrm.sh`                                   | ~~0.90.00~~ | 2.20.00        | will be implemented in 2.xx.yy       |
| [x] `dockerclean.sh`                                | ~~0.90.00~~ | 2.20.00        | will be implemented in 2.xx.yy       |
| [x] `dockergettag.sh`                               | 0.70.00     |                | porting my bash script to dtools     |
| [x] `dockergetcatalog.sh`                           | 0.70.00     |                | porting my bash script to dtools     |
| [x] rebranding `dtools2`<br>as `dtools 2.00.00`     | 2.00.00     |                |                                      |
| [x] default registry handling (registry subcmd)     | 0.70.00     |                |                                      |
| [x] build command                                   | 2.11.00     |                |                                      |
| [x] logs                                            | 0.90.00     |                |                                      |
| [x] cp                                              | 2.30.00     |                |                                      |
| [ ] load / save                                     | 2.32.00     |                |                                      |
| [ ] import / export                                 | 2.33.00     |                |                                      |
| [ ] commit                                          | 2.33.00     |                |                                      |
| [ ] environment support (`-e` flag) to `dtools run` | 2.31.00     |                |                                      |
| [ ] container attach                                | 2.31.00     |                |                                      |