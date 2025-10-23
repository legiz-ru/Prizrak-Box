//go:build !windows && !linux && !darwin

package utils

func collectDeviceDetails() DeviceDetails {
	return DeviceDetails{}
}
