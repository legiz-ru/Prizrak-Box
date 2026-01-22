package sys

// EnableProxy 开启系统代理
func EnableProxy(host string, port int) error {
	return EnableProxyForUser(host, port, "")
}

// DisableProxy 关闭代理
func DisableProxy() {
	DisableProxyForUser("")
}

// EnableProxyForUser 为指定用户开启系统代理
func EnableProxyForUser(host string, port int, username string) error {
	_ = OnHttpForUser(Addr{
		Host: host,
		Port: port,
	}, username)
	_ = OnHttpsForUser(Addr{
		Host: host,
		Port: port,
	}, username)
	_ = OnSocksForUser(Addr{
		Host: host,
		Port: port,
	}, username)

	return nil
}

// DisableProxyForUser 为指定用户关闭代理
func DisableProxyForUser(username string) {
	_ = OffAllForUser(username)
}
