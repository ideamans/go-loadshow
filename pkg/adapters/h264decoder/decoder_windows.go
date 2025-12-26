//go:build windows

package h264decoder

/*
#cgo CFLAGS: -DCOBJMACROS
#cgo LDFLAGS: -lmfplat -lmfuuid -lole32 -lmf -lmfreadwrite -lshlwapi

#include <stdint.h>
#include <windows.h>
#include <mfapi.h>
#include <mfidl.h>
#include <mfreadwrite.h>
#include <mferror.h>
#include <mftransform.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// Helper macros for Media Foundation attribute setting
static inline HRESULT MFSetAttributeSize_(IMFMediaType *pType, REFGUID guidKey, UINT32 width, UINT32 height) {
    return IMFMediaType_SetUINT64(pType, guidKey, ((UINT64)width << 32) | height);
}

static inline HRESULT MFSetAttributeRatio_(IMFMediaType *pType, REFGUID guidKey, UINT32 num, UINT32 denom) {
    return IMFMediaType_SetUINT64(pType, guidKey, ((UINT64)num << 32) | denom);
}

#define MFSetAttributeSize(pType, guidKey, width, height) MFSetAttributeSize_(pType, &guidKey, width, height)
#define MFSetAttributeRatio(pType, guidKey, num, denom) MFSetAttributeRatio_(pType, &guidKey, num, denom)

// Decoder context
typedef struct {
    IMFTransform *transform;
    IMFMediaType *inputType;
    IMFMediaType *outputType;
    int width;
    int height;

    // Output image buffer
    unsigned char *outputData;
    int outputWidth;
    int outputHeight;
    int outputReady;

    // SPS/PPS for initialization
    unsigned char *sps;
    size_t spsSize;
    unsigned char *pps;
    size_t ppsSize;
    int initialized;
} MFH264Decoder;

static int mfDecoderInitialized = 0;

static HRESULT initMFDecoder() {
    if (!mfDecoderInitialized) {
        HRESULT hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
        if (FAILED(hr) && hr != RPC_E_CHANGED_MODE) {
            return hr;
        }
        hr = MFStartup(MF_VERSION, MFSTARTUP_NOSOCKET);
        if (SUCCEEDED(hr)) {
            mfDecoderInitialized = 1;
        }
        return hr;
    }
    return S_OK;
}

// Find H.264 decoder MFT
static HRESULT findH264Decoder(IMFTransform **ppTransform) {
    MFT_REGISTER_TYPE_INFO inputType = { MFMediaType_Video, MFVideoFormat_H264 };
    MFT_REGISTER_TYPE_INFO outputType = { MFMediaType_Video, MFVideoFormat_NV12 };

    IMFActivate **ppActivate = NULL;
    UINT32 count = 0;

    HRESULT hr = MFTEnumEx(
        MFT_CATEGORY_VIDEO_DECODER,
        MFT_ENUM_FLAG_SYNCMFT | MFT_ENUM_FLAG_ASYNCMFT | MFT_ENUM_FLAG_HARDWARE | MFT_ENUM_FLAG_SORTANDFILTER,
        &inputType,
        &outputType,
        &ppActivate,
        &count);

    if (FAILED(hr) || count == 0) {
        return E_FAIL;
    }

    hr = IMFActivate_ActivateObject(ppActivate[0], &IID_IMFTransform, (void**)ppTransform);

    for (UINT32 i = 0; i < count; i++) {
        IMFActivate_Release(ppActivate[i]);
    }
    CoTaskMemFree(ppActivate);

    return hr;
}

// Parse NAL units from Annex B and extract SPS/PPS
static int mfExtractSPSPPS(unsigned char *data, size_t size,
                           unsigned char **sps, size_t *spsSize,
                           unsigned char **pps, size_t *ppsSize) {
    *sps = NULL;
    *spsSize = 0;
    *pps = NULL;
    *ppsSize = 0;

    size_t offset = 0;
    while (offset < size) {
        size_t startCodeLen = 0;
        if (offset + 3 <= size && data[offset] == 0 && data[offset + 1] == 0 && data[offset + 2] == 1) {
            startCodeLen = 3;
        } else if (offset + 4 <= size && data[offset] == 0 && data[offset + 1] == 0 &&
                   data[offset + 2] == 0 && data[offset + 3] == 1) {
            startCodeLen = 4;
        }

        if (startCodeLen == 0) {
            offset++;
            continue;
        }

        offset += startCodeLen;
        size_t naluStart = offset;

        while (offset < size) {
            if (offset + 3 <= size && data[offset] == 0 && data[offset + 1] == 0 &&
                (data[offset + 2] == 1 || (offset + 4 <= size && data[offset + 2] == 0 && data[offset + 3] == 1))) {
                break;
            }
            offset++;
        }

        size_t naluLen = offset - naluStart;
        if (naluLen > 0) {
            unsigned char nalType = data[naluStart] & 0x1F;
            if (nalType == 7 && *sps == NULL) {
                *sps = (unsigned char*)malloc(naluLen);
                if (*sps) {
                    memcpy(*sps, data + naluStart, naluLen);
                    *spsSize = naluLen;
                }
            } else if (nalType == 8 && *pps == NULL) {
                *pps = (unsigned char*)malloc(naluLen);
                if (*pps) {
                    memcpy(*pps, data + naluStart, naluLen);
                    *ppsSize = naluLen;
                }
            }
        }
    }

    return (*sps != NULL && *pps != NULL) ? 0 : -1;
}

static MFH264Decoder* mfCreateDecoder() {
    if (FAILED(initMFDecoder())) {
        return NULL;
    }

    MFH264Decoder *ctx = (MFH264Decoder*)calloc(1, sizeof(MFH264Decoder));
    if (!ctx) return NULL;

    return ctx;
}

static int mfInitializeDecoderWithDimensions(MFH264Decoder *ctx, int width, int height) {
    if (!ctx || width <= 0 || height <= 0) return -1;

    ctx->width = width;
    ctx->height = height;

    // Find decoder
    HRESULT hr = findH264Decoder(&ctx->transform);
    if (FAILED(hr)) return -1;

    // Create input media type
    hr = MFCreateMediaType(&ctx->inputType);
    if (FAILED(hr)) return -1;

    IMFMediaType_SetGUID(ctx->inputType, &MF_MT_MAJOR_TYPE, &MFMediaType_Video);
    IMFMediaType_SetGUID(ctx->inputType, &MF_MT_SUBTYPE, &MFVideoFormat_H264);
    MFSetAttributeSize(ctx->inputType, MF_MT_FRAME_SIZE, ctx->width, ctx->height);
    MFSetAttributeRatio(ctx->inputType, MF_MT_FRAME_RATE, 30, 1);
    MFSetAttributeRatio(ctx->inputType, MF_MT_PIXEL_ASPECT_RATIO, 1, 1);
    IMFMediaType_SetUINT32(ctx->inputType, &MF_MT_INTERLACE_MODE, MFVideoInterlace_Progressive);

    hr = IMFTransform_SetInputType(ctx->transform, 0, ctx->inputType, 0);
    if (FAILED(hr)) return -1;

    // Create output media type
    hr = MFCreateMediaType(&ctx->outputType);
    if (FAILED(hr)) return -1;

    IMFMediaType_SetGUID(ctx->outputType, &MF_MT_MAJOR_TYPE, &MFMediaType_Video);
    IMFMediaType_SetGUID(ctx->outputType, &MF_MT_SUBTYPE, &MFVideoFormat_NV12);
    MFSetAttributeSize(ctx->outputType, MF_MT_FRAME_SIZE, ctx->width, ctx->height);
    MFSetAttributeRatio(ctx->outputType, MF_MT_FRAME_RATE, 30, 1);
    MFSetAttributeRatio(ctx->outputType, MF_MT_PIXEL_ASPECT_RATIO, 1, 1);
    IMFMediaType_SetUINT32(ctx->outputType, &MF_MT_INTERLACE_MODE, MFVideoInterlace_Progressive);

    hr = IMFTransform_SetOutputType(ctx->transform, 0, ctx->outputType, 0);
    if (FAILED(hr)) return -1;

    hr = IMFTransform_ProcessMessage(ctx->transform, MFT_MESSAGE_NOTIFY_BEGIN_STREAMING, 0);
    if (FAILED(hr)) return -1;

    ctx->initialized = 1;
    return 0;
}

// NV12 to RGBA conversion
static void nv12ToRGBA(unsigned char *nv12, int width, int height, int stride, unsigned char *rgba) {
    unsigned char *yPlane = nv12;
    unsigned char *uvPlane = nv12 + stride * height;

    for (int y = 0; y < height; y++) {
        for (int x = 0; x < width; x++) {
            int yVal = yPlane[y * stride + x];
            int uVal = uvPlane[(y / 2) * stride + (x / 2) * 2];
            int vVal = uvPlane[(y / 2) * stride + (x / 2) * 2 + 1];

            int c = yVal - 16;
            int d = uVal - 128;
            int e = vVal - 128;

            int r = (298 * c + 409 * e + 128) >> 8;
            int g = (298 * c - 100 * d - 208 * e + 128) >> 8;
            int b = (298 * c + 516 * d + 128) >> 8;

            if (r < 0) r = 0; if (r > 255) r = 255;
            if (g < 0) g = 0; if (g > 255) g = 255;
            if (b < 0) b = 0; if (b > 255) b = 255;

            int idx = (y * width + x) * 4;
            rgba[idx + 0] = (unsigned char)r;
            rgba[idx + 1] = (unsigned char)g;
            rgba[idx + 2] = (unsigned char)b;
            rgba[idx + 3] = 255;
        }
    }
}

static int mfDecodeFrame(MFH264Decoder *ctx, unsigned char *data, size_t size) {
    if (!ctx) return -1;

    ctx->outputReady = 0;

    if (!ctx->initialized || !ctx->transform) return -1;

    // Create input sample with Annex B data
    IMFSample *inputSample = NULL;
    IMFMediaBuffer *inputBuffer = NULL;

    HRESULT hr = MFCreateMemoryBuffer((DWORD)size, &inputBuffer);
    if (FAILED(hr)) return -1;

    BYTE *bufferData = NULL;
    hr = IMFMediaBuffer_Lock(inputBuffer, &bufferData, NULL, NULL);
    if (SUCCEEDED(hr)) {
        memcpy(bufferData, data, size);
        IMFMediaBuffer_Unlock(inputBuffer);
        IMFMediaBuffer_SetCurrentLength(inputBuffer, (DWORD)size);
    }

    hr = MFCreateSample(&inputSample);
    if (FAILED(hr)) {
        IMFMediaBuffer_Release(inputBuffer);
        return -1;
    }

    hr = IMFSample_AddBuffer(inputSample, inputBuffer);
    IMFMediaBuffer_Release(inputBuffer);

    if (FAILED(hr)) {
        IMFSample_Release(inputSample);
        return -1;
    }

    // Process input
    hr = IMFTransform_ProcessInput(ctx->transform, 0, inputSample, 0);
    IMFSample_Release(inputSample);

    if (FAILED(hr) && hr != MF_E_NOTACCEPTING) {
        return -1;
    }

    // Get output
    MFT_OUTPUT_DATA_BUFFER outputData = {0};
    DWORD status = 0;

    // Create output sample
    IMFSample *outputSample = NULL;
    IMFMediaBuffer *outputBuffer = NULL;

    DWORD outputSize = ctx->width * ctx->height * 3 / 2; // NV12 size
    hr = MFCreateMemoryBuffer(outputSize, &outputBuffer);
    if (FAILED(hr)) return -1;

    hr = MFCreateSample(&outputSample);
    if (FAILED(hr)) {
        IMFMediaBuffer_Release(outputBuffer);
        return -1;
    }

    IMFSample_AddBuffer(outputSample, outputBuffer);
    outputData.pSample = outputSample;

    hr = IMFTransform_ProcessOutput(ctx->transform, 0, 1, &outputData, &status);

    if (SUCCEEDED(hr)) {
        // Extract decoded frame
        BYTE *outData = NULL;
        DWORD outLen = 0;

        hr = IMFMediaBuffer_Lock(outputBuffer, &outData, NULL, &outLen);
        if (SUCCEEDED(hr)) {
            // Allocate RGBA buffer
            size_t rgbaSize = ctx->width * ctx->height * 4;
            if (ctx->outputData == NULL || ctx->outputWidth != ctx->width || ctx->outputHeight != ctx->height) {
                if (ctx->outputData) free(ctx->outputData);
                ctx->outputData = (unsigned char*)malloc(rgbaSize);
                ctx->outputWidth = ctx->width;
                ctx->outputHeight = ctx->height;
            }

            if (ctx->outputData) {
                nv12ToRGBA(outData, ctx->width, ctx->height, ctx->width, ctx->outputData);
                ctx->outputReady = 1;
            }

            IMFMediaBuffer_Unlock(outputBuffer);
        }
    }

    IMFMediaBuffer_Release(outputBuffer);
    IMFSample_Release(outputSample);

    return ctx->outputReady ? 0 : -1;
}

static int mfGetOutputWidth(MFH264Decoder *ctx) {
    return ctx ? ctx->outputWidth : 0;
}

static int mfGetOutputHeight(MFH264Decoder *ctx) {
    return ctx ? ctx->outputHeight : 0;
}

static unsigned char* mfGetOutputData(MFH264Decoder *ctx) {
    return ctx ? ctx->outputData : NULL;
}

static void mfDestroyDecoder(MFH264Decoder *ctx) {
    if (!ctx) return;

    if (ctx->transform) {
        IMFTransform_ProcessMessage(ctx->transform, MFT_MESSAGE_NOTIFY_END_STREAMING, 0);
        IMFTransform_Release(ctx->transform);
    }
    if (ctx->inputType) IMFMediaType_Release(ctx->inputType);
    if (ctx->outputType) IMFMediaType_Release(ctx->outputType);
    if (ctx->sps) free(ctx->sps);
    if (ctx->pps) free(ctx->pps);
    if (ctx->outputData) free(ctx->outputData);
    free(ctx);
}
*/
import "C"

import (
	"image"
	"unsafe"

	"github.com/Eyevinn/mp4ff/avc"
)

// mediaFoundationDecoder implements H.264 decoding using Media Foundation on Windows.
type mediaFoundationDecoder struct {
	ctx         *C.MFH264Decoder
	initialized bool
}

func newPlatformDecoder() platformDecoder {
	return &mediaFoundationDecoder{}
}

func (d *mediaFoundationDecoder) init() error {
	d.ctx = C.mfCreateDecoder()
	if d.ctx == nil {
		return ErrPlatformNotSupported
	}
	return nil
}

func (d *mediaFoundationDecoder) decodeFrame(data []byte) (image.Image, error) {
	if d.ctx == nil {
		return nil, ErrNotInitialized
	}

	if len(data) == 0 {
		return nil, ErrDecodeFailed
	}

	// Initialize decoder on first frame with SPS
	if !d.initialized {
		width, height, err := d.parseSPSDimensions(data)
		if err != nil {
			return nil, ErrDecodeFailed
		}

		result := C.mfInitializeDecoderWithDimensions(d.ctx, C.int(width), C.int(height))
		if result != 0 {
			return nil, ErrDecodeFailed
		}
		d.initialized = true
	}

	result := C.mfDecodeFrame(
		d.ctx,
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
	)

	if result != 0 {
		return nil, ErrDecodeFailed
	}

	width := int(C.mfGetOutputWidth(d.ctx))
	height := int(C.mfGetOutputHeight(d.ctx))
	outputData := C.mfGetOutputData(d.ctx)

	if width == 0 || height == 0 || outputData == nil {
		return nil, ErrDecodeFailed
	}

	// Create Go image from output data
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	pixData := C.GoBytes(unsafe.Pointer(outputData), C.int(width*height*4))
	copy(rgba.Pix, pixData)

	return rgba, nil
}

func (d *mediaFoundationDecoder) close() {
	if d.ctx != nil {
		C.mfDestroyDecoder(d.ctx)
		d.ctx = nil
	}
	d.initialized = false
}

// parseSPSDimensions extracts video dimensions from SPS NAL unit in Annex B data.
func (d *mediaFoundationDecoder) parseSPSDimensions(data []byte) (int, int, error) {
	// Extract NAL units from Annex B format
	nalus := avc.ExtractNalusFromByteStream(data)

	for _, nalu := range nalus {
		if len(nalu) == 0 {
			continue
		}

		naluType := avc.GetNaluType(nalu[0])
		if naluType == avc.NALU_SPS {
			// Parse SPS to get dimensions
			sps, err := avc.ParseSPSNALUnit(nalu, false)
			if err != nil {
				continue
			}

			return int(sps.Width), int(sps.Height), nil
		}
	}

	return 0, 0, ErrDecodeFailed
}

// checkPlatformAvailability returns true on Windows as Media Foundation is always available.
func checkPlatformAvailability() bool {
	return true
}
