// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/31 00:12
// Original filename: src/system/types.go

package system

var JSONoutputfile = ""
var ForceRemove = false
var RemoveUnamedVolumes = true
var RemoveBlacklisted = false

type CatalogResponse struct {
	Repositories []string `json:"repositories"`
	// Some implementations also include this (not guaranteed everywhere).
	Next string `json:"next,omitempty"`
}

type TagsListResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"` // may decode as nil/empty if no tags
}

// InfoResponse matches the most common fields returned by Docker/Podman on GET /info.
// Fields not present in a given daemon are left at their zero values.
type InfoResponse struct {
	ID                string     `json:"ID"`
	Containers        int        `json:"Containers"`
	ContainersRunning int        `json:"ContainersRunning"`
	ContainersPaused  int        `json:"ContainersPaused"`
	ContainersStopped int        `json:"ContainersStopped"`
	Images            int        `json:"Images"`
	Driver            string     `json:"Driver"`
	DriverStatus      [][]string `json:"DriverStatus"`
	LoggingDriver     string     `json:"LoggingDriver"`
	CgroupDriver      string     `json:"CgroupDriver"`
	CgroupVersion     string     `json:"CgroupVersion"`
	KernelVersion     string     `json:"KernelVersion"`
	OperatingSystem   string     `json:"OperatingSystem"`
	OSType            string     `json:"OSType"`
	Architecture      string     `json:"Architecture"`
	NCPU              int        `json:"NCPU"`
	MemTotal          int64      `json:"MemTotal"`
	Name              string     `json:"Name"`
	ServerVersion     string     `json:"ServerVersion"`
	DockerRootDir     string     `json:"DockerRootDir"`
	SystemTime        string     `json:"SystemTime"`

	IndexServerAddress string `json:"IndexServerAddress"`

	Plugins         PluginsInfo    `json:"Plugins"`
	Swarm           SwarmInfo      `json:"Swarm"`
	Runtimes        map[string]any `json:"Runtimes"`
	DefaultRuntime  string         `json:"DefaultRuntime"`
	InitBinary      string         `json:"InitBinary"`
	SecurityOptions []string       `json:"SecurityOptions"`

	HTTPProxy  string `json:"HttpProxy"`
	HTTPSProxy string `json:"HttpsProxy"`
	NoProxy    string `json:"NoProxy"`

	RegistryConfig     RegistryConfig `json:"RegistryConfig"`
	Debug              bool           `json:"Debug"`
	ExperimentalBuild  bool           `json:"ExperimentalBuild"`
	LiveRestoreEnabled bool           `json:"LiveRestoreEnabled"`
}

type PluginsInfo struct {
	Volume        []string `json:"Volume"`
	Network       []string `json:"Network"`
	Authorization []string `json:"Authorization"`
	Log           []string `json:"Log"`
}

type SwarmInfo struct {
	LocalNodeState string `json:"LocalNodeState"`
}

type RegistryConfig struct {
	Mirrors               []string `json:"Mirrors"`
	InsecureRegistryCIDRs []string `json:"InsecureRegistryCIDRs"`
}

// CP STRUCTS
type dockerErrorResponse struct {
	Message string `json:"message"`
}

// containerPathStat matches the decoded JSON inside X-Docker-Container-Path-Stat.
// Mode uses Go's FileMode bit layout (e.g. os.ModeDir == 1<<31).
type containerPathStat struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Mode       int64  `json:"mode"`
	Mtime      string `json:"mtime"`
	LinkTarget string `json:"linkTarget"`
}
