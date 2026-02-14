package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyBinary(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcPath := filepath.Join(srcDir, "sg")
	content := []byte("fake-binary-content")
	if err := os.WriteFile(srcPath, content, 0755); err != nil {
		t.Fatalf("write source: %v", err)
	}

	dstPath := filepath.Join(dstDir, "sg")
	if err := copyBinary(srcPath, dstPath); err != nil {
		t.Fatalf("copyBinary: %v", err)
	}

	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}

	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("stat dest: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("permissions: got %o, want 0755", info.Mode().Perm())
	}
}

func TestCopyBinaryCreatesDir(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "sg")
	os.WriteFile(srcPath, []byte("binary"), 0755)

	dstDir := filepath.Join(t.TempDir(), "nested", "dir")
	dstPath := filepath.Join(dstDir, "sg")

	if err := copyBinary(srcPath, dstPath); err != nil {
		t.Fatalf("copyBinary to nested dir: %v", err)
	}
	if _, err := os.Stat(dstPath); err != nil {
		t.Fatalf("dest not created: %v", err)
	}
}

func TestIsSameFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sg")
	os.WriteFile(path, []byte("binary"), 0755)

	if !isSameFile(path, path) {
		t.Error("same path should be same file")
	}

	linkPath := filepath.Join(dir, "sg-link")
	if err := os.Link(path, linkPath); err != nil {
		t.Skipf("hardlink not supported: %v", err)
	}
	if !isSameFile(path, linkPath) {
		t.Error("hardlinked paths should be same file")
	}

	otherPath := filepath.Join(dir, "other")
	os.WriteFile(otherPath, []byte("binary"), 0755)
	if isSameFile(path, otherPath) {
		t.Error("different files should not be same file")
	}

	if isSameFile(path, filepath.Join(dir, "nonexistent")) {
		t.Error("nonexistent file should not match")
	}
}

func TestCopyBinaryHardlinkSafe(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "sg-src")
	content := []byte("important-binary-data")
	os.WriteFile(srcPath, content, 0755)

	dstPath := filepath.Join(dir, "sg-dst")
	if err := os.Link(srcPath, dstPath); err != nil {
		t.Skipf("hardlink not supported: %v", err)
	}

	if !isSameFile(srcPath, dstPath) {
		t.Fatal("expected same file via hardlink")
	}

	if err := copyBinary(srcPath, dstPath); err != nil {
		t.Fatalf("copyBinary on hardlinked dst: %v", err)
	}

	got, _ := os.ReadFile(srcPath)
	if string(got) != string(content) {
		t.Errorf("source corrupted after copy to hardlinked dst: got %q", got)
	}

	dstGot, _ := os.ReadFile(dstPath)
	if string(dstGot) != string(content) {
		t.Errorf("dest content wrong: got %q", dstGot)
	}
}

func TestCopyBinaryIsAtomic(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcPath := filepath.Join(srcDir, "sg")
	os.WriteFile(srcPath, []byte("new-version"), 0755)

	dstPath := filepath.Join(dstDir, "sg")
	os.WriteFile(dstPath, []byte("old-version"), 0755)

	if err := copyBinary(srcPath, dstPath); err != nil {
		t.Fatalf("copyBinary overwrite: %v", err)
	}

	got, _ := os.ReadFile(dstPath)
	if string(got) != "new-version" {
		t.Errorf("overwrite failed: got %q", got)
	}
}

func TestIsInPATH(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/usr/local/bin:/home/user/go/bin")
	if !isInPATH("/usr/local/bin") {
		t.Error("expected /usr/local/bin in PATH")
	}
	if isInPATH("/nope") {
		t.Error("expected /nope not in PATH")
	}
}

func TestFindTargetDirPicksWritablePATHEntry(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	t.Setenv("PATH", dirA+":"+dirB)
	t.Setenv("GOPATH", "/nonexistent")
	t.Setenv("HOME", "/nonexistent")

	got := findTargetDir()
	if got != dirA {
		t.Errorf("expected first writable PATH dir %s, got %s", dirA, got)
	}
}

func TestFindTargetDirPrefersKnownLocations(t *testing.T) {
	goBase := t.TempDir()
	goBin := filepath.Join(goBase, "bin")
	os.MkdirAll(goBin, 0755)
	otherDir := t.TempDir()
	t.Setenv("PATH", otherDir+":"+goBin)
	t.Setenv("GOPATH", goBase)
	t.Setenv("HOME", "/nonexistent")

	got := findTargetDir()
	if got != goBin {
		t.Errorf("expected preferred dir %s, got %s", goBin, got)
	}
}

func TestFindTargetDirReturnsEmptyWhenNothingWritable(t *testing.T) {
	t.Setenv("PATH", "/nonexistent-a:/nonexistent-b")
	t.Setenv("GOPATH", "/nonexistent")
	t.Setenv("HOME", "/nonexistent")

	got := findTargetDir()
	if got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestIsWritableDir(t *testing.T) {
	writable := t.TempDir()
	if !isWritableDir(writable) {
		t.Error("temp dir should be writable")
	}
	if isWritableDir("/nonexistent-dir") {
		t.Error("nonexistent dir should not be writable")
	}
}
