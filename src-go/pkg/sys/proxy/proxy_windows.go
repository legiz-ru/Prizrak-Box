package sys

import (
	"fmt"
	"log"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Proxy settings live under this key, in HKEY_CURRENT_USER for the account px
// runs as, or under HKEY_USERS\<SID> when a specific user is targeted. The
// latter is what makes the system proxy work while px runs elevated through the
// TUN service: HKEY_CURRENT_USER then resolves to the SYSTEM account's hive,
// which no user application ever reads.
const settingSubKey = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

func OffAll() error {
	return OffAllForUser("")
}

func OffAllForUser(username string) error {
	// На Windows поддерживается работа с конкретным пользователем через HKEY_USERS\{SID}
	if err := OffHttpsForUser(username); err != nil {
		return err
	}
	if err := OffHttpForUser(username); err != nil {
		return err
	}
	if err := OffSocksForUser(username); err != nil {
		return err
	}
	return nil
}

func SetIgnore(ignores []string) error {
	if err := setString("ProxyOverride", strings.Join(ignores, ";"), ""); err != nil {
		return err
	}
	notifyWinInet()
	return nil
}

func ClearIgnore() error {
	if err := setString("ProxyOverride", "", ""); err != nil {
		return err
	}
	notifyWinInet()
	return nil
}

func GetIgnore() ([]string, error) {
	ignores, err := getString("ProxyOverride")
	if err != nil {
		return nil, err
	}
	if ignores == "" {
		return []string{}, nil
	}
	return strings.Split(ignores, ";"), nil
}

func OnHttps(addr Addr) error {
	return OnHttpsForUser(addr, "")
}

func OnHttpsForUser(addr Addr, username string) error {
	// На Windows поддерживается работа с конкретным пользователем через HKEY_USERS\{SID}
	if err := setString("ProxyServer", addr.String(), username); err != nil {
		return err
	}
	if err := useProxyForUser(true, username); err != nil {
		return err
	}
	notifyWinInet()
	return nil
}

func OffHttps() error {
	return OffHttpsForUser("")
}

func OffHttpsForUser(username string) error {
	// На Windows поддерживается работа с конкретным пользователем
	if err := useProxyForUser(false, username); err != nil {
		return err
	}
	if err := setString("ProxyServer", "", username); err != nil {
		return err
	}
	notifyWinInet()
	return nil
}

func OnHttp(addr Addr) error {
	return OnHttpForUser(addr, "")
}

func OnHttpForUser(addr Addr, username string) error {
	// На Windows username игнорируется
	return nil
}

func OffHttp() error {
	return OffHttpForUser("")
}

func OffHttpForUser(username string) error {
	// На Windows username игнорируется
	return nil
}

func OnSocks(addr Addr) error {
	return OnSocksForUser(addr, "")
}

func OnSocksForUser(addr Addr, username string) error {
	// На Windows username игнорируется
	return nil
}

func OffSocks() error {
	return OffSocksForUser("")
}

func OffSocksForUser(username string) error {
	// На Windows username игнорируется
	return nil
}

func GetHttp() (*Addr, error) {
	// 检查代理是否启用
	enabled, err := getProxy()
	if err != nil {
		return nil, err
	}
	if !enabled {
		// 如果代理未启用，返回 nil
		return nil, nil
	}

	// 获取代理服务器地址
	addr, err := getString("ProxyServer")
	if err != nil {
		return nil, err
	}
	if addr == "" {
		return nil, nil
	}

	// 解析 HTTP 代理地址
	if strings.Contains(addr, "=") {
		// 如果 ProxyServer 包含多个协议的代理地址，提取 http= 部分
		parts := strings.Split(addr, ";")
		for _, part := range parts {
			if strings.HasPrefix(part, "http=") {
				addr = strings.TrimPrefix(part, "http=")
				break
			}
		}
	} else {
		// 如果只有一个代理地址，直接使用
		addr = strings.TrimSpace(addr)
	}

	// 返回解析后的地址
	return ParseAddrPtr(addr), nil
}

// --- WinINet notification ---------------------------------------------------

var (
	modWininet            = windows.NewLazySystemDLL("wininet.dll")
	procInternetSetOption = modWininet.NewProc("InternetSetOptionW")
)

const (
	internetOptionRefresh         = 37
	internetOptionSettingsChanged = 39
)

// notifyWinInet tells Windows that the proxy configuration changed.
//
// Writing the registry is only half of setting the system proxy: WinINet caches
// the configuration, and so does every process that has already read it. Until
// it is told otherwise, the old values stay in effect — which looks exactly like
// "the proxy was set but nothing goes through it", and resolves only when some
// unrelated event (opening the Windows proxy settings page, a network change)
// happens to force a refresh. This never used to be sent at all.
//
// Best-effort: a failure here leaves the registry correct and the refresh late,
// which is the old behaviour, so it is logged rather than surfaced.
func notifyWinInet() {
	if err := procInternetSetOption.Find(); err != nil {
		log.Printf("[SystemProxy] wininet.dll unavailable, proxy change will apply lazily: %v", err)
		return
	}
	// SETTINGS_CHANGED invalidates the cached configuration; REFRESH makes
	// WinINet re-read it from the registry. Both are documented as taking a null
	// handle and no buffer.
	if r, _, err := procInternetSetOption.Call(0, internetOptionSettingsChanged, 0, 0); r == 0 {
		log.Printf("[SystemProxy] InternetSetOption(SETTINGS_CHANGED) failed: %v", err)
	}
	if r, _, err := procInternetSetOption.Call(0, internetOptionRefresh, 0, 0); r == 0 {
		log.Printf("[SystemProxy] InternetSetOption(REFRESH) failed: %v", err)
	}
}

// --- Registry access --------------------------------------------------------

// getUserSID resolves a username to its SID string.
//
// This used to shell out to PowerShell for a WMI query, which cost a process
// spawn (and a hidden console) on every proxy change; LookupSID is the same
// lookup without either.
func getUserSID(username string) (string, error) {
	if username == "" {
		return "", nil
	}
	sid, _, _, err := windows.LookupSID("", username)
	if err != nil {
		return "", fmt.Errorf("failed to get SID for user %q: %w", username, err)
	}
	return sid.String(), nil
}

// settingsKeyFor returns the root and subkey holding the proxy settings for the
// given user (empty username = the account px itself runs as).
func settingsKeyFor(username string) (registry.Key, string, error) {
	if username == "" {
		return registry.CURRENT_USER, settingSubKey, nil
	}
	sid, err := getUserSID(username)
	if err != nil {
		return 0, "", err
	}
	return registry.USERS, sid + `\` + settingSubKey, nil
}

// openSettingsWrite opens the settings key for writing, creating it if needed
// (the previous `reg add` created it implicitly).
func openSettingsWrite(username string) (registry.Key, error) {
	root, path, err := settingsKeyFor(username)
	if err != nil {
		return 0, err
	}
	key, _, err := registry.CreateKey(root, path, registry.SET_VALUE)
	if err != nil {
		return 0, fmt.Errorf("open %s for writing: %w", path, err)
	}
	return key, nil
}

func setString(name string, value string, username string) error {
	key, err := openSettingsWrite(username)
	if err != nil {
		if username != "" {
			log.Printf("[SystemProxy] Failed to open settings for user %s: %v", username, err)
		}
		return err
	}
	defer key.Close()

	if username != "" {
		log.Printf("[SystemProxy] Setting %s=%s for user %s", name, value, username)
	}
	if err := key.SetStringValue(name, value); err != nil {
		log.Printf("[SystemProxy] Failed to set %s: %v", name, err)
		return err
	}
	return nil
}

func setDword(name string, value uint32, username string) error {
	key, err := openSettingsWrite(username)
	if err != nil {
		if username != "" {
			log.Printf("[SystemProxy] Failed to open settings for user %s: %v", username, err)
		}
		return err
	}
	defer key.Close()

	if username != "" {
		log.Printf("[SystemProxy] Setting %s=%d for user %s", name, value, username)
	}
	if err := key.SetDWordValue(name, value); err != nil {
		log.Printf("[SystemProxy] Failed to set %s: %v", name, err)
		return err
	}
	return nil
}

// getString reads a string value for the account px runs as. Reads stay on
// HKEY_CURRENT_USER: they back GetHttp/GetIgnore, which describe the proxy px
// itself would use.
func getString(name string) (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, settingSubKey, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()

	value, _, err := key.GetStringValue(name)
	if err == registry.ErrNotExist {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func useProxyForUser(enabled bool, username string) error {
	var value uint32
	if enabled {
		value = 1
	}
	return setDword("ProxyEnable", value, username)
}

func getProxy() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, settingSubKey, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer key.Close()

	value, _, err := key.GetIntegerValue("ProxyEnable")
	if err == registry.ErrNotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return value != 0, nil
}
