// Package h264dec — a high-level end-to-end decoder: Annex-B stream → first frame.
// It wires the nal/h264 layers together and holds the active parameter sets.
// It decodes only the first picture (decoding the first Annex-B H.264 frame).
package main

import (
	"errors"

	"go-openh264/bitstream"
	"go-openh264/h264"
	"go-openh264/nal"
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
// picture (Frame). Parameter sets are accumulated along the way.
func (d *Decoder) DecodeFirstFrame(annexB []byte) (*h264.Frame, *h264.SliceHeader, error) {
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
			ppsID := slicePPSID(u.RBSP)
			pps := d.pps[ppsID]
			if pps == nil {
				continue
			}
			sps := d.sps[pps.SPSID]
			if sps == nil {
				continue
			}
			frame := h264.NewFrame(sps)
			h, err := h264.DecodeSlice(u.RBSP, u.Type, u.RefIDC, sps, pps, frame)
			if err != nil {
				return nil, h, err
			}
			return frame, h, nil
		}
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
