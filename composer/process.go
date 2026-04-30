package composer

import (
	"errors"
	"syscall"
)

// collectDescendants returns all process IDs for processes spawned under the root process
func (c *Composer) collectDescendants(root int) (out []int) {
	tree, err := getProcessTree()
	if err != nil {
		c.debug("getProcessTree failed: %+v", err)
		return nil
	}

	var walk func(int)
	walk = func(pid int) {
		for _, child := range tree[pid] {
			walk(child)
			out = append(out, child)
		}
	}
	walk(root)

	return out
}

// signalAll kills all descendant processes spawned under the root process before killing the root process
func (c *Composer) signalAll(descendants []int, root int, sig syscall.Signal) {
	for _, pid := range descendants {
		if err := syscall.Kill(pid, sig); err != nil && !errors.Is(err, syscall.ESRCH) {
			c.debug("failed to kill descendant PID %d (%d): %v", pid, sig, err)
		}
	}

	if err := syscall.Kill(root, sig); err != nil && !errors.Is(err, syscall.ESRCH) {
		c.debug("failed to kill root PID %d (%d): %v", root, sig, err)
	}
}

// reapDescendants reap all descendant processes spawned under the root process that may have been re-parented to us so
// they don't linger as zombies
func (c *Composer) reapDescendants(descendants []int) {
	for _, dpid := range descendants {
		for {
			_, err := syscall.Wait4(dpid, nil, 0, nil)
			if errors.Is(err, syscall.EINTR) {
				continue
			}
			break
		}
	}
}
