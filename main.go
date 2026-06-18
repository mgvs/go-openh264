package main

import (
	"fmt"
	"image/png"
	"os"
	"strings"

	"go-openh264/mp4"
)

// h264dec — a decoder test utility. It reads an Annex-B .264 stream or an
// MP4/MOV file and saves the first decoded frame as PNG (or raw .yuv).
//
//	go run . input.{264,mp4,mov,m4v} [output.{png,yuv}]
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: h264dec <input.264> [output.png]")
		os.Exit(2)
	}
	in := os.Args[1]
	out := "frame.png"
	if len(os.Args) >= 3 {
		out = os.Args[2]
	}

	data, err := os.ReadFile(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}

	// MP4/MOV containers: demux the first keyframe to an Annex-B stream.
	lower := strings.ToLower(in)
	if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".mov") || strings.HasSuffix(lower, ".m4v") {
		data, err = mp4.ExtractFirstKeyframe(data)
		if err != nil {
			fmt.Fprintln(os.Stderr, "demux:", err)
			os.Exit(1)
		}
		// Debug: dump the demuxed Annex-B stream instead of decoding.
		if strings.HasSuffix(out, ".264") {
			if err := os.WriteFile(out, data, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "write:", err)
				os.Exit(1)
			}
			fmt.Println("demuxed Annex-B saved:", out, len(data), "bytes")
			return
		}
	}

	frame, h, err := NewDecoder().DecodeFirstFrame(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "decode:", err)
		os.Exit(1)
	}
	fmt.Printf("frame %dx%d, slice %s, QP=%d\n", frame.CropWidth, frame.CropHeight, h.Type, h.SliceQP)

	f, err := os.Create(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	defer f.Close()

	if strings.HasSuffix(out, ".yuv") {
		// Raw YUV420 planes (Y, then Cb, Cr) of the visible region — for byte-exact
		// comparison against a raw YUV420 reference.
		y, cb, cr := frame.CroppedYUV()
		for _, p := range [][]byte{y, cb, cr} {
			if _, err := f.Write(p); err != nil {
				fmt.Fprintln(os.Stderr, "write YUV:", err)
				os.Exit(1)
			}
		}
	} else if err := png.Encode(f, frame.YCbCr()); err != nil {
		fmt.Fprintln(os.Stderr, "PNG:", err)
		os.Exit(1)
	}
	fmt.Println("saved:", out)
}
