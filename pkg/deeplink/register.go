package deeplink

// RegisterProtocol ensures the provided URL scheme launches the current executable.
func RegisterProtocol(scheme, displayName string) error {
	if scheme == "" {
		return nil
	}

	return registerProtocol(scheme, displayName)
}
