package sys

import (
	"fmt"
	"os/exec"
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
// На macOS username игнорируется, так как networksetup автоматически работает с текущим пользователем
func CommandAsUser(username string, name string, arg ...string) (string, error) {
	return Command(name, arg...)
}
