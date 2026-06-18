// Package mp4 — a minimal ISO BMFF (MP4/MOV) demuxer: it extracts the first
// keyframe of the H.264 video track as an Annex-B stream (SPS+PPS+IDR) ready for
// the decoder. Pure Go, stdlib only; only what is needed for thumbnails.
package mp4

import (
	"encoding/binary"
	"errors"
)

var (
	ErrNoVideo    = errors.New("mp4: no H.264 video track found")
	ErrNoKeyframe = errors.New("mp4: could not locate the first keyframe")
)

// startCode is the 4-byte Annex-B start code.
var startCode = []byte{0, 0, 0, 1}

// ExtractFirstKeyframe parses an MP4/MOV file and returns the first keyframe of
// its H.264 track as an Annex-B byte stream (parameter sets prepended).
func ExtractFirstKeyframe(data []byte) ([]byte, error) {
	moov := findBox(data, "moov")
	if moov == nil {
		return nil, errors.New("mp4: no moov box")
	}
	stbl := videoStbl(moov)
	if stbl == nil {
		return nil, ErrNoVideo
	}

	cfg, err := parseAvcC(stbl)
	if err != nil {
		return nil, err
	}

	off, size, err := firstKeyframeRange(stbl)
	if err != nil {
		return nil, err
	}
	if off+size > len(data) || size <= 0 {
		return nil, ErrNoKeyframe
	}
	sample := data[off : off+size]

	// Build Annex-B: SPS/PPS then the sample's length-prefixed NAL units.
	var out []byte
	for _, sps := range cfg.sps {
		out = append(append(out, startCode...), sps...)
	}
	for _, pps := range cfg.pps {
		out = append(append(out, startCode...), pps...)
	}
	for p := 0; p+cfg.nalLenSize <= len(sample); {
		n := 0
		for i := 0; i < cfg.nalLenSize; i++ {
			n = n<<8 | int(sample[p+i])
		}
		p += cfg.nalLenSize
		if n <= 0 || p+n > len(sample) {
			break
		}
		out = append(append(out, startCode...), sample[p:p+n]...)
		p += n
	}
	return out, nil
}

// findBox returns the content of the first top-level child box of the given type.
func findBox(b []byte, typ string) []byte {
	var found []byte
	eachBox(b, func(t string, content []byte) {
		if found == nil && t == typ {
			found = content
		}
	})
	return found
}

// eachBox iterates the child boxes of a box payload, calling fn(type, content).
func eachBox(b []byte, fn func(typ string, content []byte)) {
	for len(b) >= 8 {
		size := int(binary.BigEndian.Uint32(b[0:4]))
		typ := string(b[4:8])
		hdr := 8
		total := size
		switch size {
		case 1:
			if len(b) < 16 {
				return
			}
			total = int(binary.BigEndian.Uint64(b[8:16]))
			hdr = 16
		case 0:
			total = len(b)
		}
		if total < hdr || total > len(b) {
			return
		}
		fn(typ, b[hdr:total])
		b = b[total:]
	}
}

// videoStbl finds the sample table (stbl) of the H.264 video track inside moov.
func videoStbl(moov []byte) []byte {
	var stbl []byte
	eachBox(moov, func(t string, trak []byte) {
		if stbl != nil || t != "trak" {
			return
		}
		mdia := findBox(trak, "mdia")
		if mdia == nil || string(handlerType(mdia)) != "vide" {
			return
		}
		if minf := findBox(mdia, "minf"); minf != nil {
			stbl = findBox(minf, "stbl")
		}
	})
	return stbl
}

// handlerType returns the 4-byte handler type from mdia/hdlr.
func handlerType(mdia []byte) []byte {
	hdlr := findBox(mdia, "hdlr")
	if len(hdlr) < 12 {
		return nil
	}
	// fullbox(4) + pre_defined(4) + handler_type(4)
	return hdlr[8:12]
}
