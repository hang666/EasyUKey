package confirmation

import (
	"os/exec"
	"runtime"

	"github.com/hang666/EasyUKey/shared/pkg/logger"
)

// OpenBrowser 打开浏览器
func OpenBrowser(url string) error {
	var cmd string
	var args []string

	logger.Logger.Info("如没有自动打开浏览器，请手动访问链接进行操作", "url", url)

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // linux
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}
