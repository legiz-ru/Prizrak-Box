//go:build !darwin && !linux

package services

// serviceInstalled reports whether the px-service is installed on this machine.
//
// On Windows the service binary is installed next to the app by the MSI and
// registered with the SCM under the name "PrizrakBoxService"; the shipped
// installer treats the presence of the binary as "installed" (mirroring
// src-electron/service.ts), so keep that behaviour here.
func serviceInstalled() bool { return serviceBinaryExists() }
