//go:build darwin

package composer

import (
	"golang.org/x/sys/unix"
)

// BecomeSubreaper is a no-op on macOS. Darwin has no equivalent of
// Linux's PR_SET_CHILD_SUBREAPER, so orphaned descendants are reparented
// to launchd (PID 1) and lineage tracking through intermediate parent
// deaths is not possible on this platform. Provided for API symmetry
// with the Linux build so cross-platform callers don't need build tags.
func BecomeSubreaper() error {
	return nil
}

// getProcessTree returns a map from parent PID to child PIDs, built from
// the kernel's process table via sysctl(KERN_PROC_ALL). Best-effort
// snapshot: reparented orphans appear under their current parent
// (typically launchd) rather than their original ancestor. See
// BecomeSubreaper for why this is unavoidable on Darwin.
func getProcessTree() (map[int][]int, error) {
	kprocs, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return nil, err
	}

	tree := make(map[int][]int)
	for i := range kprocs {
		pid := int(kprocs[i].Proc.P_pid)
		ppid := int(kprocs[i].Eproc.Ppid)
		tree[ppid] = append(tree[ppid], pid)
	}
	return tree, nil
}
