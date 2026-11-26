package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// BrowserInfo records how the browser reference was produced.
type BrowserInfo struct {
	Exe    string `json:"exe"`
	Mode   string `json:"mode"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Bytes  int    `json:"pngBytes"`
}

// CaptureBrowser renders svgText in a real browser engine (headless
// Chrome/Edge, including Windows binaries through WSL interop) and stores a
// PNG reference at outPNG.
func CaptureBrowser(svgText, outPNG string, w, h int) (*BrowserInfo, error) {
	exe, mode := findBrowser()
	if exe == "" {
		return nil, errors.New("no Chrome/Edge/Chromium binary found; set CHROME_PATH")
	}

	var svgArg, shotArg, profileArg, shotLinux string
	var cleanup func()
	if mode == "wsl-win" {
		winBase := `/mnt/c/Windows/Temp/vectorlab-shot`
		_ = os.MkdirAll(winBase, 0o755)
		d, err := os.MkdirTemp(winBase, "job-*")
		if err != nil {
			return nil, err
		}
		cleanup = func() { _ = os.RemoveAll(d) }
		svgLinux := filepath.Join(d, "scene.svg")
		shotLinux = filepath.Join(d, "ref.png")
		if err := os.WriteFile(svgLinux, []byte(svgText), 0o644); err != nil {
			return nil, err
		}
		winDir := `C:\Windows\Temp\vectorlab-shot\` + filepath.Base(d)
		svgArg = `file:///C:/Windows/Temp/vectorlab-shot/` + filepath.Base(d) + `/scene.svg`
		shotArg = winDir + `\ref.png`
		profileArg = winDir + `\profile`
		defer cleanup()
	} else {
		d, err := os.MkdirTemp("", "vectorlab-shot-*")
		if err != nil {
			return nil, err
		}
		cleanup = func() { _ = os.RemoveAll(d) }
		svgPath := filepath.Join(d, "scene.svg")
		shotLinux = filepath.Join(d, "ref.png")
		if err := os.WriteFile(svgPath, []byte(svgText), 0o644); err != nil {
			return nil, err
		}
		svgArg = "file://" + svgPath
		shotArg = shotLinux
		profileArg = filepath.Join(d, "profile")
		defer cleanup()
	}

	args := []string{
		"--headless=new",
		"--disable-gpu",
		"--no-sandbox",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-extensions",
		"--disable-dev-shm-usage",
		"--force-device-scale-factor=1",
		"--hide-scrollbars",
		"--default-background-color=FFFFFFFF",
		"--window-size=" + itoa(w) + "," + itoa(h),
		"--user-data-dir=" + profileArg,
		"--screenshot=" + shotArg,
		svgArg,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Retry with legacy headless mode for older binaries.
		args[0] = "--headless"
		ctx2, cancel2 := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel2()
		cmd2 := exec.CommandContext(ctx2, exe, args...)
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return nil, errors.New("browser render failed: " + err.Error() +
				"; retry: " + err2.Error() + "; output: " + string(out) + " / " + string(out2))
		}
	}

	data, err := os.ReadFile(shotLinux)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(outPNG, data, 0o644); err != nil {
		return nil, err
	}
	cfgFile, err := os.Open(outPNG)
	if err != nil {
		return nil, err
	}
	cfg, err := pngDecodeConfig(cfgFile)
	cfgFile.Close()
	if err != nil {
		return nil, err
	}
	return &BrowserInfo{Exe: exe, Mode: mode, Width: cfg.Width, Height: cfg.Height, Bytes: len(data)}, nil
}

func findBrowser() (string, string) {
	if p := os.Getenv("CHROME_PATH"); p != "" {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, "native"
		}
	}
	winCandidates := []string{
		`/mnt/c/Program Files/Google/Chrome/Application/chrome.exe`,
		`/mnt/c/Program Files (x86)/Google/Chrome/Application/chrome.exe`,
		`/mnt/c/Program Files (x86)/Microsoft/Edge/Application/msedge.exe`,
		`/mnt/c/Program Files/Microsoft/Edge/Application/msedge.exe`,
	}
	for _, p := range winCandidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, "wsl-win"
		}
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome", "msedge"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, "native"
		}
	}
	return "", ""
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
