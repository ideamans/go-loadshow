// Package av1decoder provides an AV1 video decoder using libaom.
package av1decoder

/*
#cgo pkg-config: aom
#include <aom/aom_decoder.h>
#include <aom/aomdx.h>
#include <stdlib.h>
#include <string.h>

static aom_codec_iface_t* get_av1_decoder_interface() {
    return aom_codec_av1_dx();
}

// Wrapper for aom_codec_dec_init
static aom_codec_err_t init_decoder(aom_codec_ctx_t *ctx, aom_codec_iface_t *iface) {
    return aom_codec_dec_init(ctx, iface, NULL, 0);
}

// Get image plane data
static unsigned char* get_plane(aom_image_t *img, int plane) {
    return img->planes[plane];
}

static int get_stride(aom_image_t *img, int plane) {
    return img->stride[plane];
}

static unsigned int get_width(aom_image_t *img) {
    return img->d_w;
}

static unsigned int get_height(aom_image_t *img) {
    return img->d_h;
}
*/
import "C"

import (
	"fmt"
	"image"
	"unsafe"
)

// Decoder implements AV1 video decoding using libaom.
type Decoder struct {
	codec *C.aom_codec_ctx_t
}

// New creates a new AV1 decoder.
func New() *Decoder {
	return &Decoder{}
}

// Init initializes the decoder.
func (d *Decoder) Init() error {
	d.codec = (*C.aom_codec_ctx_t)(C.malloc(C.sizeof_aom_codec_ctx_t))
	if d.codec == nil {
		return fmt.Errorf("failed to allocate decoder context")
	}
	C.memset(unsafe.Pointer(d.codec), 0, C.sizeof_aom_codec_ctx_t)

	iface := C.get_av1_decoder_interface()
	if res := C.init_decoder(d.codec, iface); res != C.AOM_CODEC_OK {
		C.free(unsafe.Pointer(d.codec))
		return fmt.Errorf("failed to initialize decoder: %d", res)
	}

	return nil
}

// DecodeFrame decodes an AV1 frame and returns an image.
func (d *Decoder) DecodeFrame(data []byte) (image.Image, error) {
	if d.codec == nil {
		return nil, fmt.Errorf("decoder not initialized")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty frame data")
	}

	// Decode frame
	res := C.aom_codec_decode(
		d.codec,
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		nil,
	)
	if res != C.AOM_CODEC_OK {
		return nil, fmt.Errorf("decode failed: %d", res)
	}

	// Get decoded image
	var iter C.aom_codec_iter_t
	img := C.aom_codec_get_frame(d.codec, &iter)
	if img == nil {
		return nil, fmt.Errorf("no frame available")
	}

	// Convert YUV to RGBA
	return d.yuvToRGBA(img), nil
}

// Close releases decoder resources.
func (d *Decoder) Close() {
	if d.codec != nil {
		C.aom_codec_destroy(d.codec)
		C.free(unsafe.Pointer(d.codec))
		d.codec = nil
	}
}

// yuvToRGBA converts YUV420 image to RGBA.
func (d *Decoder) yuvToRGBA(img *C.aom_image_t) *image.RGBA {
	width := int(C.get_width(img))
	height := int(C.get_height(img))

	yPlane := C.get_plane(img, 0)
	uPlane := C.get_plane(img, 1)
	vPlane := C.get_plane(img, 2)

	yStride := int(C.get_stride(img, 0))
	uStride := int(C.get_stride(img, 1))
	vStride := int(C.get_stride(img, 2))

	rgba := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			yIdx := y*yStride + x
			uIdx := (y/2)*uStride + (x / 2)
			vIdx := (y/2)*vStride + (x / 2)

			yVal := int(*(*C.uchar)(unsafe.Pointer(uintptr(unsafe.Pointer(yPlane)) + uintptr(yIdx))))
			uVal := int(*(*C.uchar)(unsafe.Pointer(uintptr(unsafe.Pointer(uPlane)) + uintptr(uIdx))))
			vVal := int(*(*C.uchar)(unsafe.Pointer(uintptr(unsafe.Pointer(vPlane)) + uintptr(vIdx))))

			// YUV to RGB conversion
			c := yVal - 16
			d := uVal - 128
			e := vVal - 128

			r := clamp((298*c + 409*e + 128) >> 8)
			g := clamp((298*c - 100*d - 208*e + 128) >> 8)
			b := clamp((298*c + 516*d + 128) >> 8)

			idx := y*rgba.Stride + x*4
			rgba.Pix[idx] = uint8(r)
			rgba.Pix[idx+1] = uint8(g)
			rgba.Pix[idx+2] = uint8(b)
			rgba.Pix[idx+3] = 255
		}
	}

	return rgba
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}
