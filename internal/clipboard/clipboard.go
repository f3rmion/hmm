// Package clipboard provides cross-platform clipboard support.
package clipboard

import (
	"os/exec"
	"runtime"
	"strings"
)

// Write copies text to the system clipboard.
func Write(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, fall back to xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		}
	case "windows":
		cmd = exec.Command("cmd", "/c", "clip")
	default:
		// Try xclip as fallback
		cmd = exec.Command("xclip", "-selection", "clipboard")
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// Available checks if clipboard functionality is available.
func Available() bool {
	switch runtime.GOOS {
	case "darwin":
		_, err := exec.LookPath("pbcopy")
		return err == nil
	case "linux":
		if _, err := exec.LookPath("xclip"); err == nil {
			return true
		}
		_, err := exec.LookPath("xsel")
		return err == nil
	case "windows":
		return true // clip is always available on Windows
	default:
		return false
	}
}
