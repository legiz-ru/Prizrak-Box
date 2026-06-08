package sys

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

func Command(name string, arg ...string) (string, error) {
	c := exec.Command(name, arg...)
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%q: %w: %q", strings.Join(append([]string{name}, arg...), " "), err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// CommandAsUser выполняет команду от имени указанного пользователя
// Используется для установки системного прокси для конкретного пользователя
func CommandAsUser(username string, name string, arg ...string) (string, error) {
	// Если username пустой или текущий пользователь уже нужный, выполняем напрямую
	if username == "" {
		return Command(name, arg...)
	}

	currentUser, err := user.Current()
	if err == nil && currentUser.Username == username {
		return Command(name, arg...)
	}

	// Получаем информацию о пользователе
	targetUser, err := user.Lookup(username)
	if err != nil {
		return "", fmt.Errorf("user %q not found: %w", username, err)
	}

	// Получаем UID пользователя для запуска D-Bus сессии
	uid, err := strconv.Atoi(targetUser.Uid)
	if err != nil {
		return "", fmt.Errorf("invalid UID for user %q: %w", username, err)
	}

	// Определяем DBUS_SESSION_BUS_ADDRESS для пользователя
	// Обычно это unix:path=/run/user/{UID}/bus
	dbusAddr := fmt.Sprintf("unix:path=/run/user/%d/bus", uid)

	// Пробуем использовать runuser (не требует пароля от root) или sudo
	// runuser доступен по умолчанию в большинстве дистрибутивов
	var cmdName string
	var cmdArgs []string

	// Проверяем, запущены ли мы от root
	if currentUser != nil && (currentUser.Username == "root" || currentUser.Uid == "0") {
		// От root используем runuser (не требует пароля)
		cmdName = "runuser"
		cmdArgs = []string{
			"-u", username,
			"--",
			"env",
			"DBUS_SESSION_BUS_ADDRESS=" + dbusAddr,
			"HOME=" + targetUser.HomeDir,
			"USER=" + username,
			name,
		}
	} else {
		// Не от root - используем sudo с -n (может потребовать настройки sudoers)
		cmdName = "sudo"
		cmdArgs = []string{
			"-n", // non-interactive mode
			"-u", username,
			"env",
			"DBUS_SESSION_BUS_ADDRESS=" + dbusAddr,
			"HOME=" + targetUser.HomeDir,
			"USER=" + username,
			name,
		}
	}

	cmdArgs = append(cmdArgs, arg...)

	c := exec.Command(cmdName, cmdArgs...)
	// Наследуем переменные окружения
	c.Env = os.Environ()
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s as %q: %w: %q", strings.Join(append([]string{cmdName, "-u", username, name}, arg...), " "), username, err, out)
	}
	return strings.TrimSpace(string(out)), nil
}
