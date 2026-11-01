package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Lock is held by a harness run for as long as it runs sandboxed agents. Runs share one sandbox temp
// folder and tidy what appears there, so a second run at the same time would move the first one's files.
type Lock struct{ f *os.File }

// Acquire takes the lock in dir, or says who holds it.
func Acquire(dir string) (*Lock, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, ".harness.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		holder, _ := os.ReadFile(path)
		f.Close()
		return nil, fmt.Errorf("another harness run is in progress (pid %s); runs share one sandbox temp folder, so they go one at a time",
			strings.TrimSpace(string(holder)))
	}
	if err := f.Truncate(0); err == nil {
		fmt.Fprintf(f, "%d\n", os.Getpid())
	}
	return &Lock{f}, nil
}

// Release gives the lock up.
func (l *Lock) Release() {
	if l != nil && l.f != nil {
		_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
		_ = l.f.Close()
	}
}
