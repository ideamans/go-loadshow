//go:build windows

package h264encoder

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
#include <codecapi.h>
#include <shlwapi.h>
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

// Encoded frame buffer
typedef struct {
    unsigned char *data;
    size_t size;
    int64_t timestampUs;
    int isKeyframe;
} MFEncodedFrame;

typedef struct {
    IMFTransform *transform;
    IMFMediaType *inputType;
    IMFMediaType *outputType;
    int width;
    int height;
    double fps;
    int64_t frameIndex;

    // Output frame storage
    MFEncodedFrame *frames;
    int frameCount;
    int frameCapacity;

    // SPS/PPS extracted from codec private data
    unsigned char *sps;
    size_t spsSize;
    unsigned char *pps;
    size_t ppsSize;
} MFH264Encoder;

// Forward declarations
static int processOutput(MFH264Encoder *enc, int64_t timestampUs);
static int encodeFrameInternal(MFH264Encoder *enc, unsigned char *rgbaData, int64_t timestampUs, int forceKeyframe);
static int flushEncoderInternal(MFH264Encoder *enc);

static int mfInitialized = 0;

static HRESULT initMF() {
    if (!mfInitialized) {
        HRESULT hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
        if (FAILED(hr) && hr != RPC_E_CHANGED_MODE) {
            return hr;
        }
        hr = MFStartup(MF_VERSION, MFSTARTUP_NOSOCKET);
        if (SUCCEEDED(hr)) {
            mfInitialized = 1;
        }
        return hr;
    }
    return S_OK;
}

// Find H.264 encoder MFT
static HRESULT findH264Encoder(IMFTransform **ppTransform) {
    MFT_REGISTER_TYPE_INFO inputType = { MFMediaType_Video, MFVideoFormat_NV12 };
    MFT_REGISTER_TYPE_INFO outputType = { MFMediaType_Video, MFVideoFormat_H264 };

    IMFActivate **ppActivate = NULL;
    UINT32 count = 0;

    HRESULT hr = MFTEnumEx(
        MFT_CATEGORY_VIDEO_ENCODER,
        MFT_ENUM_FLAG_SYNCMFT | MFT_ENUM_FLAG_ASYNCMFT | MFT_ENUM_FLAG_HARDWARE | MFT_ENUM_FLAG_SORTANDFILTER,
        &inputType,
        &outputType,
        &ppActivate,
        &count);

    if (FAILED(hr) || count == 0) {
        return E_FAIL;
    }

    // Use first available encoder
    hr = IMFActivate_ActivateObject(ppActivate[0], &IID_IMFTransform, (void**)ppTransform);

    // Release activates
    for (UINT32 i = 0; i < count; i++) {
        IMFActivate_Release(ppActivate[i]);
    }
    CoTaskMemFree(ppActivate);

    return hr;
}

static MFH264Encoder* createEncoder(int width, int height, double fps, int bitrate, int quality) {
    HRESULT hr = initMF();
    if (FAILED(hr)) {
        return NULL;
    }

    MFH264Encoder *enc = (MFH264Encoder*)calloc(1, sizeof(MFH264Encoder));
    if (!enc) return NULL;

    enc->width = width;
    enc->height = height;
    enc->fps = fps;
    enc->frameIndex = 0;
    enc->frameCapacity = 10000;
    enc->frames = (MFEncodedFrame*)calloc(enc->frameCapacity, sizeof(MFEncodedFrame));
    if (!enc->frames) {
        free(enc);
        return NULL;
    }

    // Find H.264 encoder
    hr = findH264Encoder(&enc->transform);
    if (FAILED(hr) || !enc->transform) {
        free(enc->frames);
        free(enc);
        return NULL;
    }

    // Create output media type (H.264)
    hr = MFCreateMediaType(&enc->outputType);
    if (FAILED(hr)) goto cleanup;

    IMFMediaType_SetGUID(enc->outputType, &MF_MT_MAJOR_TYPE, &MFMediaType_Video);
    IMFMediaType_SetGUID(enc->outputType, &MF_MT_SUBTYPE, &MFVideoFormat_H264);
    MFSetAttributeSize(enc->outputType, MF_MT_FRAME_SIZE, width, height);
    MFSetAttributeRatio(enc->outputType, MF_MT_FRAME_RATE, (UINT32)(fps * 1000), 1000);
    IMFMediaType_SetUINT32(enc->outputType, &MF_MT_INTERLACE_MODE, MFVideoInterlace_Progressive);
    MFSetAttributeRatio(enc->outputType, MF_MT_PIXEL_ASPECT_RATIO, 1, 1);

    // Set bitrate
    UINT32 avgBitrate = bitrate > 0 ? bitrate * 1000 : (UINT32)(width * height * fps * 0.1);
    IMFMediaType_SetUINT32(enc->outputType, &MF_MT_AVG_BITRATE, avgBitrate);

    // Set H.264 profile (Baseline for compatibility)
    IMFMediaType_SetUINT32(enc->outputType, &MF_MT_MPEG2_PROFILE, eAVEncH264VProfile_Base);
    IMFMediaType_SetUINT32(enc->outputType, &MF_MT_MPEG2_LEVEL, eAVEncH264VLevel3_1);

    hr = IMFTransform_SetOutputType(enc->transform, 0, enc->outputType, 0);
    if (FAILED(hr)) goto cleanup;

    // Create input media type (NV12)
    hr = MFCreateMediaType(&enc->inputType);
    if (FAILED(hr)) goto cleanup;

    IMFMediaType_SetGUID(enc->inputType, &MF_MT_MAJOR_TYPE, &MFMediaType_Video);
    IMFMediaType_SetGUID(enc->inputType, &MF_MT_SUBTYPE, &MFVideoFormat_NV12);
    MFSetAttributeSize(enc->inputType, MF_MT_FRAME_SIZE, width, height);
    MFSetAttributeRatio(enc->inputType, MF_MT_FRAME_RATE, (UINT32)(fps * 1000), 1000);
    IMFMediaType_SetUINT32(enc->inputType, &MF_MT_INTERLACE_MODE, MFVideoInterlace_Progressive);
    MFSetAttributeRatio(enc->inputType, MF_MT_PIXEL_ASPECT_RATIO, 1, 1);

    hr = IMFTransform_SetInputType(enc->transform, 0, enc->inputType, 0);
    if (FAILED(hr)) goto cleanup;

    // Start streaming
    hr = IMFTransform_ProcessMessage(enc->transform, MFT_MESSAGE_NOTIFY_BEGIN_STREAMING, 0);
    if (FAILED(hr)) goto cleanup;

    hr = IMFTransform_ProcessMessage(enc->transform, MFT_MESSAGE_NOTIFY_START_OF_STREAM, 0);
    if (FAILED(hr)) goto cleanup;

    return enc;

cleanup:
    if (enc->outputType) IMFMediaType_Release(enc->outputType);
    if (enc->inputType) IMFMediaType_Release(enc->inputType);
    if (enc->transform) IMFTransform_Release(enc->transform);
    free(enc->frames);
    free(enc);
    return NULL;
}

// Convert RGBA to NV12
static void rgbaToNV12(unsigned char *rgba, unsigned char *nv12, int width, int height) {
    int yPlaneSize = width * height;

    // Y plane
    for (int y = 0; y < height; y++) {
        for (int x = 0; x < width; x++) {
            int rgbaIdx = (y * width + x) * 4;
            int r = rgba[rgbaIdx];
            int g = rgba[rgbaIdx + 1];
            int b = rgba[rgbaIdx + 2];

            int yVal = ((66 * r + 129 * g + 25 * b + 128) >> 8) + 16;
            if (yVal > 255) yVal = 255;
            if (yVal < 0) yVal = 0;
            nv12[y * width + x] = (unsigned char)yVal;
        }
    }

    // UV plane (interleaved)
    unsigned char *uv = nv12 + yPlaneSize;
    for (int y = 0; y < height; y += 2) {
        for (int x = 0; x < width; x += 2) {
            int rgbaIdx = (y * width + x) * 4;
            int r = rgba[rgbaIdx];
            int g = rgba[rgbaIdx + 1];
            int b = rgba[rgbaIdx + 2];

            int uVal = ((-38 * r - 74 * g + 112 * b + 128) >> 8) + 128;
            int vVal = ((112 * r - 94 * g - 18 * b + 128) >> 8) + 128;

            if (uVal > 255) uVal = 255;
            if (uVal < 0) uVal = 0;
            if (vVal > 255) vVal = 255;
            if (vVal < 0) vVal = 0;

            int uvIdx = (y / 2) * width + x;
            uv[uvIdx] = (unsigned char)uVal;
            uv[uvIdx + 1] = (unsigned char)vVal;
        }
    }
}

static int processOutput(MFH264Encoder *enc, int64_t timestampUs) {
    MFT_OUTPUT_DATA_BUFFER outputBuffer = {0};
    DWORD status = 0;

    // Allocate output sample
    IMFSample *outputSample = NULL;
    IMFMediaBuffer *outputMediaBuffer = NULL;

    HRESULT hr = MFCreateSample(&outputSample);
    if (FAILED(hr)) return -1;

    DWORD bufferSize = enc->width * enc->height * 2;  // Should be enough for compressed data
    hr = MFCreateMemoryBuffer(bufferSize, &outputMediaBuffer);
    if (FAILED(hr)) {
        IMFSample_Release(outputSample);
        return -1;
    }

    hr = IMFSample_AddBuffer(outputSample, outputMediaBuffer);
    IMFMediaBuffer_Release(outputMediaBuffer);
    if (FAILED(hr)) {
        IMFSample_Release(outputSample);
        return -1;
    }

    outputBuffer.pSample = outputSample;

    hr = IMFTransform_ProcessOutput(enc->transform, 0, 1, &outputBuffer, &status);

    if (hr == MF_E_TRANSFORM_NEED_MORE_INPUT) {
        IMFSample_Release(outputSample);
        return 0;  // Need more input, not an error
    }

    if (FAILED(hr)) {
        IMFSample_Release(outputSample);
        return -1;
    }

    // Get output data
    IMFMediaBuffer *buffer = NULL;
    hr = IMFSample_ConvertToContiguousBuffer(outputBuffer.pSample, &buffer);
    if (FAILED(hr)) {
        IMFSample_Release(outputSample);
        return -1;
    }

    BYTE *data = NULL;
    DWORD dataLen = 0;
    hr = IMFMediaBuffer_Lock(buffer, &data, NULL, &dataLen);
    if (FAILED(hr)) {
        IMFMediaBuffer_Release(buffer);
        IMFSample_Release(outputSample);
        return -1;
    }

    // Check if keyframe
    UINT32 isKeyframe = 0;
    IMFSample_GetUINT32(outputBuffer.pSample, &MFSampleExtension_CleanPoint, &isKeyframe);

    // Store frame
    if (enc->frameCount < enc->frameCapacity) {
        MFEncodedFrame *frame = &enc->frames[enc->frameCount];
        frame->data = (unsigned char*)malloc(dataLen);
        if (frame->data) {
            memcpy(frame->data, data, dataLen);
            frame->size = dataLen;
            frame->timestampUs = timestampUs;
            frame->isKeyframe = isKeyframe ? 1 : 0;
            enc->frameCount++;
        }
    }

    IMFMediaBuffer_Unlock(buffer);
    IMFMediaBuffer_Release(buffer);
    IMFSample_Release(outputSample);

    return 1;  // Got output
}

static int encodeFrameInternal(MFH264Encoder *enc, unsigned char *rgbaData, int64_t timestampUs, int forceKeyframe) {
    if (!enc || !enc->transform) return -1;

    // Create input buffer
    DWORD nv12Size = enc->width * enc->height * 3 / 2;
    IMFMediaBuffer *inputBuffer = NULL;
    HRESULT hr = MFCreateMemoryBuffer(nv12Size, &inputBuffer);
    if (FAILED(hr)) return -1;

    BYTE *bufferData = NULL;
    hr = IMFMediaBuffer_Lock(inputBuffer, &bufferData, NULL, NULL);
    if (FAILED(hr)) {
        IMFMediaBuffer_Release(inputBuffer);
        return -1;
    }

    // Convert RGBA to NV12
    rgbaToNV12(rgbaData, bufferData, enc->width, enc->height);

    IMFMediaBuffer_Unlock(inputBuffer);
    IMFMediaBuffer_SetCurrentLength(inputBuffer, nv12Size);

    // Create sample
    IMFSample *inputSample = NULL;
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

    // Set timestamp (100-nanosecond units)
    LONGLONG sampleTime = timestampUs * 10;
    IMFSample_SetSampleTime(inputSample, sampleTime);

    // Set duration
    LONGLONG duration = (LONGLONG)(10000000.0 / enc->fps);
    IMFSample_SetSampleDuration(inputSample, duration);

    // Process input
    hr = IMFTransform_ProcessInput(enc->transform, 0, inputSample, 0);
    IMFSample_Release(inputSample);

    if (FAILED(hr)) return -1;

    // Try to get output
    while (processOutput(enc, timestampUs) > 0) {
        // Keep getting output until none available
    }

    enc->frameIndex++;
    return 0;
}

static int flushEncoderInternal(MFH264Encoder *enc) {
    if (!enc || !enc->transform) return -1;

    // Signal end of stream
    IMFTransform_ProcessMessage(enc->transform, MFT_MESSAGE_NOTIFY_END_OF_STREAM, 0);
    IMFTransform_ProcessMessage(enc->transform, MFT_MESSAGE_COMMAND_DRAIN, 0);

    // Get remaining output
    while (processOutput(enc, 0) > 0) {
        // Keep getting output
    }

    return 0;
}

static void destroyEncoder(MFH264Encoder *enc) {
    if (!enc) return;

    if (enc->transform) {
        IMFTransform_ProcessMessage(enc->transform, MFT_MESSAGE_NOTIFY_END_STREAMING, 0);
        IMFTransform_Release(enc->transform);
    }
    if (enc->inputType) IMFMediaType_Release(enc->inputType);
    if (enc->outputType) IMFMediaType_Release(enc->outputType);

    if (enc->frames) {
        for (int i = 0; i < enc->frameCount; i++) {
            free(enc->frames[i].data);
        }
        free(enc->frames);
    }
    if (enc->sps) free(enc->sps);
    if (enc->pps) free(enc->pps);

    free(enc);
}

static int getFrameCount(MFH264Encoder *enc) {
    return enc ? enc->frameCount : 0;
}

static MFEncodedFrame* getFrame(MFH264Encoder *enc, int index) {
    if (!enc || index < 0 || index >= enc->frameCount) return NULL;
    return &enc->frames[index];
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

// mediaFoundationEncoder implements H.264 encoding using Media Foundation on Windows.
type mediaFoundationEncoder struct {
	encoder *C.MFH264Encoder
	width   int
	height  int
	fps     float64

	mu          sync.Mutex
	firstFrame  bool
	initialized bool
}

func newPlatformEncoder() platformEncoder {
	return &mediaFoundationEncoder{}
}

func (e *mediaFoundationEncoder) init(width, height int, fps float64, opts ports.EncoderOptions) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.width = width
	e.height = height
	e.fps = fps
	e.firstFrame = true

	encoder := C.createEncoder(
		C.int(width),
		C.int(height),
		C.double(fps),
		C.int(opts.Bitrate),
		C.int(opts.Quality),
	)

	if encoder == nil {
		return fmt.Errorf("failed to create Media Foundation H.264 encoder")
	}

	e.encoder = encoder
	e.initialized = true

	return nil
}

func (e *mediaFoundationEncoder) encodeFrame(img image.Image, timestampMs int) ([]encodedFrame, error) {
	if !e.initialized {
		return nil, ErrNotInitialized
	}

	// Convert image to RGBA
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	timestampUs := int64(timestampMs) * 1000
	forceKeyframe := 0
	if e.firstFrame {
		forceKeyframe = 1
		e.firstFrame = false
	}

	result := C.encodeFrameInternal(
		e.encoder,
		(*C.uchar)(unsafe.Pointer(&rgba.Pix[0])),
		C.int64_t(timestampUs),
		C.int(forceKeyframe),
	)

	if result != 0 {
		return nil, ErrEncodingFailed
	}

	return nil, nil
}

func (e *mediaFoundationEncoder) flush() ([]encodedFrame, error) {
	if !e.initialized {
		return nil, nil
	}

	C.flushEncoderInternal(e.encoder)

	// Collect all frames
	frameCount := int(C.getFrameCount(e.encoder))
	frames := make([]encodedFrame, 0, frameCount)

	for i := 0; i < frameCount; i++ {
		cFrame := C.getFrame(e.encoder, C.int(i))
		if cFrame == nil {
			continue
		}

		data := C.GoBytes(unsafe.Pointer(cFrame.data), C.int(cFrame.size))
		frames = append(frames, encodedFrame{
			data:        data,
			timestampUs: int64(cFrame.timestampUs),
			isKeyframe:  cFrame.isKeyframe != 0,
		})
	}

	return frames, nil
}

func (e *mediaFoundationEncoder) close() {
	if e.encoder != nil {
		C.destroyEncoder(e.encoder)
		e.encoder = nil
	}
	e.initialized = false
}

// checkPlatformAvailability returns true on Windows as Media Foundation is always available.
func checkPlatformAvailability() bool {
	return true
}
