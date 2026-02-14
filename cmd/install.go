package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func NewInstallCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install sg binary into your PATH",
		Long:  "Copies the running sg binary to a directory in your PATH, making it available system-wide.",
		RunE: func(c *cobra.Command, args []string) error {
			return runInstall(dir)
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "target directory (default: auto-detect from PATH)")
	return cmd
}

func runInstall(dir string) error {
	srcPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find current binary: %w", err)
	}
	srcPath, err = filepath.EvalSymlinks(srcPath)
	if err != nil {
		return fmt.Errorf("cannot resolve binary path: %w", err)
	}

	if dir == "" {
		dir = findTargetDir()
		if dir == "" {
			return fmt.Errorf("no suitable directory found in PATH; use --dir to specify one")
		}
	}

	dstPath := filepath.Join(dir, "sg")
	if isSameFile(srcPath, dstPath) {
		fmt.Printf("sg is already installed at %s\n", dstPath)
		return nil
	}

	if err := copyBinary(srcPath, dstPath); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	fmt.Printf("Installed sg to %s\n", dstPath)
	if !isInPATH(dir) {
		fmt.Printf("Warning: %s is not in your PATH\n", dir)
	}
	return nil
}

func isSameFile(a, b string) bool {
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	if errA != nil || errB != nil {
		return false
	}
	return os.SameFile(infoA, infoB)
}

func findTargetDir() string {
	home, _ := os.UserHomeDir()
	preferred := map[string]int{
		"/usr/local/bin":                    0,
		"/opt/homebrew/bin":                 1,
		filepath.Join(home, ".local", "bin"): 2,
	}
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		preferred[filepath.Join(gopath, "bin")] = 3
	} else if home != "" {
		preferred[filepath.Join(home, "go", "bin")] = 3
	}

	bestDir := ""
	bestPriority := len(preferred) + 1

	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		if !isWritableDir(dir) {
			continue
		}
		if p, ok := preferred[dir]; ok && p < bestPriority {
			bestDir = dir
			bestPriority = p
			continue
		}
		if bestDir == "" {
			bestDir = dir
		}
	}
	return bestDir
}

func isWritableDir(dir string) bool {
	tmp, err := os.CreateTemp(dir, ".sg-probe-*")
	if err != nil {
		return false
	}
	name := tmp.Name()
	tmp.Close()
	os.Remove(name)
	return true
}

func isInPATH(dir string) bool {
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == dir {
			return true
		}
	}
	return false
}

func copyBinary(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".sg-install-*")
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			return fmt.Errorf("permission denied — try: sudo sg install --dir %s", filepath.Dir(dst))
		}
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("copy binary: %w", err)
	}
	if err := tmp.Chmod(0755); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("set permissions: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("move binary into place: %w", err)
	}
	return nil
}
