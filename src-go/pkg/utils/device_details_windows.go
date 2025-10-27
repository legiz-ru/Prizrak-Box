//go:build windows

package utils

import (
	"fmt"
	"strings"

	"github.com/denisbrodbeck/machineid"
	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows/registry"
)

type win32ComputerSystem struct {
	Model string
}

type win32BaseBoard struct {
	Product string
}

func collectDeviceDetails() DeviceDetails {
	details := DeviceDetails{}

	if guid, err := readRegistryString(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, "MachineGuid"); err == nil {
		details.HWID = guid
	}

	if details.HWID == "" {
		if id, err := machineid.ProtectedID("prizrak-box"); err == nil {
			details.HWID = id
		}
	}

	if productName, err := readRegistryString(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "ProductName"); err == nil {
		details.OS = productName
	}

	major, _ := readRegistryUint32(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "CurrentMajorVersionNumber")
	minor, _ := readRegistryUint32(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "CurrentMinorVersionNumber")
	build, _ := readRegistryString(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "CurrentBuildNumber")
	if build == "" {
		build, _ = readRegistryString(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "CurrentBuild")
	}

	if major != 0 || minor != 0 || build != "" {
		if build == "" {
			build = "0"
		}
		details.OSVersion = fmt.Sprintf("%d.%d.%s", major, minor, build)
	}

	if details.OSVersion == "" {
		if version, err := readRegistryString(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "ReleaseId"); err == nil && version != "" {
			details.OSVersion = version
		}
	}

	// Hardware model
	var systems []win32ComputerSystem
	if err := wmi.Query("SELECT Model FROM Win32_ComputerSystem", &systems); err == nil && len(systems) > 0 {
		details.Model = strings.TrimSpace(systems[0].Model)
	}

	var boards []win32BaseBoard
	if err := wmi.Query("SELECT Product FROM Win32_BaseBoard", &boards); err == nil && len(boards) > 0 {
		product := strings.TrimSpace(boards[0].Product)
		if product != "" {
			if details.Model == "" {
				details.Model = product
			} else if !strings.EqualFold(details.Model, product) {
				details.Model = details.Model + "/" + product
			}
		}
	}

	if details.Model == "" {
		if pretty, err := readRegistryString(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "ProductName"); err == nil {
			details.Model = pretty
		}
	}

	return details
}

func readRegistryString(root registry.Key, path, name string) (string, error) {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()

	value, _, err := key.GetStringValue(name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func readRegistryUint32(root registry.Key, path, name string) (uint32, error) {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return 0, err
	}
	defer key.Close()

	value, _, err := key.GetIntegerValue(name)
	if err != nil {
		return 0, err
	}
	return uint32(value), nil
}
