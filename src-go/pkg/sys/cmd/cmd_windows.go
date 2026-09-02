package sys

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

func Command(name string, arg ...string) (string, error) {
	c := exec.Command(name, arg...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%q: %w: %q", strings.Join(append([]string{name}, arg...), " "), err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// CommandAsUser выполняет команду от имени указанного пользователя
// На Windows username игнорируется, так как реестр автоматически работает с текущим пользователем
func CommandAsUser(username string, name string, arg ...string) (string, error) {
	return Command(name, arg...)
}
