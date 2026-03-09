package restore

type USBDeviceInfo struct {
	Model      string `json:"model"`
	Generation string `json:"generation"`
	ProductID  uint16 `json:"productId"`
	Mode       string `json:"mode"`
	Restorable bool   `json:"restorable"`
}

func EnumerateIPods() ([]USBDeviceInfo, error) {
	usbIPods, err := DetectUSBIPods()
	if err != nil {
		return nil, err
	}

	var devices []USBDeviceInfo
	for _, u := range usbIPods {
		devices = append(devices, USBDeviceInfo{
			Model:      u.Model.Name,
			Generation: u.Model.Family + " " + u.Model.Generation,
			Mode:       string(u.Mode),
			Restorable: u.Model.Restorable,
		})
	}
	return devices, nil
}
