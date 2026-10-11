//go:build !linux

package notifier

// NewProcess is a no-op on platforms without procfs. Process attribution is
// currently only implemented on Linux.
func NewProcess(int) *Process {
	return nil
}
