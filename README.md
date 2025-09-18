# dtools2

A drop-in client replacement for the official docker and podman clients
___

# Overview
This tool intends to be a mostly-full replacement of the official docker and podman clients. Mostly-full means that all of the basic features (image/container/volume/network management) will be implemented, but some others might not be.

This tool will not rely on the official GO SDKs, but pure REST API calls. Why ? Some of the reasons, here:
- Docker and Podman REST APIs are nearly identical for my use-case
- The GO SDK (well, docker's, at least) keeps changing data structures, function signatures, etc. I had bad experiences with it when the tool would compile cleanly at one point, then I'd update the dependencies (`go mod tidy`), the SDK would have changed so much that I could not compile the software without extensive refactoring.

Also, the idea behind this tool is to make the TUI more attractive, verbose, informative, than the official clients

# Feature list : what's in it, what isn't

## Features included
- image list, pull, push, removal
- container list, creation, removal, start/stop/restart
- network list, creation, removal, attach, detach
- volume list, creation, removal
- execution (shell in the container)
- system information
- multi-registry auth facilities
- remote daemon connectivity
- http/https aware
- maybe more ? :-)


## Features not include
- 