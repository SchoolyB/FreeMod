package memory

// MemRegion describes a readable memory region in a process.
type MemRegion struct {
	Start uintptr
	Size  uintptr
	Perms string // e.g. "rw-"
}

// MemoryReader defines read operations on a process's memory.
type MemoryReader interface {
	ReadInt(pid int, addr uintptr) (int32, error)
	ReadInt64(pid int, addr uintptr) (int64, error)
	ReadFloat(pid int, addr uintptr) (float32, error)
	ReadBytes(pid int, addr uintptr, length int) ([]byte, error)
	ReadableRegions(pid int) ([]MemRegion, error)
}

// MemoryWriter defines write operations on a process's memory.
type MemoryWriter interface {
	WriteInt(pid int, addr uintptr, val int32) error
	WriteInt64(pid int, addr uintptr, val int64) error
	WriteFloat(pid int, addr uintptr, val float32) error
	WriteBytes(pid int, addr uintptr, data []byte) error
}

// Memory combines read and write capabilities.
type Memory interface {
	MemoryReader
	MemoryWriter
}
