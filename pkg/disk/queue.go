package disk

import "github.com/hwameistor/local-disk-manager/pkg/disk/manager"

// Push
func (ctr Controller) Push(disk manager.Event) {
	ctr.diskQueue <- disk
}

// Pop
func (ctr Controller) Pop() manager.Event {
	return <-ctr.diskQueue
}
