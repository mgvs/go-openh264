package mp4

import (
	"encoding/binary"
	"errors"
)

type avcConfig struct {
	sps, pps   [][]byte
	nalLenSize int
}

// parseAvcC finds and parses the AVCDecoderConfigurationRecord (avcC) in stbl.
func parseAvcC(stbl []byte) (*avcConfig, error) {
	stsd := findBox(stbl, "stsd")
	if len(stsd) < 8 {
		return nil, errors.New("mp4: no stsd")
	}
	// stsd: fullbox(4) + entry_count(4), then sample entry boxes.
	var avcc []byte
	eachBox(stsd[8:], func(t string, entry []byte) {
		if avcc == nil && (t == "avc1" || t == "avc3") && len(entry) > 78 {
			avcc = findBox(entry[78:], "avcC") // skip the visual sample entry header
		}
	})
	if len(avcc) < 7 {
		return nil, errors.New("mp4: no avcC")
	}

	cfg := &avcConfig{nalLenSize: int(avcc[4]&0x03) + 1}
	p := 5
	numSPS := int(avcc[p] & 0x1f)
	p++
	for i := 0; i < numSPS && p+2 <= len(avcc); i++ {
		n := int(binary.BigEndian.Uint16(avcc[p:]))
		p += 2
		if p+n > len(avcc) {
			break
		}
		cfg.sps = append(cfg.sps, avcc[p:p+n])
		p += n
	}
	if p >= len(avcc) {
		return cfg, nil
	}
	numPPS := int(avcc[p])
	p++
	for i := 0; i < numPPS && p+2 <= len(avcc); i++ {
		n := int(binary.BigEndian.Uint16(avcc[p:]))
		p += 2
		if p+n > len(avcc) {
			break
		}
		cfg.pps = append(cfg.pps, avcc[p:p+n])
		p += n
	}
	return cfg, nil
}

// u32 reads a big-endian uint32 at offset i (0 if out of range).
func u32(b []byte, i int) int {
	if i+4 > len(b) {
		return 0
	}
	return int(binary.BigEndian.Uint32(b[i:]))
}

// firstKeyframeRange returns the byte offset and size of the first keyframe sample.
func firstKeyframeRange(stbl []byte) (int, int, error) {
	// First sync sample (1-based); without stss every sample is a sync sample.
	keyframe := 1
	if stss := findBox(stbl, "stss"); len(stss) >= 8 && u32(stss, 4) > 0 {
		keyframe = u32(stss, 8)
	}

	sizes := parseStsz(findBox(stbl, "stsz"))
	chunks := parseChunkOffsets(stbl)
	stsc := parseStsc(findBox(stbl, "stsc"))
	if len(chunks) == 0 || keyframe < 1 {
		return 0, 0, ErrNoKeyframe
	}

	// Locate the chunk holding sample `keyframe` (1-based) and its index within it.
	target := keyframe - 1 // 0-based
	sampleCount := 0
	for i := 0; i < len(stsc); i++ {
		firstChunk := stsc[i].firstChunk
		nextFirst := len(chunks) + 1
		if i+1 < len(stsc) {
			nextFirst = stsc[i+1].firstChunk
		}
		spc := stsc[i].samplesPerChunk
		for chunk := firstChunk; chunk < nextFirst && chunk <= len(chunks); chunk++ {
			if target < sampleCount+spc {
				idxInChunk := target - sampleCount
				off := chunks[chunk-1]
				for s := target - idxInChunk; s < target; s++ {
					off += sampleSize(sizes, s)
				}
				return off, sampleSize(sizes, target), nil
			}
			sampleCount += spc
		}
	}
	return 0, 0, ErrNoKeyframe
}

type stscEntry struct{ firstChunk, samplesPerChunk int }

func parseStsc(b []byte) []stscEntry {
	if len(b) < 8 {
		return nil
	}
	n := u32(b, 4)
	var out []stscEntry
	for i := 0; i < n; i++ {
		o := 8 + i*12
		if o+8 > len(b) {
			break
		}
		out = append(out, stscEntry{firstChunk: u32(b, o), samplesPerChunk: u32(b, o+4)})
	}
	return out
}

// parseStsz returns (defaultSize, perSampleSizes). If defaultSize != 0, all
// samples share it.
func parseStsz(b []byte) []int {
	if len(b) < 12 {
		return nil
	}
	def := u32(b, 4)
	count := u32(b, 8)
	if def != 0 {
		return []int{-def} // encode constant size as a single negative sentinel
	}
	out := make([]int, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, u32(b, 12+i*4))
	}
	return out
}

func sampleSize(sizes []int, s int) int {
	if len(sizes) == 1 && sizes[0] < 0 {
		return -sizes[0]
	}
	if s < 0 || s >= len(sizes) {
		return 0
	}
	return sizes[s]
}

func parseChunkOffsets(stbl []byte) []int {
	if stco := findBox(stbl, "stco"); len(stco) >= 8 {
		n := u32(stco, 4)
		out := make([]int, 0, n)
		for i := 0; i < n; i++ {
			out = append(out, u32(stco, 8+i*4))
		}
		return out
	}
	if co64 := findBox(stbl, "co64"); len(co64) >= 8 {
		n := u32(co64, 4)
		out := make([]int, 0, n)
		for i := 0; i < n; i++ {
			o := 8 + i*8
			if o+8 > len(co64) {
				break
			}
			out = append(out, int(binary.BigEndian.Uint64(co64[o:])))
		}
		return out
	}
	return nil
}
