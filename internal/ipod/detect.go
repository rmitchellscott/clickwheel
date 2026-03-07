package ipod

type DeviceInfo struct {
	MountPoint string `json:"mountPoint"`
	Name       string `json:"name"`
	FreeSpace  int64  `json:"freeSpace"`
	TotalSpace int64  `json:"totalSpace"`
}
