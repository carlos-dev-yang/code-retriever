package embedlock

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"
)

// Acquire uses an OS root handle so a `.cidx` or lock-file symlink swap cannot
// redirect the process outside its repository root.
func Acquire(ctx context.Context, path string) (func(), error) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	dir, err := root.Open(".cidx")
	if err != nil {
		return nil, err
	}
	info, err := dir.Stat()
	dir.Close()
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("embed lock state directory invalid")
	}
	f, err := root.OpenFile(".cidx/embed.lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		f.Close()
		return nil, fmt.Errorf("embed lock path invalid")
	}
	for {
		if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
			return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
		}
		if err != syscall.EWOULDBLOCK {
			f.Close()
			return nil, err
		}
		select {
		case <-ctx.Done():
			f.Close()
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}
