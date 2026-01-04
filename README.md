[!(https://github.com/jeanfrancoisgratton/dtools2/blob/develop/dtools2-logo_small.png)]
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
- system information (in a future version)
- multi-registry auth facilities
- remote daemon connectivity
- http/https aware


## Features not included
- Advanced networking

# New features

## Blacklist
If you wish to protect some resources (images, networks, volumes, containers) from accidental (or bulk) removal<br>
You add a blacklisted resource this way `dtools blacklist add RESOURCE_TYPE RESOURCE_NAME`
- RESOURCE_TYPE are volume, network, image, container
- RESOURCE_NAME is the name of the resource to be blacklisted

Now, if a resource is blacklisted (say, a container named mycontainer) and you still needed it to be removed, you<br>
would use the `-B` flag, this way: `dtools rmc -B mycontainer`

The blacklist file is located in $HOME/.config/JFG/dtools/blacklist.json, and has the following format:
```json
{
  "Volumes": [
    "myvol-1",
    "myvol-2"
  ],
  "Networks": [
    "mynet-1"
  ],
  "Images": [
    "myimage-2:1.00.00",
    "myimage2:latest"
  ],
  "Containers": [
    "mycontainer-1"
  ]
}
```

## Environment
You can define a default image registry so that commands that need fetching information from it in other commands.

You create an environment file with `dtools env add REGISTRY_URL`.<br>The filepath is ~/.config/JFG/dtools/defaultRegistry.json and looks like this:

```json
{
  "RegistryName": "https://nexus:9820",
  "Comments": "Home nexus repository manager",
  "Username": "jfgratton",
  "EncodedPasswd": "Pm75M/5SbsTVkEPVXy+eQjFudEwWgHf0"
}
```
Please note that for now only the RegistryName field is used, the others will be in future releases.

The default registry is used `dtools system catalog` and `dtools system tags`, which are explained below

## List images (catalog) in a remote registry

This is one of the commands that needs the default registry mentioned above. You use it this way:`dtools system catalog`<br>
The output is in JSON, prettified just like `jq` would do:

```json
[2:00:21|jfgratton@london:src]: dtools sys catalog
🛈 Connecting to: unix:///var/run/docker.sock

{
  "repositories": [
    "apkbuilder",
    "bare_alpine",
    "forgejo",
    "gitea",
    "haproxy",
    "jenkins",
    "mmost",
    "nexus",
    "nginx",
    "postgresql",
    "rpmbuilder",
    "tracker",
    "vault",
    "vwarden"
  ]
}
```

## List all tags off a specific image
Here's how it'd look:

```bash
[2:00:28|jfgratton@london:src]: dtools sys tags nexus
🛈 Connecting to: unix:///var/run/docker.sock

{
  "name": "nexus",
  "tags": [
    "18.00.00",
    "18.01.00",
    "18.05.00",
    "18.06.00",
    "18.10.00",
    "18.20.00",
    "18.20.02",
    "18.21.00",
    "latest"
  ]
}
```

## Prettified output
Most of the commands are prettified with colorized output. We're way past the era of VT220 terminals, might as well use that !<br>
Also all lists are output as tables, for better readability
<br><br>
# Summary of all commands

## Container commands

### info
`dtools container info CONTAINER_NAME`<br>
`dtools info CONTAINER_NAME`
gives extended information about a named container, like this:
```bash
[2:05:12|jfgratton@london:src]: dtools container info nexus           
🛈 Connecting to: unix:///var/run/docker.sock

Container name      nexus
Image               nexus:9820/nexus:18.10.00
Created             2026.01.04 01:56:57
State               exited
Status              exited (255) 7 minutes ago
RW filesystem size  0.001 MB
RootFS size         812.732 MB
Exposed ports       
Mount points        ✅  [nexus-data]:/opt/nexus3/sonatype-work/nexus3
Command             /entrypoint.sh
```

### kill/killall, start/startall, stop/stopall, restart/restartall, pause/unpause, rename
*Note:*
You can skip the command name (`container`) from the subcommand, thus `dtools startall` and `dtools container startall` are both valid.<br><br>

These commands do what the name implies, obviously. An extra touch is that you can pass multiple container names to<br>
the kill/start/stop/restart commands:

```bash
[2:08:59|jfgratton@london:src]: dtools kill bareAlpine nexus;dtools up bareAlpine nexus
🛈 Connecting to: unix:///var/run/docker.sock

⏳ Container bareAlpine STOPPED
⏳ Container nexus STOPPED
🛈 Connecting to: unix:///var/run/docker.sock

⏳ Container bareAlpine STARTED
⏳ Container nexus STARTED
```
Notice that `dtools up` is an alias to `dtools start`. Some commands have aliases set that way, explore with -`h`

### list containers
`dtools container lsc [-r] [-x]`, `dtools lsc [-r] [-x]`

This lists all containers on the daemon
- `-r` : only running containers
- `-x` : provides extended information

### remove containers
`dtools container rmc [flags] CONTAINER_NAME`, `dtools rmc [flags] CONTAINER_NAME`

You can chain multiple containers like this: `dtools rmc [flags] CONTAINER1 CONTAINER2 CONTAINER3` etc

## Image commands

### list images
`dtools image lsi`, `dtools lsi`

### pull/push image from/to a remote registry
`dtools image pull REPO:IMAGE:TAG`, `dtools pull REPO:IMAGE:TAG`<br>
`dtools image push REPO:IMAGE:TAG`, `dtools push REPO:IMAGE:TAG`

### tag an image
`dtools image tag IMAGE:TAG IMAGE:NEW_TAG`, `dtools tag IMAGE:TAG IMAGE:NEW_TAG`

## Network commands

### list networks
`dtools network lsn`, `dtools lsn`

### create networks
`dtools network create [flags] NETWORK_NAME`<br>
The flags are important, have a look at them : `dtools net add -h`

### remove networks
`dtools net remove [flags] NETWORK_NAME`, `dtools net rmn NETWORK_NAME`, `dtools rmn NETWORK_NAME`<br>
You can chain multiple network names to remove them all at once

### attach/detach a network from a container
`dtools network attach NETWORK_NAME CONTAINER_NAME`, `dtools network detach NETWORK_NAME CONTAINER_NAME`

## Volume commands

### create volumes
`dtools volume create [-d DRIVER_NAME] VOLUME_NAME`<br>

A very basic support is offered here. Basically, you should create a volume using the `local` driver.<br>
If you have third-party drivers and those do not need any special treatment such as options, etc, you can use that with the `-d` flag