package openh264

import (
	"bytes"
	"os"
	"testing"
)

// TestDecodeRealBaseline — regression test on a real Constrained Baseline frame
// (176x144). The reference testdata/*.yuv is the expected raw YUV420 output; our
// decoder must match it byte-for-byte.
func TestDecodeRealBaseline(t *testing.T) {
	annexB, err := os.ReadFile("testdata/baseline_176x144.264")
	if err != nil {
		t.Fatalf("read .264: %v", err)
	}
	want, err := os.ReadFile("testdata/baseline_176x144.yuv")
	if err != nil {
		t.Fatalf("read reference: %v", err)
	}

	frame, h, err := NewDecoder().DecodeFirstFrame(annexB)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if frame.CropWidth != 176 || frame.CropHeight != 144 {
		t.Fatalf("size %dx%d, want 176x144", frame.CropWidth, frame.CropHeight)
	}
	if !h.Type.IsI() {
		t.Fatalf("slice %s, want I", h.Type)
	}

	var got bytes.Buffer
	got.Write(frame.Y)
	got.Write(frame.Cb)
	got.Write(frame.Cr)
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("YUV did not match the reference (got %d bytes, want %d)", got.Len(), len(want))
	}
}

// TestDecodeRealMainCABAC — regression on a real Main-profile (CABAC) frame
// (176x144). The output must match the raw YUV420 reference byte-for-byte.
func TestDecodeRealMainCABAC(t *testing.T) {
	annexB, err := os.ReadFile("testdata/main_cabac_176x144.264")
	if err != nil {
		t.Fatalf("read .264: %v", err)
	}
	want, err := os.ReadFile("testdata/main_cabac_176x144.yuv")
	if err != nil {
		t.Fatalf("read reference: %v", err)
	}
	frame, _, err := NewDecoder().DecodeFirstFrame(annexB)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	var got bytes.Buffer
	got.Write(frame.Y)
	got.Write(frame.Cb)
	got.Write(frame.Cr)
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("CABAC YUV did not match the reference (got %d, want %d)", got.Len(), len(want))
	}
}

// TestDecodeRealHigh — regression on a real High-profile frame (176x144, 8x8
// transform). The output must match the raw YUV420 reference byte-for-byte.
func TestDecodeRealHigh(t *testing.T) {
	annexB, err := os.ReadFile("testdata/high_176x144.264")
	if err != nil {
		t.Fatalf("read .264: %v", err)
	}
	want, err := os.ReadFile("testdata/high_176x144.yuv")
	if err != nil {
		t.Fatalf("read reference: %v", err)
	}
	frame, _, err := NewDecoder().DecodeFirstFrame(annexB)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	var got bytes.Buffer
	got.Write(frame.Y)
	got.Write(frame.Cb)
	got.Write(frame.Cr)
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("High YUV did not match the reference (got %d, want %d)", got.Len(), len(want))
	}
}
