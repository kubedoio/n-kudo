package firecracker

// MachineConfig represents the Firecracker machine configuration.
// This is sent via PUT /machine-config
type MachineConfig struct {
	VcpuCount       int  `json:"vcpu_count"`
	MemSizeMiB      int  `json:"mem_size_mib"`
	Smt             bool `json:"smt,omitempty"`
	TrackDirtyPages bool `json:"track_dirty_pages,omitempty"`
}

// BootSource represents the kernel boot source configuration.
// This is sent via PUT /boot-source
type BootSource struct {
	KernelImagePath string `json:"kernel_image_path"`
	BootArgs        string `json:"boot_args,omitempty"`
	InitrdPath      string `json:"initrd_path,omitempty"`
}

// Drive represents a disk drive configuration.
// This is sent via PUT /drives/{drive_id}
type Drive struct {
	DriveID           string `json:"drive_id"`
	PathOnHost        string `json:"path_on_host"`
	IsRootDevice      bool   `json:"is_root_device"`
	IsReadOnly        bool   `json:"is_read_only"`
	PartUuid          string `json:"partuuid,omitempty"`
	CacheType         string `json:"cache_type,omitempty"`
	IoEngine          string `json:"io_engine,omitempty"`
	RateLimiter       string `json:"rate_limiter,omitempty"`
	Socket            string `json:"socket,omitempty"`
}

// NetworkInterface represents a network interface configuration.
// This is sent via PUT /network-interfaces/{iface_id}
type NetworkInterface struct {
	IfaceID           string             `json:"iface_id"`
	HostDevName       string             `json:"host_dev_name"`
	GuestMac          string             `json:"guest_mac,omitempty"`
}

// RateLimiter represents rate limiting configuration.
type RateLimiter struct {
	Bandwidth  *TokenBucket `json:"bandwidth,omitempty"`
	Operations *TokenBucket `json:"ops,omitempty"`
}

// TokenBucket represents a token bucket for rate limiting.
type TokenBucket struct {
	OneTimeBurst int64  `json:"one_time_burst,omitempty"`
	RefillTime   int64  `json:"refill_time"`
	Size         int64  `json:"size"`
}

// ActionType represents the type of action to perform on the VM.
type ActionType string

const (
	ActionTypeInstanceStart   ActionType = "InstanceStart"
	ActionTypeInstanceHalt    ActionType = "SendCtrlAltDel"
	ActionTypeInstanceReset   ActionType = "InstanceReset"
	ActionTypeInstancePause   ActionType = "Pause"
	ActionTypeInstanceResume  ActionType = "Resume"
	ActionTypeInstanceReboot  ActionType = "InstanceReboot"
	ActionTypeInstanceShutdown ActionType = "InstanceShutdown"
)

// Action represents an action to perform on the VM.
// This is sent via PUT /actions
type Action struct {
	ActionType ActionType `json:"action_type"`
}

// VMInfo represents VM information returned by GET /
type VMInfo struct {
	ID          string `json:"id,omitempty"`
	State       string `json:"state,omitempty"`
	VmmVersion  string `json:"vmm_version,omitempty"`
	AppName     string `json:"app_name,omitempty"`
}

// FullVmConfig represents the full VM configuration.
type FullVmConfig struct {
	Balloon        interface{}        `json:"balloon,omitempty"`
	Drives         []Drive            `json:"drives,omitempty"`
	BootSource     BootSource         `json:"boot-source,omitempty"`
	Logger         interface{}        `json:"logger,omitempty"`
	MachineConfig  MachineConfig      `json:"machine-config,omitempty"`
	Metrics        interface{}        `json:"metrics,omitempty"`
	MmdsConfig     interface{}        `json:"mmds-config,omitempty"`
	NetworkInterfaces []NetworkInterface `json:"network-interfaces,omitempty"`
}
