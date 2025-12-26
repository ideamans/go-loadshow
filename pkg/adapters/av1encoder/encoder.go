// Package av1encoder provides an AV1 video encoder using libaom.
package av1encoder

/*
#cgo !windows pkg-config: aom
#cgo windows CFLAGS: -IC:/vcpkg/installed/x64-windows-static/include
#cgo windows LDFLAGS: -LC:/vcpkg/installed/x64-windows-static/lib -laom -static -lpthread
#include <aom/aom_encoder.h>
#include <aom/aomcx.h>
#include <stdlib.h>
#include <string.h>

static aom_codec_iface_t* get_av1_interface() {
    return aom_codec_av1_cx();
}

// Wrapper for aom_codec_enc_init
static aom_codec_err_t init_encoder(aom_codec_ctx_t *ctx, aom_codec_iface_t *iface,
                                     aom_codec_enc_cfg_t *cfg, aom_codec_flags_t flags) {
    return aom_codec_enc_init_ver(ctx, iface, cfg, flags, AOM_ENCODER_ABI_VERSION);
}

// Helper functions to access packet data
static int is_frame_packet(const aom_codec_cx_pkt_t *pkt) {
    return pkt->kind == AOM_CODEC_CX_FRAME_PKT;
}

static void* get_frame_buf(const aom_codec_cx_pkt_t *pkt) {
    return pkt->data.frame.buf;
}

static size_t get_frame_sz(const aom_codec_cx_pkt_t *pkt) {
    return pkt->data.frame.sz;
}

static int is_keyframe(const aom_codec_cx_pkt_t *pkt) {
    return (pkt->data.frame.flags & AOM_FRAME_IS_KEY) != 0;
}

static aom_codec_pts_t get_frame_pts(const aom_codec_cx_pkt_t *pkt) {
    return pkt->data.frame.pts;
}

// Helper to set YUV plane data
static void set_yuv_pixel(aom_image_t *img, int plane, int idx, unsigned char val) {
    img->planes[plane][idx] = val;
}

static int get_plane_stride(aom_image_t *img, int plane) {
    return img->stride[plane];
}

// Wrapper for aom_codec_control (it's a variadic macro)
static aom_codec_err_t set_cpu_used(aom_codec_ctx_t *ctx, int value) {
    return aom_codec_control(ctx, AOME_SET_CPUUSED, value);
}
*/
import "C"

import (
	"fmt"
	"image"
	"image/draw"
	"sync"
	"unsafe"

	"github.com/user/loadshow/pkg/ports"
)

// Encoder implements ports.VideoEncoder using libaom for AV1 encoding.
type Encoder struct {
	mu sync.Mutex

	codec    *C.aom_codec_ctx_t
	cfg      *C.aom_codec_enc_cfg_t
	rawFrame *C.aom_image_t

	width   int
	height  int
	fps     float64
	options ports.EncoderOptions

	frames     []encodedFrame
	frameCount int
}

type encodedFrame struct {
	data        []byte
	timestampUs int64
	isKeyframe  bool
}

// New creates a new AV1 encoder.
func New() *Encoder {
	return &Encoder{}
}

// Begin initializes the encoder.
func (e *Encoder) Begin(width, height int, fps float64, opts ports.EncoderOptions) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.width = width
	e.height = height
	e.fps = fps
	e.options = opts
	e.frames = nil
	e.frameCount = 0

	// Allocate codec context
	e.codec = (*C.aom_codec_ctx_t)(C.malloc(C.sizeof_aom_codec_ctx_t))
	if e.codec == nil {
		return fmt.Errorf("failed to allocate codec context")
	}
	C.memset(unsafe.Pointer(e.codec), 0, C.sizeof_aom_codec_ctx_t)

	// Allocate config
	e.cfg = (*C.aom_codec_enc_cfg_t)(C.malloc(C.sizeof_aom_codec_enc_cfg_t))
	if e.cfg == nil {
		C.free(unsafe.Pointer(e.codec))
		return fmt.Errorf("failed to allocate encoder config")
	}

	// Get codec interface
	iface := C.get_av1_interface()

	// Get default config
	if res := C.aom_codec_enc_config_default(iface, e.cfg, 0); res != C.AOM_CODEC_OK {
		e.cleanup()
		return fmt.Errorf("failed to get default config: %d", res)
	}

	// Configure encoder
	e.cfg.g_w = C.uint(width)
	e.cfg.g_h = C.uint(height)
	e.cfg.g_timebase.num = 1
	e.cfg.g_timebase.den = C.int(fps * 1000)
	e.cfg.g_error_resilient = 0
	e.cfg.g_threads = 4 // Use multiple threads for faster encoding

	// Use realtime mode for faster encoding
	e.cfg.g_usage = C.AOM_USAGE_REALTIME

	// Bitrate settings
	if opts.Bitrate > 0 {
		e.cfg.rc_target_bitrate = C.uint(opts.Bitrate)
	} else {
		// Default bitrate based on resolution
		e.cfg.rc_target_bitrate = C.uint(width * height / 1000)
	}

	// Quality settings (CQ mode)
	e.cfg.rc_end_usage = C.AOM_CQ
	if opts.Quality > 0 && opts.Quality <= 63 {
		e.cfg.rc_min_quantizer = C.uint(opts.Quality)
		e.cfg.rc_max_quantizer = C.uint(opts.Quality + 10)
		if e.cfg.rc_max_quantizer > 63 {
			e.cfg.rc_max_quantizer = 63
		}
	}

	// Initialize codec
	if res := C.init_encoder(e.codec, iface, e.cfg, 0); res != C.AOM_CODEC_OK {
		e.cleanup()
		return fmt.Errorf("failed to initialize encoder: %d", res)
	}

	// Set CPU usage for speed (0 = slowest/best, 10 = fastest)
	C.set_cpu_used(e.codec, 8)

	// Allocate raw frame
	e.rawFrame = (*C.aom_image_t)(C.malloc(C.sizeof_aom_image_t))
	if e.rawFrame == nil {
		C.aom_codec_destroy(e.codec)
		e.cleanup()
		return fmt.Errorf("failed to allocate raw frame")
	}

	if C.aom_img_alloc(e.rawFrame, C.AOM_IMG_FMT_I420, C.uint(width), C.uint(height), 32) == nil {
		C.free(unsafe.Pointer(e.rawFrame))
		C.aom_codec_destroy(e.codec)
		e.cleanup()
		return fmt.Errorf("failed to allocate image buffer")
	}

	return nil
}

// EncodeFrame encodes a single frame.
func (e *Encoder) EncodeFrame(img image.Image, timestampMs int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.codec == nil {
		return fmt.Errorf("encoder not initialized")
	}

	// Convert image to RGBA
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	// Convert RGBA to YUV420
	e.rgbaToYUV420(rgba)

	// Calculate timestamp in timebase units
	pts := C.aom_codec_pts_t(timestampMs * int(e.fps))

	// Encode frame
	flags := C.aom_enc_frame_flags_t(0)
	if e.frameCount == 0 {
		flags = C.AOM_EFLAG_FORCE_KF // Force keyframe for first frame
	}

	res := C.aom_codec_encode(e.codec, e.rawFrame, pts, 1, flags)
	if res != C.AOM_CODEC_OK {
		return fmt.Errorf("encoding failed: %d", res)
	}

	// Get encoded data
	var iter C.aom_codec_iter_t
	for {
		pkt := C.aom_codec_get_cx_data(e.codec, &iter)
		if pkt == nil {
			break
		}

		if C.is_frame_packet(pkt) != 0 {
			buf := C.get_frame_buf(pkt)
			sz := C.get_frame_sz(pkt)
			frameData := C.GoBytes(buf, C.int(sz))
			keyframe := C.is_keyframe(pkt) != 0
			pktPts := int64(C.get_frame_pts(pkt))

			// Convert PTS to microseconds
			timestampUs := pktPts * 1000 / int64(e.fps)

			e.frames = append(e.frames, encodedFrame{
				data:        frameData,
				timestampUs: timestampUs,
				isKeyframe:  keyframe,
			})
		}
	}

	e.frameCount++
	return nil
}

// End finalizes encoding and returns the MP4 data.
func (e *Encoder) End() ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.codec == nil {
		return nil, fmt.Errorf("encoder not initialized")
	}

	// Flush encoder
	res := C.aom_codec_encode(e.codec, nil, 0, 1, 0)
	if res != C.AOM_CODEC_OK {
		return nil, fmt.Errorf("flush failed: %d", res)
	}

	// Get remaining packets
	var iter C.aom_codec_iter_t
	for {
		pkt := C.aom_codec_get_cx_data(e.codec, &iter)
		if pkt == nil {
			break
		}

		if C.is_frame_packet(pkt) != 0 {
			buf := C.get_frame_buf(pkt)
			sz := C.get_frame_sz(pkt)
			frameData := C.GoBytes(buf, C.int(sz))
			keyframe := C.is_keyframe(pkt) != 0
			pktPts := int64(C.get_frame_pts(pkt))

			timestampUs := pktPts * 1000 / int64(e.fps)

			e.frames = append(e.frames, encodedFrame{
				data:        frameData,
				timestampUs: timestampUs,
				isKeyframe:  keyframe,
			})
		}
	}

	// Build MP4 container
	mp4Data, err := e.buildMP4()
	if err != nil {
		return nil, fmt.Errorf("build mp4: %w", err)
	}

	// Cleanup
	e.cleanup()

	return mp4Data, nil
}

func (e *Encoder) cleanup() {
	if e.rawFrame != nil {
		C.aom_img_free(e.rawFrame)
		C.free(unsafe.Pointer(e.rawFrame))
		e.rawFrame = nil
	}
	if e.codec != nil {
		C.aom_codec_destroy(e.codec)
		C.free(unsafe.Pointer(e.codec))
		e.codec = nil
	}
	if e.cfg != nil {
		C.free(unsafe.Pointer(e.cfg))
		e.cfg = nil
	}
}

// rgbaToYUV420 converts RGBA image to YUV420 format in the raw frame buffer.
func (e *Encoder) rgbaToYUV420(rgba *image.RGBA) {
	width := e.width
	height := e.height

	yStride := int(C.get_plane_stride(e.rawFrame, 0))
	uStride := int(C.get_plane_stride(e.rawFrame, 1))
	vStride := int(C.get_plane_stride(e.rawFrame, 2))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*rgba.Stride + x*4
			r := int(rgba.Pix[idx])
			g := int(rgba.Pix[idx+1])
			b := int(rgba.Pix[idx+2])

			// RGB to YUV conversion
			yVal := ((66*r + 129*g + 25*b + 128) >> 8) + 16
			if yVal > 255 {
				yVal = 255
			}
			if yVal < 0 {
				yVal = 0
			}
			C.set_yuv_pixel(e.rawFrame, 0, C.int(y*yStride+x), C.uchar(yVal))

			if y%2 == 0 && x%2 == 0 {
				uIdx := (y/2)*uStride + (x / 2)
				vIdx := (y/2)*vStride + (x / 2)

				uVal := ((-38*r - 74*g + 112*b + 128) >> 8) + 128
				vVal := ((112*r - 94*g - 18*b + 128) >> 8) + 128

				if uVal > 255 {
					uVal = 255
				}
				if uVal < 0 {
					uVal = 0
				}
				if vVal > 255 {
					vVal = 255
				}
				if vVal < 0 {
					vVal = 0
				}

				C.set_yuv_pixel(e.rawFrame, 1, C.int(uIdx), C.uchar(uVal))
				C.set_yuv_pixel(e.rawFrame, 2, C.int(vIdx), C.uchar(vVal))
			}
		}
	}
}

// Ensure Encoder implements ports.VideoEncoder
var _ ports.VideoEncoder = (*Encoder)(nil)
