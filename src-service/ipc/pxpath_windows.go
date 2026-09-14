//go:build windows

package ipc

import "fmt"

// trustedSystemPath has no Windows implementation: the installer puts px next to
// px-service, which validatePxPath already accepts, so anything else is refused
// rather than approved by a weaker ACL check.
func trustedSystemPath(resolved string) error {
	return fmt.Errorf("%q is not next to the installed px-service", resolved)
}
