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
	if srcPath == dstPath {
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

func findTargetDir() string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		"/usr/local/bin",
		filepath.Join(home, ".local", "bin"),
	}
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		candidates = append(candidates, filepath.Join(gopath, "bin"))
	} else if home != "" {
		candidates = append(candidates, filepath.Join(home, "go", "bin"))
	}

	for _, c := range candidates {
		info, err := os.Stat(c)
		if err == nil && info.IsDir() && isInPATH(c) {
			return c
		}
	}
	return ""
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

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			return fmt.Errorf("permission denied — try: sudo sg install --dir %s", filepath.Dir(dst))
		}
		return fmt.Errorf("create target: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}
	return nil
}
