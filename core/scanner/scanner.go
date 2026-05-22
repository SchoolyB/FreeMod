package scanner

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"

	"github.com/freemod/freemod/core/memory"
)

// maxRegionSize is the largest single region we will read in one scan pass.
// Regions larger than this (e.g. JVM heap, graphics buffers) are skipped —
// game state values are never stored in multi-hundred-MB monolithic blobs.
const maxRegionSize = 64 * 1024 * 1024 // 64 MB

// ScanForInt scans all readable memory regions of the process for the given
// int32 value and returns the addresses where it is found.
func ScanForInt(mem memory.Memory, pid int, value int32) ([]uintptr, error) {
	regions, err := mem.ReadableRegions(pid)
	if err != nil {
		return nil, fmt.Errorf("enumerating regions: %w", err)
	}

	target := make([]byte, 4)
	binary.LittleEndian.PutUint32(target, uint32(value))

	var matches []uintptr

	for _, region := range regions {
		if region.Size < 4 || region.Size > maxRegionSize {
			continue
		}

		data, err := mem.ReadBytes(pid, region.Start, int(region.Size))
		if err != nil {
			// Silently skip unreadable regions (common for guard pages, etc.)
			continue
		}

		for i := 0; i <= len(data)-4; i++ {
			if data[i] == target[0] &&
				data[i+1] == target[1] &&
				data[i+2] == target[2] &&
				data[i+3] == target[3] {
				matches = append(matches, region.Start+uintptr(i))
			}
		}
	}

	return matches, nil
}

// batchRead reads values at a list of addresses using as few syscalls as possible.
// It sorts addresses and groups ones within batchSpan of each other into a single
// region read, then extracts values from the buffer.
const batchSpan = 4096 // group addresses within 4 KB into one read

func batchReadInt32(mem memory.Memory, pid int, addrs []uintptr) map[uintptr]int32 {
	if len(addrs) == 0 {
		return nil
	}
	sorted := make([]uintptr, len(addrs))
	copy(sorted, addrs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	out := make(map[uintptr]int32, len(addrs))
	i := 0
	for i < len(sorted) {
		base := sorted[i]
		end := base + 4
		j := i + 1
		for j < len(sorted) && sorted[j]+4-base <= batchSpan {
			if sorted[j]+4 > end {
				end = sorted[j] + 4
			}
			j++
		}
		size := int(end - base)
		data, err := mem.ReadBytes(pid, base, size)
		if err != nil {
			i = j
			continue
		}
		for k := i; k < j; k++ {
			off := int(sorted[k] - base)
			if off+4 <= len(data) {
				out[sorted[k]] = int32(binary.LittleEndian.Uint32(data[off : off+4]))
			}
		}
		i = j
	}
	return out
}

// NarrowScan filters a previous address list to only those whose current
// int32 value matches newValue.
func NarrowScan(mem memory.Memory, pid int, addrs []uintptr, newValue int32) ([]uintptr, error) {
	vals := batchReadInt32(mem, pid, addrs)
	var matches []uintptr
	for _, addr := range addrs {
		if v, ok := vals[addr]; ok && v == newValue {
			matches = append(matches, addr)
		}
	}
	return matches, nil
}

// ScanForFloat32 scans all readable memory regions for the given float32 value.
func ScanForFloat32(mem memory.Memory, pid int, value float32) ([]uintptr, error) {
	regions, err := mem.ReadableRegions(pid)
	if err != nil {
		return nil, fmt.Errorf("enumerating regions: %w", err)
	}

	target := make([]byte, 4)
	binary.LittleEndian.PutUint32(target, math.Float32bits(value))

	var matches []uintptr
	for _, region := range regions {
		if region.Size < 4 || region.Size > maxRegionSize {
			continue
		}
		data, err := mem.ReadBytes(pid, region.Start, int(region.Size))
		if err != nil {
			continue
		}
		for i := 0; i <= len(data)-4; i++ {
			if data[i] == target[0] && data[i+1] == target[1] &&
				data[i+2] == target[2] && data[i+3] == target[3] {
				matches = append(matches, region.Start+uintptr(i))
			}
		}
	}
	return matches, nil
}

// NarrowScanFloat32 filters a previous address list to those whose current
// float32 value (by exact bit pattern) matches newValue.
func NarrowScanFloat32(mem memory.Memory, pid int, addrs []uintptr, newValue float32) ([]uintptr, error) {
	bits := math.Float32bits(newValue)
	var matches []uintptr
	for _, addr := range addrs {
		data, err := mem.ReadBytes(pid, addr, 4)
		if err != nil {
			continue
		}
		if binary.LittleEndian.Uint32(data) == bits {
			matches = append(matches, addr)
		}
	}
	return matches, nil
}

// ReadFloat32AtAddrs reads the current float32 value at each address, silently
// dropping unreadable ones. Returns parallel slices of surviving addresses and values.
func ReadFloat32AtAddrs(mem memory.Memory, pid int, addrs []uintptr) ([]uintptr, []float32) {
	out := make([]uintptr, 0, len(addrs))
	vals := make([]float32, 0, len(addrs))
	for _, addr := range addrs {
		data, err := mem.ReadBytes(pid, addr, 4)
		if err != nil {
			continue
		}
		out = append(out, addr)
		vals = append(vals, math.Float32frombits(binary.LittleEndian.Uint32(data)))
	}
	return out, vals
}

// NarrowFloat32ByMode filters a previous float32 address list by comparing the
// current value against the previously recorded value.
// mode: "increased", "decreased", "changed", or "unchanged".
func NarrowFloat32ByMode(mem memory.Memory, pid int, addrs []uintptr, prevVals []float32, mode string) ([]uintptr, []float32, error) {
	if len(addrs) != len(prevVals) {
		return nil, nil, fmt.Errorf("address/value slice length mismatch")
	}
	var survivors []uintptr
	var newVals []float32
	for i, addr := range addrs {
		data, err := mem.ReadBytes(pid, addr, 4)
		if err != nil {
			continue
		}
		cur := math.Float32frombits(binary.LittleEndian.Uint32(data))
		var keep bool
		switch mode {
		case "increased":
			keep = cur > prevVals[i]
		case "decreased":
			keep = cur < prevVals[i]
		case "changed":
			keep = cur != prevVals[i]
		case "unchanged":
			keep = cur == prevVals[i]
		default:
			return nil, nil, fmt.Errorf("unknown scan mode %q", mode)
		}
		if keep {
			survivors = append(survivors, addr)
			newVals = append(newVals, cur)
		}
	}
	return survivors, newVals, nil
}

// ReadValuesAtAddrs reads the current int32 value at each address using batched
// reads. Returns parallel slices of surviving addresses and their values.
func ReadValuesAtAddrs(mem memory.Memory, pid int, addrs []uintptr) ([]uintptr, []int32) {
	vals := batchReadInt32(mem, pid, addrs)
	out := make([]uintptr, 0, len(vals))
	vs := make([]int32, 0, len(vals))
	for _, addr := range addrs {
		if v, ok := vals[addr]; ok {
			out = append(out, addr)
			vs = append(vs, v)
		}
	}
	return out, vs
}

// NarrowByMode filters a previous address list by comparing each address's
// current int32 value against the previously recorded value. mode must be one
// of: "increased", "decreased", "changed", "unchanged".
// Returns the surviving addresses and their current values.
func NarrowByMode(mem memory.Memory, pid int, addrs []uintptr, prevVals []int32, mode string) ([]uintptr, []int32, error) {
	if len(addrs) != len(prevVals) {
		return nil, nil, fmt.Errorf("address/value slice length mismatch")
	}
	cur := batchReadInt32(mem, pid, addrs)
	var survivors []uintptr
	var newVals []int32
	for i, addr := range addrs {
		v, ok := cur[addr]
		if !ok {
			continue
		}
		var keep bool
		switch mode {
		case "increased":
			keep = v > prevVals[i]
		case "decreased":
			keep = v < prevVals[i]
		case "changed":
			keep = v != prevVals[i]
		case "unchanged":
			keep = v == prevVals[i]
		default:
			return nil, nil, fmt.Errorf("unknown scan mode %q", mode)
		}
		if keep {
			survivors = append(survivors, addr)
			newVals = append(newVals, v)
		}
	}
	return survivors, newVals, nil
}

// ScanForInt64 scans all readable memory regions for the given int64 value.
func ScanForInt64(mem memory.Memory, pid int, value int64) ([]uintptr, error) {
	regions, err := mem.ReadableRegions(pid)
	if err != nil {
		return nil, fmt.Errorf("enumerating regions: %w", err)
	}
	target := make([]byte, 8)
	binary.LittleEndian.PutUint64(target, uint64(value))
	var matches []uintptr
	for _, region := range regions {
		if region.Size < 8 || region.Size > maxRegionSize {
			continue
		}
		data, err := mem.ReadBytes(pid, region.Start, int(region.Size))
		if err != nil {
			continue
		}
		for i := 0; i <= len(data)-8; i++ {
			if data[i] == target[0] && data[i+1] == target[1] &&
				data[i+2] == target[2] && data[i+3] == target[3] &&
				data[i+4] == target[4] && data[i+5] == target[5] &&
				data[i+6] == target[6] && data[i+7] == target[7] {
				matches = append(matches, region.Start+uintptr(i))
			}
		}
	}
	return matches, nil
}

// ScanForFloat64 scans all readable memory regions for the given float64 value.
func ScanForFloat64(mem memory.Memory, pid int, value float64) ([]uintptr, error) {
	regions, err := mem.ReadableRegions(pid)
	if err != nil {
		return nil, fmt.Errorf("enumerating regions: %w", err)
	}
	target := make([]byte, 8)
	binary.LittleEndian.PutUint64(target, math.Float64bits(value))
	var matches []uintptr
	for _, region := range regions {
		if region.Size < 8 || region.Size > maxRegionSize {
			continue
		}
		data, err := mem.ReadBytes(pid, region.Start, int(region.Size))
		if err != nil {
			continue
		}
		for i := 0; i <= len(data)-8; i++ {
			if data[i] == target[0] && data[i+1] == target[1] &&
				data[i+2] == target[2] && data[i+3] == target[3] &&
				data[i+4] == target[4] && data[i+5] == target[5] &&
				data[i+6] == target[6] && data[i+7] == target[7] {
				matches = append(matches, region.Start+uintptr(i))
			}
		}
	}
	return matches, nil
}

// NarrowScanInt64 filters a previous address list to those matching newValue as int64.
func NarrowScanInt64(mem memory.Memory, pid int, addrs []uintptr, newValue int64) ([]uintptr, error) {
	var matches []uintptr
	for _, addr := range addrs {
		data, err := mem.ReadBytes(pid, addr, 8)
		if err != nil {
			continue
		}
		if int64(binary.LittleEndian.Uint64(data)) == newValue {
			matches = append(matches, addr)
		}
	}
	return matches, nil
}

// NarrowScanFloat64 filters a previous address list to those matching newValue as float64.
func NarrowScanFloat64(mem memory.Memory, pid int, addrs []uintptr, newValue float64) ([]uintptr, error) {
	bits := math.Float64bits(newValue)
	var matches []uintptr
	for _, addr := range addrs {
		data, err := mem.ReadBytes(pid, addr, 8)
		if err != nil {
			continue
		}
		if binary.LittleEndian.Uint64(data) == bits {
			matches = append(matches, addr)
		}
	}
	return matches, nil
}

// NarrowInt64ByMode filters a previous int64 address list by change mode.
func NarrowInt64ByMode(mem memory.Memory, pid int, addrs []uintptr, prevVals []int64, mode string) ([]uintptr, []int64, error) {
	if len(addrs) != len(prevVals) {
		return nil, nil, fmt.Errorf("address/value slice length mismatch")
	}
	var survivors []uintptr
	var newVals []int64
	for i, addr := range addrs {
		data, err := mem.ReadBytes(pid, addr, 8)
		if err != nil {
			continue
		}
		cur := int64(binary.LittleEndian.Uint64(data))
		var keep bool
		switch mode {
		case "increased":
			keep = cur > prevVals[i]
		case "decreased":
			keep = cur < prevVals[i]
		case "changed":
			keep = cur != prevVals[i]
		case "unchanged":
			keep = cur == prevVals[i]
		default:
			return nil, nil, fmt.Errorf("unknown scan mode %q", mode)
		}
		if keep {
			survivors = append(survivors, addr)
			newVals = append(newVals, cur)
		}
	}
	return survivors, newVals, nil
}

// NarrowFloat64ByMode filters a previous float64 address list by change mode.
func NarrowFloat64ByMode(mem memory.Memory, pid int, addrs []uintptr, prevVals []float64, mode string) ([]uintptr, []float64, error) {
	if len(addrs) != len(prevVals) {
		return nil, nil, fmt.Errorf("address/value slice length mismatch")
	}
	var survivors []uintptr
	var newVals []float64
	for i, addr := range addrs {
		data, err := mem.ReadBytes(pid, addr, 8)
		if err != nil {
			continue
		}
		cur := math.Float64frombits(binary.LittleEndian.Uint64(data))
		var keep bool
		switch mode {
		case "increased":
			keep = cur > prevVals[i]
		case "decreased":
			keep = cur < prevVals[i]
		case "changed":
			keep = cur != prevVals[i]
		case "unchanged":
			keep = cur == prevVals[i]
		default:
			return nil, nil, fmt.Errorf("unknown scan mode %q", mode)
		}
		if keep {
			survivors = append(survivors, addr)
			newVals = append(newVals, cur)
		}
	}
	return survivors, newVals, nil
}

// FindModuleBase scans the process's readable regions for a Mach-O 64-bit
// header (magic 0xFEEDFACF) at an address >= 0x100000000, which is the
// default load address of the main executable on macOS arm64/amd64.
// Returns the actual (ASLR-adjusted) base address of the main module.
func FindModuleBase(mem memory.Memory, pid int) (uintptr, error) {
	const (
		machoMagic64  = uint32(0xFEEDFACF)
		defaultBase   = uintptr(0x100000000)
	)

	regions, err := mem.ReadableRegions(pid)
	if err != nil {
		return 0, fmt.Errorf("enumerating regions: %w", err)
	}

	for _, r := range regions {
		if r.Start < defaultBase || r.Size < 4 {
			continue
		}
		data, err := mem.ReadBytes(pid, r.Start, 4)
		if err != nil {
			continue
		}
		if binary.LittleEndian.Uint32(data) == machoMagic64 {
			return r.Start, nil
		}
	}
	return 0, fmt.Errorf("main module base not found for pid %d", pid)
}

// ResolvePointer walks a pointer chain starting at base, applying each offset
// in turn: it reads the pointer at (current + offset) and follows it.
// Returns the final resolved address.
func ResolvePointer(mem memory.Memory, pid int, base uintptr, offsets []uintptr) (uintptr, error) {
	addr := base
	for i, offset := range offsets {
		addr += offset
		val, err := mem.ReadInt64(pid, addr)
		if err != nil {
			return 0, fmt.Errorf("resolving pointer at step %d (0x%x): %w", i, addr, err)
		}
		addr = uintptr(val)
	}
	return addr, nil
}
