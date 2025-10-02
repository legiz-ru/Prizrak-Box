//go:build !windows

package deeplink

func registerProtocol(string, string) error {
	return nil
}
