//go:build linux

package composer

import (
	"bytes"
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

// BecomeSubreaper marks the calling process as a "child subreaper".
// Descendants whose intermediate ancestors exit are reparented to this
// process instead of to PID 1, so every descendant remains reachable from
// os.Getpid() in the tree returned by getProcessTree, no matter how many
// intermediate parents have died.
//
// Call once at startup, before spawning any subprocesses. The flag is
// process-wide (not inherited by children) and only needs to be set on the
// topmost ancestor; orphans reparent to the nearest living ancestor
// subreaper. Requires Linux 3.4+.
func BecomeSubreaper() error {
	return unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0)
}

// getProcessTree returns a map from parent PID to child PIDs, built from
// /proc. Best-effort snapshot: processes may exit mid-scan. When
// BecomeSubreaper has been called at startup, every descendant of this
// process is reachable from os.Getpid() in the returned tree.
func getProcessTree() (map[int][]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	tree := make(map[int][]int)
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		ppid, ok := readPPID("/proc/" + entry.Name() + "/status")
		if !ok {
			continue
		}
		tree[ppid] = append(tree[ppid], pid)
	}
	return tree, nil
}

// readPPID extracts the PPid field from a /proc/[pid]/status file. Returns
// false if the file can't be read or the field is missing (process exited
// mid-scan, permission denied, etc).
func readPPID(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	prefix := []byte("PPid:")
	for len(data) > 0 {
		nl := bytes.IndexByte(data, '\n')
		var line []byte
		if nl < 0 {
			line, data = data, nil
		} else {
			line, data = data[:nl], data[nl+1:]
		}
		if !bytes.HasPrefix(line, prefix) {
			continue
		}
		fields := bytes.Fields(line)
		if len(fields) < 2 {
			return 0, false
		}
		ppid, err := strconv.Atoi(string(fields[1]))
		if err != nil {
			return 0, false
		}
		return ppid, true
	}
	return 0, false
}
