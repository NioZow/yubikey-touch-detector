//go:build linux

package detector

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/maximbaz/yubikey-touch-detector/notifier"
)

// attributeDevice finds a process that currently has the given hidraw device
// open. It is used for U2F/HMAC operations, where no unix socket connects the
// requester to the detector.
//
// When deleted is true the device node is expected to have disappeared (HMAC
// touch requests reset the HID interface), so file descriptors are matched
// against "<devicePath> (deleted)".
//
// The scan is best-effort: it only sees processes of the current user, so
// requests originating from privileged processes (e.g. sudo/pam-u2f) may not be
// attributed.
func attributeDevice(devicePath string, deleted bool) *notifier.Process {
	if devicePath == "" {
		return nil
	}

	var wantDev, wantIno uint64
	wantStat := false
	if !deleted {
		if info, err := os.Stat(devicePath); err == nil {
			if stat, ok := info.Sys().(*syscall.Stat_t); ok {
				wantDev = uint64(stat.Dev)
				wantIno = stat.Ino
				wantStat = true
			}
		}
	}

	self := os.Getpid()
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == self {
			continue
		}
		fdDir := filepath.Join("/proc", entry.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		for _, fd := range fds {
			fdPath := filepath.Join(fdDir, fd.Name())
			if !fdMatchesDevice(fdPath, devicePath, deleted, wantStat, wantDev, wantIno) {
				continue
			}
			if process := notifier.NewProcess(pid); process != nil {
				process.Source = "proc-scan"
				return process
			}
		}
	}
	return nil
}

func fdMatchesDevice(fdPath, devicePath string, deleted, wantStat bool, wantDev, wantIno uint64) bool {
	if deleted {
		link, err := os.Readlink(fdPath)
		return err == nil && strings.HasPrefix(link, devicePath)
	}
	if link, err := os.Readlink(fdPath); err == nil && link == devicePath {
		return true
	}
	if !wantStat {
		return false
	}
	info, err := os.Stat(fdPath)
	if err != nil {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && uint64(stat.Dev) == wantDev && uint64(stat.Ino) == wantIno
}
