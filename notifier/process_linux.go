//go:build linux

package notifier

import (
	"os"
	"os/user"
	"strconv"
	"strings"
)

// NewProcess collects metadata about the process with the given PID by reading
// /proc. It returns nil when the PID is invalid or the procfs entry has already
// disappeared.
func NewProcess(pid int) *Process {
	if pid <= 0 {
		return nil
	}
	base := "/proc/" + strconv.Itoa(pid)
	process := &Process{PID: pid, Source: "proc"}

	if status, err := os.ReadFile(base + "/status"); err == nil {
		parseProcStatus(string(status), process)
	}
	if comm, err := os.ReadFile(base + "/comm"); err == nil {
		process.Comm = strings.TrimSpace(string(comm))
	}
	if exe, err := os.Readlink(base + "/exe"); err == nil {
		process.Exe = exe
	}
	if cwd, err := os.Readlink(base + "/cwd"); err == nil {
		process.Cwd = cwd
	}
	if raw, err := os.ReadFile(base + "/cmdline"); err == nil {
		process.Cmdline = splitNullList(string(raw))
	}

	if process.Comm == "" && process.Exe == "" && len(process.Cmdline) == 0 {
		return nil
	}

	if process.User == "" && process.UID != 0 {
		if account, err := user.LookupId(strconv.FormatUint(uint64(process.UID), 10)); err == nil {
			process.User = account.Username
		}
	}
	return process
}

func parseProcStatus(status string, process *Process) {
	for _, line := range strings.Split(status, "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		fields := strings.Fields(value)
		switch key {
		case "Name":
			if process.Comm == "" && len(fields) > 0 {
				process.Comm = fields[0]
			}
		case "PPid":
			if len(fields) > 0 {
				if pid, err := strconv.Atoi(fields[0]); err == nil {
					process.PPID = pid
				}
			}
		case "Uid":
			if len(fields) > 0 {
				if uid, err := strconv.ParseUint(fields[0], 10, 32); err == nil {
					process.UID = uint32(uid)
				}
			}
		case "Gid":
			if len(fields) > 0 {
				if gid, err := strconv.ParseUint(fields[0], 10, 32); err == nil {
					process.GID = uint32(gid)
				}
			}
		}
	}
}

func splitNullList(value string) []string {
	var result []string
	for _, item := range strings.Split(value, "\x00") {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
