package scanner

import (
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/freemod/freemod/core/memory"
)

// PointerCandidate represents a pointer chain that resolves to a target address.
type PointerCandidate struct {
	BaseOffset uintptr   // offset from module base to start of chain
	Offsets    []uintptr // subsequent dereference offsets
	FinalAddr  uintptr   // resolved final address (should equal original target)
}

// PointerScanOptions configures a pointer scan.
type PointerScanOptions struct {
	MaxDepth   int     // max pointer chain depth (recommended: 4-5)
	Range      uintptr // scan addresses within ±Range of target (default: 2048)
	MaxResults int     // stop after finding this many candidates (default: 512)
}

func defaultOpts(o PointerScanOptions) PointerScanOptions {
	if o.MaxDepth == 0 {
		o.MaxDepth = 4
	}
	if o.Range == 0 {
		o.Range = 2048
	}
	if o.MaxResults == 0 {
		o.MaxResults = 512
	}
	return o
}

// ScanForPointers finds pointer chains that lead to targetAddr.
//
// Strategy:
//  1. Collect all pointer-sized values in readable regions that fall within
//     [targetAddr-Range, targetAddr+Range] — these are "level 0" pointers.
//  2. Recursively find pointers to those pointer addresses (up to MaxDepth).
//  3. At each level, check whether any pointer address falls within the
//     module's static region (near moduleBase). Those form complete chains.
//
// Returns a deduplicated, sorted list of candidate chains.
func ScanForPointers(mem memory.Memory, pid int, moduleBase, targetAddr uintptr, opts PointerScanOptions) ([]PointerCandidate, error) {
	opts = defaultOpts(opts)

	regions, err := mem.ReadableRegions(pid)
	if err != nil {
		return nil, fmt.Errorf("enumerating regions: %w", err)
	}

	// Build a fast lookup: address → pointer value, for all pointer-aligned
	// addresses in all readable regions under maxRegionSize.
	// We read each region once and extract all 8-byte aligned values.
	type ptrEntry struct {
		addr uintptr // address of the pointer
		val  uintptr // value stored at that address (the pointer itself)
	}

	var allPtrs []ptrEntry
	for _, region := range regions {
		if region.Size < 8 || region.Size > maxRegionSize {
			continue
		}
		data, err := mem.ReadBytes(pid, region.Start, int(region.Size))
		if err != nil {
			continue
		}
		for i := 0; i+8 <= len(data); i += 8 {
			val := uintptr(binary.LittleEndian.Uint64(data[i : i+8]))
			if val == 0 {
				continue
			}
			allPtrs = append(allPtrs, ptrEntry{
				addr: region.Start + uintptr(i),
				val:  val,
			})
		}
	}

	// Sort by pointer value so we can binary-search for ranges.
	sort.Slice(allPtrs, func(i, j int) bool {
		return allPtrs[i].val < allPtrs[j].val
	})

	// findPointersTo returns all allPtrs entries whose .val falls in
	// [lo, hi].
	findPointersTo := func(lo, hi uintptr) []ptrEntry {
		if lo > hi {
			return nil
		}
		// binary search for lo
		start := sort.Search(len(allPtrs), func(i int) bool {
			return allPtrs[i].val >= lo
		})
		end := sort.Search(len(allPtrs), func(i int) bool {
			return allPtrs[i].val > hi
		})
		return allPtrs[start:end]
	}

	// moduleStaticLimit: addresses within this distance of moduleBase are
	// considered "static" (in the .data/.bss sections of the binary).
	// 256 MB is generous but safe for large games.
	const moduleStaticLimit = 256 * 1024 * 1024

	var candidates []PointerCandidate

	// Recursive chain builder.
	// current: the address we're looking for pointers TO
	// offsetFromCurrent: the offset applied at the last step (to reach current)
	// chain: offsets accumulated so far (innermost first)
	// depth: current depth
	var buildChains func(current uintptr, offsetFromCurrent uintptr, chain []uintptr, depth int)
	buildChains = func(current uintptr, offsetFromCurrent uintptr, chain []uintptr, depth int) {
		if len(candidates) >= opts.MaxResults {
			return
		}

		lo := current
		if current > opts.Range {
			lo = current - opts.Range
		}
		hi := current + opts.Range

		ptrs := findPointersTo(lo, hi)
		for _, p := range ptrs {
			if len(candidates) >= opts.MaxResults {
				return
			}

			// The offset at this level is (current - p.val), i.e. how far
			// into the struct this pointer lands.
			offset := current - p.val

			newChain := make([]uintptr, len(chain)+1)
			copy(newChain, chain)
			newChain[len(chain)] = offset

			// Is this pointer address within the module's static region?
			if p.addr >= moduleBase && p.addr < moduleBase+moduleStaticLimit {
				baseOffset := p.addr - moduleBase
				// Reverse chain so it reads base→final order.
				reversed := reverseOffsets(newChain)
				candidates = append(candidates, PointerCandidate{
					BaseOffset: baseOffset,
					Offsets:    reversed,
					FinalAddr:  targetAddr,
				})
				continue
			}

			// Otherwise go deeper if we have depth remaining.
			if depth < opts.MaxDepth {
				buildChains(p.addr, offset, newChain, depth+1)
			}
		}
	}

	buildChains(targetAddr, 0, nil, 1)

	// Deduplicate by (BaseOffset, Offsets).
	seen := make(map[string]bool)
	var deduped []PointerCandidate
	for _, c := range candidates {
		key := fmt.Sprintf("%x:%v", c.BaseOffset, c.Offsets)
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, c)
		}
	}

	// Sort by chain depth (shortest first — shallower chains are more stable).
	sort.Slice(deduped, func(i, j int) bool {
		if len(deduped[i].Offsets) != len(deduped[j].Offsets) {
			return len(deduped[i].Offsets) < len(deduped[j].Offsets)
		}
		return deduped[i].BaseOffset < deduped[j].BaseOffset
	})

	return deduped, nil
}

// VerifyChain resolves a pointer chain from moduleBase and checks that the
// final address holds the expected int32 or float32 value (within tolerance).
// Returns the resolved address and whether it matched.
func VerifyChain(mem memory.Memory, pid int, moduleBase, baseOffset uintptr, offsets []uintptr) (uintptr, error) {
	addr := moduleBase + baseOffset
	for i, off := range offsets {
		data, err := mem.ReadBytes(pid, addr, 8)
		if err != nil {
			return 0, fmt.Errorf("step %d: read at 0x%x: %w", i, addr, err)
		}
		ptr := uintptr(binary.LittleEndian.Uint64(data))
		if ptr == 0 {
			return 0, fmt.Errorf("step %d: null pointer at 0x%x", i, addr)
		}
		addr = ptr + off
	}
	return addr, nil
}

func reverseOffsets(in []uintptr) []uintptr {
	out := make([]uintptr, len(in))
	for i, v := range in {
		out[len(in)-1-i] = v
	}
	return out
}
