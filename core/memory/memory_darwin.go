//go:build darwin

package memory

/*
#include <mach/mach.h>
#include <mach/mach_vm.h>
#include <stdlib.h>

kern_return_t get_task_for_pid(int pid, mach_port_t *task) {
    return task_for_pid(mach_task_self(), pid, task);
}

kern_return_t vm_read_mem(mach_port_t task, mach_vm_address_t addr, mach_vm_size_t size, void *buf, mach_vm_size_t *bytes_read) {
    return mach_vm_read_overwrite(task, addr, size, (mach_vm_address_t)buf, bytes_read);
}

kern_return_t vm_write_mem(mach_port_t task, mach_vm_address_t addr, void *data, mach_msg_type_number_t data_len) {
    return mach_vm_write(task, addr, (vm_offset_t)data, data_len);
}
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"math"
	"unsafe"
)

// DarwinMemory implements Memory using macOS Mach APIs.
type DarwinMemory struct{}

// New returns a DarwinMemory instance.
func New() Memory {
	return &DarwinMemory{}
}

func getTask(pid int) (C.mach_port_t, error) {
	var task C.mach_port_t
	kr := C.get_task_for_pid(C.int(pid), &task)
	if kr != C.KERN_SUCCESS {
		return 0, fmt.Errorf("task_for_pid(%d) failed: kern_return %d (try running as root)", pid, kr)
	}
	return task, nil
}

func (m *DarwinMemory) ReadBytes(pid int, addr uintptr, length int) ([]byte, error) {
	task, err := getTask(pid)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, length)
	var bytesRead C.mach_vm_size_t

	kr := C.vm_read_mem(task, C.mach_vm_address_t(addr), C.mach_vm_size_t(length),
		unsafe.Pointer(&buf[0]), &bytesRead)
	if kr != C.KERN_SUCCESS {
		return nil, fmt.Errorf("mach_vm_read_overwrite at 0x%x failed: kern_return %d", addr, kr)
	}

	return buf[:bytesRead], nil
}

func (m *DarwinMemory) ReadInt(pid int, addr uintptr) (int32, error) {
	b, err := m.ReadBytes(pid, addr, 4)
	if err != nil {
		return 0, err
	}
	return int32(binary.LittleEndian.Uint32(b)), nil
}

func (m *DarwinMemory) ReadInt64(pid int, addr uintptr) (int64, error) {
	b, err := m.ReadBytes(pid, addr, 8)
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(b)), nil
}

func (m *DarwinMemory) ReadFloat(pid int, addr uintptr) (float32, error) {
	b, err := m.ReadBytes(pid, addr, 4)
	if err != nil {
		return 0, err
	}
	bits := binary.LittleEndian.Uint32(b)
	return math.Float32frombits(bits), nil
}

func (m *DarwinMemory) WriteBytes(pid int, addr uintptr, data []byte) error {
	task, err := getTask(pid)
	if err != nil {
		return err
	}

	kr := C.vm_write_mem(task, C.mach_vm_address_t(addr),
		unsafe.Pointer(&data[0]), C.mach_msg_type_number_t(len(data)))
	if kr != C.KERN_SUCCESS {
		return fmt.Errorf("mach_vm_write at 0x%x failed: kern_return %d", addr, kr)
	}
	return nil
}

func (m *DarwinMemory) WriteInt(pid int, addr uintptr, val int32) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(val))
	return m.WriteBytes(pid, addr, buf)
}

func (m *DarwinMemory) WriteInt64(pid int, addr uintptr, val int64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(val))
	return m.WriteBytes(pid, addr, buf)
}

func (m *DarwinMemory) WriteFloat(pid int, addr uintptr, val float32) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, math.Float32bits(val))
	return m.WriteBytes(pid, addr, buf)
}

func (m *DarwinMemory) ReadableRegions(pid int) ([]MemRegion, error) {
	task, err := getTask(pid)
	if err != nil {
		return nil, err
	}

	var regions []MemRegion
	var addr C.mach_vm_address_t = 0
	var size C.mach_vm_size_t
	var depth C.natural_t = 1024

	for {
		var info C.vm_region_submap_info_data_64_t
		var infoCount C.mach_msg_type_number_t = C.VM_REGION_SUBMAP_INFO_COUNT_64

		kr := C.mach_vm_region_recurse(task, &addr, &size, &depth,
			(C.vm_region_recurse_info_t)(unsafe.Pointer(&info)),
			&infoCount)

		if kr != C.KERN_SUCCESS {
			break
		}

		// Only include regions with read+write permissions and not submaps
		if info.is_submap == 0 {
			perms := ""
			if info.protection&C.VM_PROT_READ != 0 {
				perms += "r"
			} else {
				perms += "-"
			}
			if info.protection&C.VM_PROT_WRITE != 0 {
				perms += "w"
			} else {
				perms += "-"
			}
			if info.protection&C.VM_PROT_EXECUTE != 0 {
				perms += "x"
			} else {
				perms += "-"
			}

			if info.protection&C.VM_PROT_READ != 0 {
				regions = append(regions, MemRegion{
					Start: uintptr(addr),
					Size:  uintptr(size),
					Perms: perms,
				})
			}
		}

		addr += size
	}

	return regions, nil
}
