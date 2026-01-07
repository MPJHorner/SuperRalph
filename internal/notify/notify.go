package notify

import (
	"os/exec"
	"runtime"
)

// Send sends a system notification
func Send(title, message string) error {
	switch runtime.GOOS {
	case "darwin":
		return sendMacOS(title, message)
	case "linux":
		return sendLinux(title, message)
	default:
		// Silently ignore unsupported platforms
		return nil
	}
}

func sendMacOS(title, message string) error {
	script := `display notification "` + message + `" with title "` + title + `"`
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

func sendLinux(title, message string) error {
	cmd := exec.Command("notify-send", title, message)
	return cmd.Run()
}

// SendSuccess sends a success notification
func SendSuccess(message string) error {
	return Send("SuperRalph", "✓ "+message)
}

// SendError sends an error notification
func SendError(message string) error {
	return Send("SuperRalph", "✗ "+message)
}

// SendComplete sends a completion notification
func SendComplete(iterations int) error {
	return Send("SuperRalph", "PRD complete! Finished in "+string(rune(iterations))+" iterations")
}
