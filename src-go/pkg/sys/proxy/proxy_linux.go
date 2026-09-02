package sys

import (
	"bytes"
	"fmt"
	sys "github.com/legiz-ru/prizrak-box/pkg/sys/cmd"
	"log"
	"strconv"
	"strings"
)

func OffAll() error {
	return OffAllForUser("")
}

func OffAllForUser(username string) error {
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
	buf := bytes.NewBuffer(nil)
	buf.WriteString("[ ")
	for i, item := range ignores {
		if item == "" {
			continue
		}
		buf.WriteByte('\'')
		buf.WriteString(item)
		buf.WriteByte('\'')
		if len(ignores)-1 != i {
			buf.WriteString(", ")
		}
	}
	buf.WriteString(" ]")
	return set("", "ignore-hosts", buf.String())
}

func ClearIgnore() error {
	return set("", "ignore-hosts", "[]")
}

func GetIgnore() ([]string, error) {
	data, err := get("", "ignore-hosts")
	if err != nil {
		return nil, err
	}
	data = strings.TrimPrefix(data, "@as")
	data = strings.TrimSpace(data)
	data = strings.TrimPrefix(data, "[")
	data = strings.TrimSuffix(data, "]")
	data = strings.TrimSpace(data)
	if data == "" {
		return []string{}, nil
	}
	ignores := strings.Split(data, ",")
	for i := range ignores {
		item := ignores[i]
		item = strings.TrimSpace(item)
		item = strings.Trim(item, "'")
		ignores[i] = item
	}
	return ignores, nil
}

func OnHttps(addr Addr) error {
	return OnHttpsForUser(addr, "")
}

func OnHttpsForUser(addr Addr, username string) error {
	err := setForUser("https", "host", addr.Host, username)
	if err != nil {
		return err
	}
	err = setForUser("https", "port", strconv.Itoa(addr.Port), username)
	if err != nil {
		return err
	}
	err = setForUser("", "mode", "manual", username)
	if err != nil {
		return err
	}
	return nil
}

func OffHttps() error {
	return OffHttpsForUser("")
}

func OffHttpsForUser(username string) error {
	err := resetForUser("", "mode", username)
	if err != nil {
		return err
	}
	err = resetForUser("https", "host", username)
	if err != nil {
		return err
	}
	err = resetForUser("https", "port", username)
	if err != nil {
		return err
	}
	return nil
}

func GetHttps() (*Addr, error) {
	mode, err := get("", "mode")
	if err != nil {
		return nil, err
	}
	mode = strings.Trim(mode, "'")
	if mode != "manual" {
		return nil, nil
	}
	host, err := get("https", "host")
	if err != nil {
		return nil, err
	}
	port, err := get("https", "port")
	if err != nil {
		return nil, err
	}
	return ParseAddrPtr(fmt.Sprintf("%s:%s", host, port)), nil
}

func OnHttp(addr Addr) error {
	return OnHttpForUser(addr, "")
}

func OnHttpForUser(addr Addr, username string) error {
	err := setForUser("http", "host", addr.Host, username)
	if err != nil {
		return err
	}
	err = setForUser("http", "port", strconv.Itoa(addr.Port), username)
	if err != nil {
		return err
	}
	err = setForUser("", "mode", "manual", username)
	if err != nil {
		return err
	}
	return nil
}

func OffHttp() error {
	return OffHttpForUser("")
}

func OffHttpForUser(username string) error {
	err := resetForUser("", "mode", username)
	if err != nil {
		return err
	}
	err = resetForUser("http", "host", username)
	if err != nil {
		return err
	}
	err = resetForUser("http", "port", username)
	if err != nil {
		return err
	}
	return nil
}

func GetHttp() (*Addr, error) {
	mode, err := get("", "mode")
	if err != nil {
		return nil, err
	}
	if mode != "manual" {
		return nil, nil
	}
	host, err := get("http", "host")
	if err != nil {
		return nil, err
	}
	port, err := get("http", "port")
	if err != nil {
		return nil, err
	}
	return ParseAddrPtr(fmt.Sprintf("%s:%s", host, port)), nil
}

func OnSocks(addr Addr) error {
	return OnSocksForUser(addr, "")
}

func OnSocksForUser(addr Addr, username string) error {
	err := setForUser("socks", "host", addr.Host, username)
	if err != nil {
		return err
	}
	err = setForUser("socks", "port", strconv.Itoa(addr.Port), username)
	if err != nil {
		return err
	}
	err = setForUser("", "mode", "manual", username)
	if err != nil {
		return err
	}
	return nil
}

func OffSocks() error {
	return OffSocksForUser("")
}

func OffSocksForUser(username string) error {
	err := resetForUser("", "mode", username)
	if err != nil {
		return err
	}
	err = resetForUser("socks", "host", username)
	if err != nil {
		return err
	}
	err = resetForUser("socks", "port", username)
	if err != nil {
		return err
	}
	return nil
}

func GetSocks() (*Addr, error) {
	mode, err := get("", "mode")
	if err != nil {
		return nil, err
	}
	if mode != "manual" {
		return nil, nil
	}
	host, err := get("socks", "host")
	if err != nil {
		return nil, err
	}
	port, err := get("socks", "port")
	if err != nil {
		return nil, err
	}
	return ParseAddrPtr(fmt.Sprintf("%s:%s", host, port)), nil
}

const scheme = "org.gnome.system.proxy"

func reset(sub, key string) error {
	return resetForUser(sub, key, "")
}

func get(sub, key string) (string, error) {
	return getForUser(sub, key, "")
}

func set(sub, key string, val string) error {
	return setForUser(sub, key, val, "")
}

func resetForUser(sub, key string, username string) error {
	scheme := scheme
	if sub != "" {
		scheme = scheme + "." + sub
	}
	_, err := sys.CommandAsUser(username, "gsettings", "reset", scheme, key)
	return err
}

func getForUser(sub, key string, username string) (string, error) {
	scheme := scheme
	if sub != "" {
		scheme = scheme + "." + sub
	}
	out, err := sys.CommandAsUser(username, "gsettings", "get", scheme, key)
	if err != nil {
		return "", err
	}
	out = strings.Trim(out, "'")
	return out, nil
}

func setForUser(sub, key string, val string, username string) error {
	scheme := scheme
	if sub != "" {
		scheme = scheme + "." + sub
	}

	if username != "" {
		log.Printf("[SystemProxy] Setting %s.%s=%s for user %s", scheme, key, val, username)
	}

	_, err := sys.CommandAsUser(username, "gsettings", "set", scheme, key, val)
	if err != nil {
		log.Printf("[SystemProxy] Failed to set %s.%s for user %s: %v", scheme, key, username, err)
	}
	return err
}
