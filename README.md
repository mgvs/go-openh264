# go-openh264

An H.264/AVC decoder in **pure Go** — no cgo, no external binaries, cross-platform.
The goal is to produce a frame image from H.264 using a single static pure-Go binary.

This is **not a full port** of any existing codec — it is a **decoder only** (no
encoder), and it implements just the subset needed to turn a keyframe into an
image. It cannot encode H.264.

## Port references

- Cisco OpenH264 (BSD-2) — <https://github.com/cisco/openh264>
- The ISO/IEC 14496-10 (H.264) specification

## Structure

| Package | Purpose | Status |
|---|---|---|
| `bitstream` | bit reader + Exp-Golomb, alignment, `more_rbsp_data` | ✅ |
| `nal` | Annex-B split, EBSP→RBSP, NAL header | ✅ |
| `h264` | SPS/PPS parsing (frame size, profile, entropy) | ✅ |
| `h264` | slice header (I/P/B), `SliceQP`, ref-list/marking | ✅ |
| `h264` | slice_data skeleton, `mb_type` (I), frame buffer, **I_PCM** | ✅ |
| `h264` | intra prediction: I_16x16 (4) + chroma 4:2:0 (4) | ✅ |
| `h264` | inverse transform/dequant (4x4, luma/chroma DC) | ✅ |
| `h264` | CAVLC: coeff_token, levels, total_zeros, run_before, assembly | ✅ |
| `h264` | **I_16x16 reconstruction** (mb_pred+residual, nC context, 4:2:0) | ✅ |
| `h264` | **I_NxN reconstruction** (intra 4×4, 9 modes, me(CBP)) | ✅ |
| `h264` | **deblocking filter** (I-slice, luma+chroma 4:2:0) | ✅ |
| `h264` | **CABAC** engine, context init, I-slice syntax + residual | ✅ |
| `h264` | **High profile** Intra_8×8 (prediction, 8×8 transform, deblocking) | ✅ |
| `mp4` | minimal ISO BMFF demux: first keyframe + AVCC→Annex-B | ✅ |
| `.` (h264dec) | end-to-end decode `.264`/`.mp4` → PNG/YUV | ✅ |
| `h264` | P-slice / B-slice (inter prediction) | — *out of scope* |

## License

The port code is BSD-2 (see `LICENSE`, `NOTICE`).

## Tests

```sh
cd go-openh264 && go test ./...
```
