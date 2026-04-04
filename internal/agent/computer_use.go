package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/aniclew/aniclew/internal/types"
)

// ComputerUseToolDefs returns tools for desktop/browser control.
func ComputerUseToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{
			Name:        "Screenshot",
			Description: "Take a screenshot of the current screen or a specific window. Returns the image for analysis.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"region": {"type": "string", "description": "Optional: 'full' (default), 'active' (active window only)"}
				}
			}`),
		},
		{
			Name:        "MouseClick",
			Description: "Click at a specific screen coordinate.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"x": {"type": "integer", "description": "X coordinate"},
					"y": {"type": "integer", "description": "Y coordinate"},
					"button": {"type": "string", "description": "'left' (default), 'right', 'double'"}
				},
				"required": ["x", "y"]
			}`),
		},
		{
			Name:        "TypeText",
			Description: "Type text at the current cursor position.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"text": {"type": "string", "description": "Text to type"},
					"key": {"type": "string", "description": "Special key: 'enter', 'tab', 'escape', 'backspace', 'ctrl+c', 'ctrl+v', etc."}
				}
			}`),
		},
		{
			Name:        "OpenApp",
			Description: "Open an application or URL on the desktop.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"target": {"type": "string", "description": "App name, file path, or URL to open"}
				},
				"required": ["target"]
			}`),
		},
		{
			Name:        "ListWindows",
			Description: "List all open windows/applications.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		{
			Name:        "FileManager",
			Description: "Manage files on desktop: copy, move, rename, delete, organize by pattern.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"action": {"type": "string", "description": "'copy', 'move', 'rename', 'delete', 'organize'"},
					"source": {"type": "string", "description": "Source path or glob pattern"},
					"destination": {"type": "string", "description": "Destination path"},
					"pattern": {"type": "string", "description": "For organize: group by 'extension', 'date', 'name'"}
				},
				"required": ["action", "source"]
			}`),
		},
		{
			Name:        "Clipboard",
			Description: "Read or write the system clipboard.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"action": {"type": "string", "description": "'read' or 'write'"},
					"text": {"type": "string", "description": "Text to write (for 'write' action)"}
				},
				"required": ["action"]
			}`),
		},
	}
}

// ExecuteComputerUseTool handles desktop control tools.
func ExecuteComputerUseTool(name string, input json.RawMessage, workDir string) (string, bool, bool) {
	switch name {
	case "Screenshot":
		r, e := executeScreenshot(input, workDir)
		return r, e, true
	case "MouseClick":
		r, e := executeMouseClick(input)
		return r, e, true
	case "TypeText":
		r, e := executeTypeText(input)
		return r, e, true
	case "OpenApp":
		r, e := executeOpenApp(input)
		return r, e, true
	case "ListWindows":
		r, e := executeListWindows()
		return r, e, true
	case "FileManager":
		r, e := executeFileManager(input, workDir)
		return r, e, true
	case "Clipboard":
		r, e := executeClipboard(input)
		return r, e, true
	default:
		return "", false, false
	}
}

// ── Screenshot ──

func executeScreenshot(input json.RawMessage, workDir string) (string, bool) {
	var args struct{ Region string `json:"region"` }
	json.Unmarshal(input, &args)

	tmpFile := workDir + "/.aniclew-screenshot.png"

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// PowerShell screenshot
		ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$screen = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
$bitmap = New-Object System.Drawing.Bitmap($screen.Width, $screen.Height)
$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
$graphics.CopyFromScreen(0, 0, 0, 0, $screen.Size)
$bitmap.Save('%s')
$graphics.Dispose()
$bitmap.Dispose()
`, strings.ReplaceAll(tmpFile, "/", "\\"))
		cmd = exec.Command("powershell", "-Command", ps)
	case "darwin":
		cmd = exec.Command("screencapture", "-x", tmpFile)
	default: // linux
		cmd = exec.Command("scrot", tmpFile)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("Screenshot failed: %v. Install required tools (scrot on Linux).", err), true
	}

	// Read and encode as base64
	data, err := readFileBytes(tmpFile)
	if err != nil {
		return fmt.Sprintf("Failed to read screenshot: %v", err), true
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	// Truncate for display (actual image would be sent via multimodal API)
	info := fmt.Sprintf("Screenshot captured: %s (%d KB). Base64 length: %d chars.",
		tmpFile, len(data)/1024, len(b64))

	return info, false
}

// ── Mouse Click (platform-specific) ──

func executeMouseClick(input json.RawMessage) (string, bool) {
	var args struct {
		X      int    `json:"x"`
		Y      int    `json:"y"`
		Button string `json:"button"`
	}
	json.Unmarshal(input, &args)
	if args.Button == "" {
		args.Button = "left"
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		ps := fmt.Sprintf(`
Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
public class Mouse {
    [DllImport("user32.dll")] public static extern bool SetCursorPos(int X, int Y);
    [DllImport("user32.dll")] public static extern void mouse_event(int dwFlags, int dx, int dy, int dwData, int dwExtraInfo);
}
"@
[Mouse]::SetCursorPos(%d, %d)
Start-Sleep -Milliseconds 50
[Mouse]::mouse_event(0x0002, 0, 0, 0, 0)
[Mouse]::mouse_event(0x0004, 0, 0, 0, 0)
`, args.X, args.Y)
		cmd = exec.Command("powershell", "-Command", ps)
	case "darwin":
		cmd = exec.Command("osascript", "-e",
			fmt.Sprintf(`tell application "System Events" to click at {%d, %d}`, args.X, args.Y))
	default:
		cmd = exec.Command("xdotool", "mousemove", fmt.Sprintf("%d", args.X), fmt.Sprintf("%d", args.Y), "click", "1")
	}

	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("Click failed: %v", err), true
	}
	return fmt.Sprintf("Clicked at (%d, %d) [%s]", args.X, args.Y, args.Button), false
}

// ── Type Text ──

func executeTypeText(input json.RawMessage) (string, bool) {
	var args struct {
		Text string `json:"text"`
		Key  string `json:"key"`
	}
	json.Unmarshal(input, &args)

	if args.Key != "" {
		return sendSpecialKey(args.Key)
	}

	if args.Text == "" {
		return "No text or key specified", true
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		ps := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.SendKeys]::SendWait('%s')
`, escapeSendKeys(args.Text))
		cmd = exec.Command("powershell", "-Command", ps)
	case "darwin":
		cmd = exec.Command("osascript", "-e",
			fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, args.Text))
	default:
		cmd = exec.Command("xdotool", "type", "--", args.Text)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("Type failed: %v", err), true
	}
	return fmt.Sprintf("Typed: %s", args.Text), false
}

// ── Open App/URL ──

func executeOpenApp(input json.RawMessage) (string, bool) {
	var args struct{ Target string `json:"target"` }
	json.Unmarshal(input, &args)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", args.Target)
	case "darwin":
		cmd = exec.Command("open", args.Target)
	default:
		cmd = exec.Command("xdg-open", args.Target)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Sprintf("Failed to open: %v", err), true
	}
	return fmt.Sprintf("Opened: %s", args.Target), false
}

// ── List Windows ──

func executeListWindows() (string, bool) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("powershell", "-Command",
			`Get-Process | Where-Object {$_.MainWindowTitle} | Format-Table Id,ProcessName,MainWindowTitle -AutoSize`)
	case "darwin":
		cmd = exec.Command("osascript", "-e",
			`tell application "System Events" to get name of every process whose visible is true`)
	default:
		cmd = exec.Command("wmctrl", "-l")
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("List windows failed: %v", err), true
	}
	return string(out), false
}

// ── File Manager ──

func executeFileManager(input json.RawMessage, workDir string) (string, bool) {
	var args struct {
		Action      string `json:"action"`
		Source      string `json:"source"`
		Destination string `json:"destination"`
		Pattern     string `json:"pattern"`
	}
	json.Unmarshal(input, &args)

	src := resolvePath(args.Source, workDir)
	dst := ""
	if args.Destination != "" {
		dst = resolvePath(args.Destination, workDir)
	}

	switch args.Action {
	case "copy":
		if dst == "" {
			return "Destination required for copy", true
		}
		cmd := exec.Command("cp", "-r", src, dst)
		if runtime.GOOS == "windows" {
			cmd = exec.Command("xcopy", src, dst, "/E", "/I", "/Y")
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Copy failed: %v\n%s", err, out), true
		}
		return fmt.Sprintf("Copied %s → %s", args.Source, args.Destination), false

	case "move":
		if dst == "" {
			return "Destination required for move", true
		}
		cmd := exec.Command("mv", src, dst)
		if runtime.GOOS == "windows" {
			cmd = exec.Command("move", src, dst)
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Move failed: %v\n%s", err, out), true
		}
		return fmt.Sprintf("Moved %s → %s", args.Source, args.Destination), false

	case "rename":
		if dst == "" {
			return "Destination (new name) required", true
		}
		cmd := exec.Command("mv", src, dst)
		if runtime.GOOS == "windows" {
			cmd = exec.Command("ren", src, dst)
		}
		if err := cmd.Run(); err != nil {
			return fmt.Sprintf("Rename failed: %v", err), true
		}
		return fmt.Sprintf("Renamed %s → %s", args.Source, args.Destination), false

	case "delete":
		// Safety check
		if src == "/" || src == "C:\\" || src == workDir {
			return "Blocked: cannot delete root or workspace directory", true
		}
		cmd := exec.Command("rm", "-r", src)
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/c", "rd", "/s", "/q", src)
		}
		if err := cmd.Run(); err != nil {
			return fmt.Sprintf("Delete failed: %v", err), true
		}
		return fmt.Sprintf("Deleted: %s", args.Source), false

	case "organize":
		return organizeFiles(src, args.Pattern)

	default:
		return fmt.Sprintf("Unknown action: %s", args.Action), true
	}
}

func organizeFiles(dir, pattern string) (string, bool) {
	// Use bash to organize files by extension
	script := fmt.Sprintf(`
cd "%s" && for f in *.*; do
  ext="${f##*.}"
  mkdir -p "$ext" 2>/dev/null
  mv "$f" "$ext/" 2>/dev/null
done && echo "Organized by extension"
`, dir)

	if pattern == "date" {
		script = fmt.Sprintf(`
cd "%s" && for f in *.*; do
  d=$(date -r "$f" +%%Y-%%m 2>/dev/null || echo "unknown")
  mkdir -p "$d" 2>/dev/null
  mv "$f" "$d/" 2>/dev/null
done && echo "Organized by date"
`, dir)
	}

	cmd := exec.Command("bash", "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Organize failed: %v\n%s", err, out), true
	}
	return strings.TrimSpace(string(out)), false
}

// ── Clipboard ──

func executeClipboard(input json.RawMessage) (string, bool) {
	var args struct {
		Action string `json:"action"`
		Text   string `json:"text"`
	}
	json.Unmarshal(input, &args)

	switch args.Action {
	case "read":
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("powershell", "-Command", "Get-Clipboard")
		case "darwin":
			cmd = exec.Command("pbpaste")
		default:
			cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Clipboard read failed: %v", err), true
		}
		return fmt.Sprintf("Clipboard content:\n%s", string(out)), false

	case "write":
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("powershell", "-Command", fmt.Sprintf("Set-Clipboard -Value '%s'", args.Text))
		case "darwin":
			cmd = exec.Command("bash", "-c", fmt.Sprintf("echo -n '%s' | pbcopy", args.Text))
		default:
			cmd = exec.Command("bash", "-c", fmt.Sprintf("echo -n '%s' | xclip -selection clipboard", args.Text))
		}
		if err := cmd.Run(); err != nil {
			return fmt.Sprintf("Clipboard write failed: %v", err), true
		}
		return fmt.Sprintf("Copied to clipboard: %s", truncateStr(args.Text, 100)), false

	default:
		return "Action must be 'read' or 'write'", true
	}
}

// ── Helpers ──

func readFileBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func sendSpecialKey(key string) (string, bool) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		keyMap := map[string]string{
			"enter": "{ENTER}", "tab": "{TAB}", "escape": "{ESC}",
			"backspace": "{BACKSPACE}", "delete": "{DELETE}",
			"ctrl+c": "^c", "ctrl+v": "^v", "ctrl+z": "^z", "ctrl+s": "^s",
		}
		winKey := keyMap[strings.ToLower(key)]
		if winKey == "" {
			winKey = "{" + strings.ToUpper(key) + "}"
		}
		cmd = exec.Command("powershell", "-Command",
			fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.SendKeys]::SendWait('%s')`, winKey))
	case "darwin":
		cmd = exec.Command("osascript", "-e",
			fmt.Sprintf(`tell application "System Events" to key code %s`, mapKeyMac(key)))
	default:
		cmd = exec.Command("xdotool", "key", key)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("Key press failed: %v", err), true
	}
	return fmt.Sprintf("Pressed: %s", key), false
}

func mapKeyMac(key string) string {
	m := map[string]string{
		"enter": "36", "tab": "48", "escape": "53",
		"backspace": "51", "delete": "117",
	}
	if v, ok := m[strings.ToLower(key)]; ok {
		return v
	}
	return "36" // default enter
}

func escapeSendKeys(text string) string {
	// Escape special SendKeys characters
	r := strings.NewReplacer(
		"+", "{+}", "^", "{^}", "%", "{%}",
		"~", "{~}", "(", "{(}", ")", "{)}",
		"{", "{{}", "}", "{}}", "[", "{[}", "]", "{]}",
	)
	return r.Replace(text)
}
