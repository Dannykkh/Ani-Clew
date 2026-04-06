package tray

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
)

// TrayConfig holds tray app settings.
type TrayConfig struct {
	Port    int
	Model   string
	Provider string
}

// ShowNotification sends a system notification (cross-platform).
func ShowNotification(title, message string) {
	switch runtime.GOOS {
	case "windows":
		// PowerShell toast notification
		ps := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType=WindowsRuntime] > $null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$text = $template.GetElementsByTagName("text")
$text.Item(0).AppendChild($template.CreateTextNode("%s")) > $null
$text.Item(1).AppendChild($template.CreateTextNode("%s")) > $null
$toast = [Windows.UI.Notifications.ToastNotification]::new($template)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("AniClew").Show($toast)
`, title, message)
		cmd := exec.Command("powershell", "-Command", ps)
		cmd.Run()

	case "darwin":
		// macOS notification
		script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
		exec.Command("osascript", "-e", script).Run()

	case "linux":
		// Linux notification
		exec.Command("notify-send", title, message).Run()
	}

	log.Printf("[Tray] Notification: %s — %s", title, message)
}

// OpenBrowser opens the default browser to AniClew's URL.
func OpenBrowser(port int) {
	url := fmt.Sprintf("http://localhost:%d/app", port)
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		log.Printf("[Tray] Failed to open browser: %v", err)
	}
}
