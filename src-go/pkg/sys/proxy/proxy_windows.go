package sys

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/textproto"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	sys "github.com/legiz-ru/prizrak-box/pkg/sys/cmd"
)

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
	return set("ProxyOverride", "REG_SZ", strings.Join(ignores, ";"))
}

func ClearIgnore() error {
	return set("ProxyOverride", "REG_SZ", "")
}

func GetIgnore() ([]string, error) {
	m, err := get("ProxyOverride")
	if err != nil {
		return nil, err
	}
	ignores := m["ProxyOverride"]
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
	err := setForUser("ProxyServer", "REG_SZ", addr.String(), username)
	if err != nil {
		return err
	}

	return useProxyForUser(true, username)
}

func OffHttps() error {
	return OffHttpsForUser("")
}

func OffHttpsForUser(username string) error {
	// На Windows поддерживается работа с конкретным пользователем
	err := useProxyForUser(false, username)
	if err != nil {
		return err
	}

	return setForUser("ProxyServer", "REG_SZ", "", username)
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
	m, err := get("ProxyServer")
	if err != nil {
		return nil, err
	}
	addr := m["ProxyServer"]
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

const settingPath = `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Internet Settings`

// getUserSID получает SID пользователя по имени
func getUserSID(username string) (string, error) {
	if username == "" {
		return "", nil
	}

	// Используем PowerShell для получения SID (совместимость с Windows 10 и Windows 11)
	psScript := fmt.Sprintf("(Get-WmiObject -Class Win32_UserAccount -Filter \"Name='%s'\").SID", username)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get SID for user %q: %w: %s", username, err, string(out))
	}

	// Парсим вывод: SID напрямую
	sid := strings.TrimSpace(string(out))
	if sid != "" {
		log.Printf("[SystemProxy] Got SID for user %s: %s", username, sid)
		return sid, nil
	}

	return "", fmt.Errorf("SID not found for user %q in PowerShell output", username)
}

// getSettingPath возвращает путь к настройкам прокси для пользователя
func getSettingPath(username string) (string, error) {
	if username == "" {
		return settingPath, nil
	}

	sid, err := getUserSID(username)
	if err != nil {
		return "", err
	}

	// Используем HKEY_USERS\{SID}\ вместо HKEY_CURRENT_USER
	return fmt.Sprintf(`HKEY_USERS\%s\Software\Microsoft\Windows\CurrentVersion\Internet Settings`, sid), nil
}

func set(key string, typ string, value string) error {
	return setForUser(key, typ, value, "")
}

func setForUser(key string, typ string, value string, username string) error {
	path, err := getSettingPath(username)
	if err != nil {
		return err
	}

	if username != "" {
		log.Printf("[SystemProxy] Setting %s=%s for user %s at %s", key, value, username, path)
	}

	_, err = sys.Command(`reg`, `add`, path, `/v`, key, `/t`, typ, `/d`, value, `/f`)
	if err != nil && username != "" {
		log.Printf("[SystemProxy] Failed to set %s for user %s: %v", key, username, err)
	}
	return err
}

func get(keys ...string) (map[string]string, error) {
	buf, err := sys.Command(`reg`, `query`, settingPath)
	if err != nil {
		return nil, err
	}
	return getFrom(buf, settingPath, keys...)
}

func del(key string) error {
	_, err := sys.Command(`reg`, `delete`, settingPath, `/v`, key, `/f`)
	return err
}

func strBool(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func useProxy(b bool) error {
	return useProxyForUser(b, "")
}

func useProxyForUser(b bool, username string) error {
	return setForUser("ProxyEnable", "REG_DWORD", strBool(b), username)
}

func getProxy() (bool, error) {
	m, err := get("ProxyEnable", "REG_DWORD")
	if err != nil {
		return false, err
	}
	i, err := strconv.ParseInt(m["ProxyEnable"], 0, 0)
	if err != nil {
		return false, err
	}
	return i != 0, nil
}

func getFrom(data string, path string, keys ...string) (map[string]string, error) {
	m := map[string]string{}
	index := strings.Index(data, path)
	if index == -1 {
		return m, nil
	}
	data = data[index+len(path):]
	reader := textproto.NewReader(bufio.NewReader(bytes.NewBufferString(data)))
	_, _ = reader.ReadLine()
	for len(m) != len(keys) {
		row, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if row == "" {
			break
		}
		row = strings.TrimSpace(row)
		s := strings.SplitN(row, "    ", 3)
		key := s[0]
		skip := true
		for _, k := range keys {
			if k == key {
				skip = false
				break
			}
		}
		if skip {
			continue
		}
		val := ""
		if len(s) == 3 {
			val = s[2]
		}
		m[key] = val
	}
	return m, nil
}
