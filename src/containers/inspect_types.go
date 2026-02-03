// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/02/03 15:54
// Original filename: src/containers/inspect_types.go

package containers

// Inspect command related types

// ContainerInspect represents the full inspection data for a container
type ContainerInspect struct {
	ID              string                  `json:"Id"`
	Created         string                  `json:"Created"`
	Path            string                  `json:"Path"`
	Args            []string                `json:"Args"`
	State           *ContainerState         `json:"State"`
	Image           string                  `json:"Image"`
	ResolvConfPath  string                  `json:"ResolvConfPath,omitempty"`
	HostnamePath    string                  `json:"HostnamePath,omitempty"`
	HostsPath       string                  `json:"HostsPath,omitempty"`
	LogPath         string                  `json:"LogPath,omitempty"`
	Name            string                  `json:"Name"`
	RestartCount    int                     `json:"RestartCount,omitempty"`
	Driver          string                  `json:"Driver,omitempty"`
	Platform        string                  `json:"Platform,omitempty"`
	MountLabel      string                  `json:"MountLabel,omitempty"`
	ProcessLabel    string                  `json:"ProcessLabel,omitempty"`
	AppArmorProfile string                  `json:"AppArmorProfile,omitempty"`
	ExecIDs         []string                `json:"ExecIDs,omitempty"`
	HostConfig      *HostConfig             `json:"HostConfig,omitempty"`
	GraphDriver     *GraphDriver            `json:"GraphDriver,omitempty"`
	SizeRw          int64                   `json:"SizeRw,omitempty"`
	SizeRootFs      int64                   `json:"SizeRootFs,omitempty"`
	Mounts          []InspectMount          `json:"Mounts,omitempty"`
	Config          *ContainerConfig        `json:"Config,omitempty"`
	NetworkSettings *InspectNetworkSettings `json:"NetworkSettings,omitempty"`
}

type ContainerState struct {
	Status     string  `json:"Status"`
	Running    bool    `json:"Running"`
	Paused     bool    `json:"Paused"`
	Restarting bool    `json:"Restarting"`
	OOMKilled  bool    `json:"OOMKilled,omitempty"`
	Dead       bool    `json:"Dead,omitempty"`
	Pid        int     `json:"Pid"`
	ExitCode   int     `json:"ExitCode,omitempty"`
	Error      string  `json:"Error,omitempty"`
	StartedAt  string  `json:"StartedAt,omitempty"`
	FinishedAt string  `json:"FinishedAt,omitempty"`
	Health     *Health `json:"Health,omitempty"`
}

type Health struct {
	Status        string      `json:"Status,omitempty"`
	FailingStreak int         `json:"FailingStreak,omitempty"`
	Log           []HealthLog `json:"Log,omitempty"`
}

type HealthLog struct {
	Start    string `json:"Start,omitempty"`
	End      string `json:"End,omitempty"`
	ExitCode int    `json:"ExitCode,omitempty"`
	Output   string `json:"Output,omitempty"`
}

type HostConfig struct {
	Binds              []string                 `json:"Binds,omitempty"`
	ContainerIDFile    string                   `json:"ContainerIDFile,omitempty"`
	LogConfig          *LogConfig               `json:"LogConfig,omitempty"`
	NetworkMode        string                   `json:"NetworkMode,omitempty"`
	PortBindings       map[string][]PortBinding `json:"PortBindings,omitempty"`
	RestartPolicy      *RestartPolicy           `json:"RestartPolicy,omitempty"`
	AutoRemove         bool                     `json:"AutoRemove,omitempty"`
	VolumeDriver       string                   `json:"VolumeDriver,omitempty"`
	VolumesFrom        []string                 `json:"VolumesFrom,omitempty"`
	CapAdd             []string                 `json:"CapAdd,omitempty"`
	CapDrop            []string                 `json:"CapDrop,omitempty"`
	CgroupnsMode       string                   `json:"CgroupnsMode,omitempty"`
	Dns                []string                 `json:"Dns,omitempty"`
	DnsOptions         []string                 `json:"DnsOptions,omitempty"`
	DnsSearch          []string                 `json:"DnsSearch,omitempty"`
	ExtraHosts         []string                 `json:"ExtraHosts,omitempty"`
	GroupAdd           []string                 `json:"GroupAdd,omitempty"`
	IpcMode            string                   `json:"IpcMode,omitempty"`
	Cgroup             string                   `json:"Cgroup,omitempty"`
	Links              []string                 `json:"Links,omitempty"`
	OomScoreAdj        int                      `json:"OomScoreAdj,omitempty"`
	PidMode            string                   `json:"PidMode,omitempty"`
	Privileged         bool                     `json:"Privileged,omitempty"`
	PublishAllPorts    bool                     `json:"PublishAllPorts,omitempty"`
	ReadonlyRootfs     bool                     `json:"ReadonlyRootfs,omitempty"`
	SecurityOpt        []string                 `json:"SecurityOpt,omitempty"`
	UTSMode            string                   `json:"UTSMode,omitempty"`
	UsernsMode         string                   `json:"UsernsMode,omitempty"`
	ShmSize            int64                    `json:"ShmSize,omitempty"`
	Runtime            string                   `json:"Runtime,omitempty"`
	Isolation          string                   `json:"Isolation,omitempty"`
	CpuShares          int64                    `json:"CpuShares,omitempty"`
	Memory             int64                    `json:"Memory,omitempty"`
	NanoCpus           int64                    `json:"NanoCpus,omitempty"`
	CgroupParent       string                   `json:"CgroupParent,omitempty"`
	BlkioWeight        uint16                   `json:"BlkioWeight,omitempty"`
	CpuPeriod          int64                    `json:"CpuPeriod,omitempty"`
	CpuQuota           int64                    `json:"CpuQuota,omitempty"`
	CpuRealtimePeriod  int64                    `json:"CpuRealtimePeriod,omitempty"`
	CpuRealtimeRuntime int64                    `json:"CpuRealtimeRuntime,omitempty"`
	CpusetCpus         string                   `json:"CpusetCpus,omitempty"`
	CpusetMems         string                   `json:"CpusetMems,omitempty"`
	Devices            []Device                 `json:"Devices,omitempty"`
	DeviceCgroupRules  []string                 `json:"DeviceCgroupRules,omitempty"`
	DiskQuota          int64                    `json:"DiskQuota,omitempty"`
	KernelMemory       int64                    `json:"KernelMemory,omitempty"`
	MemoryReservation  int64                    `json:"MemoryReservation,omitempty"`
	MemorySwap         int64                    `json:"MemorySwap,omitempty"`
	MemorySwappiness   *int64                   `json:"MemorySwappiness,omitempty"`
	OomKillDisable     *bool                    `json:"OomKillDisable,omitempty"`
	PidsLimit          *int64                   `json:"PidsLimit,omitempty"`
	Ulimits            []Ulimit                 `json:"Ulimits,omitempty"`
	CpuCount           int64                    `json:"CpuCount,omitempty"`
	CpuPercent         int64                    `json:"CpuPercent,omitempty"`
	IOMaximumIOps      uint64                   `json:"IOMaximumIOps,omitempty"`
	IOMaximumBandwidth uint64                   `json:"IOMaximumBandwidth,omitempty"`
	MaskedPaths        []string                 `json:"MaskedPaths,omitempty"`
	ReadonlyPaths      []string                 `json:"ReadonlyPaths,omitempty"`
}

type LogConfig struct {
	Type   string            `json:"Type,omitempty"`
	Config map[string]string `json:"Config,omitempty"`
}

type PortBinding struct {
	HostIP   string `json:"HostIp,omitempty"`
	HostPort string `json:"HostPort,omitempty"`
}

type RestartPolicy struct {
	Name              string `json:"Name,omitempty"`
	MaximumRetryCount int    `json:"MaximumRetryCount,omitempty"`
}

type Device struct {
	PathOnHost        string `json:"PathOnHost,omitempty"`
	PathInContainer   string `json:"PathInContainer,omitempty"`
	CgroupPermissions string `json:"CgroupPermissions,omitempty"`
}

type Ulimit struct {
	Name string `json:"Name,omitempty"`
	Soft int64  `json:"Soft,omitempty"`
	Hard int64  `json:"Hard,omitempty"`
}

type GraphDriver struct {
	Name string            `json:"Name,omitempty"`
	Data map[string]string `json:"Data,omitempty"`
}

type InspectMount struct {
	Type        string `json:"Type,omitempty"`
	Name        string `json:"Name,omitempty"`
	Source      string `json:"Source,omitempty"`
	Destination string `json:"Destination,omitempty"`
	Driver      string `json:"Driver,omitempty"`
	Mode        string `json:"Mode,omitempty"`
	RW          bool   `json:"RW,omitempty"`
	Propagation string `json:"Propagation,omitempty"`
}

type ContainerConfig struct {
	Hostname        string              `json:"Hostname,omitempty"`
	Domainname      string              `json:"Domainname,omitempty"`
	User            string              `json:"User,omitempty"`
	AttachStdin     bool                `json:"AttachStdin,omitempty"`
	AttachStdout    bool                `json:"AttachStdout,omitempty"`
	AttachStderr    bool                `json:"AttachStderr,omitempty"`
	ExposedPorts    map[string]struct{} `json:"ExposedPorts,omitempty"`
	Tty             bool                `json:"Tty,omitempty"`
	OpenStdin       bool                `json:"OpenStdin,omitempty"`
	StdinOnce       bool                `json:"StdinOnce,omitempty"`
	Env             []string            `json:"Env,omitempty"`
	Cmd             []string            `json:"Cmd,omitempty"`
	Healthcheck     *HealthConfig       `json:"Healthcheck,omitempty"`
	ArgsEscaped     bool                `json:"ArgsEscaped,omitempty"`
	Image           string              `json:"Image,omitempty"`
	Volumes         map[string]struct{} `json:"Volumes,omitempty"`
	WorkingDir      string              `json:"WorkingDir,omitempty"`
	Entrypoint      []string            `json:"Entrypoint,omitempty"`
	NetworkDisabled bool                `json:"NetworkDisabled,omitempty"`
	MacAddress      string              `json:"MacAddress,omitempty"`
	OnBuild         []string            `json:"OnBuild,omitempty"`
	Labels          map[string]string   `json:"Labels,omitempty"`
	StopSignal      string              `json:"StopSignal,omitempty"`
	StopTimeout     *int                `json:"StopTimeout,omitempty"`
	Shell           []string            `json:"Shell,omitempty"`
}

type HealthConfig struct {
	Test        []string `json:"Test,omitempty"`
	Interval    int64    `json:"Interval,omitempty"`
	Timeout     int64    `json:"Timeout,omitempty"`
	StartPeriod int64    `json:"StartPeriod,omitempty"`
	Retries     int      `json:"Retries,omitempty"`
}

type InspectNetworkSettings struct {
	Bridge                 string                              `json:"Bridge,omitempty"`
	SandboxID              string                              `json:"SandboxID,omitempty"`
	HairpinMode            bool                                `json:"HairpinMode,omitempty"`
	LinkLocalIPv6Address   string                              `json:"LinkLocalIPv6Address,omitempty"`
	LinkLocalIPv6PrefixLen int                                 `json:"LinkLocalIPv6PrefixLen,omitempty"`
	Ports                  map[string][]InspectPortBinding     `json:"Ports,omitempty"`
	SandboxKey             string                              `json:"SandboxKey,omitempty"`
	SecondaryIPAddresses   []InspectAddress                    `json:"SecondaryIPAddresses,omitempty"`
	SecondaryIPv6Addresses []InspectAddress                    `json:"SecondaryIPv6Addresses,omitempty"`
	EndpointID             string                              `json:"EndpointID,omitempty"`
	Gateway                string                              `json:"Gateway,omitempty"`
	GlobalIPv6Address      string                              `json:"GlobalIPv6Address,omitempty"`
	GlobalIPv6PrefixLen    int                                 `json:"GlobalIPv6PrefixLen,omitempty"`
	IPAddress              string                              `json:"IPAddress,omitempty"`
	IPPrefixLen            int                                 `json:"IPPrefixLen,omitempty"`
	IPv6Gateway            string                              `json:"IPv6Gateway,omitempty"`
	MacAddress             string                              `json:"MacAddress,omitempty"`
	Networks               map[string]*InspectEndpointSettings `json:"Networks,omitempty"`
}

type InspectPortBinding struct {
	HostIP   string `json:"HostIp,omitempty"`
	HostPort string `json:"HostPort,omitempty"`
}

type InspectAddress struct {
	Addr      string `json:"Addr,omitempty"`
	PrefixLen int    `json:"PrefixLen,omitempty"`
}

type InspectEndpointSettings struct {
	IPAMConfig          *InspectIPAM      `json:"IPAMConfig,omitempty"`
	Links               []string          `json:"Links,omitempty"`
	Aliases             []string          `json:"Aliases,omitempty"`
	NetworkID           string            `json:"NetworkID,omitempty"`
	EndpointID          string            `json:"EndpointID,omitempty"`
	Gateway             string            `json:"Gateway,omitempty"`
	IPAddress           string            `json:"IPAddress,omitempty"`
	IPPrefixLen         int               `json:"IPPrefixLen,omitempty"`
	IPv6Gateway         string            `json:"IPv6Gateway,omitempty"`
	GlobalIPv6Address   string            `json:"GlobalIPv6Address,omitempty"`
	GlobalIPv6PrefixLen int               `json:"GlobalIPv6PrefixLen,omitempty"`
	MacAddress          string            `json:"MacAddress,omitempty"`
	DriverOpts          map[string]string `json:"DriverOpts,omitempty"`
}

type InspectIPAM struct {
	IPv4Address  string   `json:"IPv4Address,omitempty"`
	IPv6Address  string   `json:"IPv6Address,omitempty"`
	LinkLocalIPs []string `json:"LinkLocalIPs,omitempty"`
}
