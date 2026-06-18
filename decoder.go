// Package openh264 — a high-level end-to-end decoder: Annex-B stream → first frame.
// It wires the nal/h264 layers together and holds the active parameter sets.
// It decodes only the first picture (decoding the first Annex-B H.264 frame).
package openh264

import (
	"errors"

	"github.com/mgvs/go-openh264/bitstream"
	"github.com/mgvs/go-openh264/h264"
	"github.com/mgvs/go-openh264/nal"
)

// Decoder holds parsed SPS/PPS and decodes frames from an Annex-B stream.
type Decoder struct {
	sps map[uint32]*h264.SPS
	pps map[uint32]*h264.PPS
}

func NewDecoder() *Decoder {
	return &Decoder{sps: map[uint32]*h264.SPS{}, pps: map[uint32]*h264.PPS{}}
}

// ErrNoPicture — no decodable picture was found in the stream.
var ErrNoPicture = errors.New("h264: no decodable picture found")

// DecodeFirstFrame parses an Annex-B stream and returns the first decoded
// picture (Frame). Parameter sets are accumulated along the way. A picture may
// be split into several slices (multi-slice); all slices of the first picture
// are decoded into the same frame before returning.
func (d *Decoder) DecodeFirstFrame(annexB []byte) (*h264.Frame, *h264.SliceHeader, error) {
	var (
		frame   *h264.Frame
		lastHdr *h264.SliceHeader
	)
	for _, u := range nal.ParseAnnexB(annexB) {
		switch u.Type {
		case nal.TypeSPS:
			if s, err := h264.ParseSPS(u.RBSP); err == nil {
				d.sps[s.ID] = s
			}
		case nal.TypePPS:
			if p, err := h264.ParsePPS(u.RBSP); err == nil {
				d.pps[p.ID] = p
			}
		case nal.TypeIDR, nal.TypeNonIDR:
			pps := d.pps[slicePPSID(u.RBSP)]
			if pps == nil {
				continue
			}
			sps := d.sps[pps.SPSID]
			if sps == nil {
				continue
			}
			// Новая картинка (first_mb_in_slice==0 после уже декодированной) —
			// возвращаем первую.
			if frame != nil && sliceFirstMB(u.RBSP) == 0 {
				return frame, lastHdr, nil
			}
			if frame == nil {
				frame = h264.NewFrame(sps)
			}
			h, err := h264.DecodeSlice(u.RBSP, u.Type, u.RefIDC, sps, pps, frame)
			if err != nil {
				// Первый слайс упал — это ошибка; иначе вернём, что собрали
				// (частичный кадр лучше пустого).
				if lastHdr == nil {
					return nil, h, err
				}
				return frame, lastHdr, nil
			}
			lastHdr = h
		}
	}
	if frame != nil {
		return frame, lastHdr, nil
	}
	return nil, nil, ErrNoPicture
}

// slicePPSID reads pic_parameter_set_id from the start of the slice header
// (first_mb_in_slice, slice_type, pic_parameter_set_id — three ue values in a row).
func slicePPSID(rbsp []byte) uint32 {
	r := bitstream.NewReader(rbsp)
	r.UE() // first_mb_in_slice
	r.UE() // slice_type
	return r.UE()
}

// sliceFirstMB reads first_mb_in_slice (the first ue of the slice header).
func sliceFirstMB(rbsp []byte) uint32 {
	return bitstream.NewReader(rbsp).UE()
}
