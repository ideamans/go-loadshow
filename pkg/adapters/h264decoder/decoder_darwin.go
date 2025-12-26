//go:build darwin

package h264decoder

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework VideoToolbox -framework CoreMedia -framework CoreFoundation -framework CoreVideo

#include <VideoToolbox/VideoToolbox.h>
#include <CoreMedia/CoreMedia.h>
#include <CoreFoundation/CoreFoundation.h>
#include <CoreVideo/CoreVideo.h>
#include <stdlib.h>
#include <string.h>

// Decoder context
typedef struct {
    VTDecompressionSessionRef session;
    CMFormatDescriptionRef formatDesc;

    // Output image
    unsigned char *outputData;
    int outputWidth;
    int outputHeight;
    int outputReady;
} VTDecoderContext;

// Decompression output callback
static void decompressionOutputCallback(void *decompressionOutputRefCon,
                                        void *sourceFrameRefCon,
                                        OSStatus status,
                                        VTDecodeInfoFlags infoFlags,
                                        CVImageBufferRef imageBuffer,
                                        CMTime presentationTimeStamp,
                                        CMTime presentationDuration) {
    VTDecoderContext *ctx = (VTDecoderContext*)decompressionOutputRefCon;
    if (status != noErr || imageBuffer == NULL) {
        ctx->outputReady = 0;
        return;
    }

    CVPixelBufferLockBaseAddress(imageBuffer, kCVPixelBufferLock_ReadOnly);

    size_t width = CVPixelBufferGetWidth(imageBuffer);
    size_t height = CVPixelBufferGetHeight(imageBuffer);

    // Allocate output buffer if needed
    size_t bufferSize = width * height * 4;
    if (ctx->outputData == NULL || ctx->outputWidth != (int)width || ctx->outputHeight != (int)height) {
        if (ctx->outputData) free(ctx->outputData);
        ctx->outputData = (unsigned char*)malloc(bufferSize);
        ctx->outputWidth = (int)width;
        ctx->outputHeight = (int)height;
    }

    if (ctx->outputData == NULL) {
        CVPixelBufferUnlockBaseAddress(imageBuffer, kCVPixelBufferLock_ReadOnly);
        ctx->outputReady = 0;
        return;
    }

    // Check pixel format
    OSType pixelFormat = CVPixelBufferGetPixelFormatType(imageBuffer);

    if (pixelFormat == kCVPixelFormatType_32BGRA) {
        // BGRA to RGBA conversion
        void *baseAddress = CVPixelBufferGetBaseAddress(imageBuffer);
        size_t bytesPerRow = CVPixelBufferGetBytesPerRow(imageBuffer);

        for (size_t y = 0; y < height; y++) {
            unsigned char *src = (unsigned char*)baseAddress + y * bytesPerRow;
            unsigned char *dst = ctx->outputData + y * width * 4;
            for (size_t x = 0; x < width; x++) {
                dst[x * 4 + 0] = src[x * 4 + 2]; // R
                dst[x * 4 + 1] = src[x * 4 + 1]; // G
                dst[x * 4 + 2] = src[x * 4 + 0]; // B
                dst[x * 4 + 3] = src[x * 4 + 3]; // A
            }
        }
    } else if (pixelFormat == kCVPixelFormatType_420YpCbCr8BiPlanarVideoRange ||
               pixelFormat == kCVPixelFormatType_420YpCbCr8BiPlanarFullRange) {
        // NV12 to RGBA conversion
        unsigned char *yPlane = CVPixelBufferGetBaseAddressOfPlane(imageBuffer, 0);
        unsigned char *uvPlane = CVPixelBufferGetBaseAddressOfPlane(imageBuffer, 1);
        size_t yBytesPerRow = CVPixelBufferGetBytesPerRowOfPlane(imageBuffer, 0);
        size_t uvBytesPerRow = CVPixelBufferGetBytesPerRowOfPlane(imageBuffer, 1);

        for (size_t y = 0; y < height; y++) {
            for (size_t x = 0; x < width; x++) {
                int yVal = yPlane[y * yBytesPerRow + x];
                int uVal = uvPlane[(y / 2) * uvBytesPerRow + (x / 2) * 2];
                int vVal = uvPlane[(y / 2) * uvBytesPerRow + (x / 2) * 2 + 1];

                // YUV to RGB conversion
                int c = yVal - 16;
                int d = uVal - 128;
                int e = vVal - 128;

                int r = (298 * c + 409 * e + 128) >> 8;
                int g = (298 * c - 100 * d - 208 * e + 128) >> 8;
                int b = (298 * c + 516 * d + 128) >> 8;

                if (r < 0) r = 0; if (r > 255) r = 255;
                if (g < 0) g = 0; if (g > 255) g = 255;
                if (b < 0) b = 0; if (b > 255) b = 255;

                size_t idx = (y * width + x) * 4;
                ctx->outputData[idx + 0] = (unsigned char)r;
                ctx->outputData[idx + 1] = (unsigned char)g;
                ctx->outputData[idx + 2] = (unsigned char)b;
                ctx->outputData[idx + 3] = 255;
            }
        }
    }

    CVPixelBufferUnlockBaseAddress(imageBuffer, kCVPixelBufferLock_ReadOnly);
    ctx->outputReady = 1;
}

// Parse NAL units from Annex B and extract SPS/PPS
static int extractSPSPPS(unsigned char *data, size_t size,
                         unsigned char **sps, size_t *spsSize,
                         unsigned char **pps, size_t *ppsSize) {
    *sps = NULL;
    *spsSize = 0;
    *pps = NULL;
    *ppsSize = 0;

    size_t offset = 0;
    while (offset < size) {
        // Find start code
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

        // Find next start code or end
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
            if (nalType == 7 && *sps == NULL) { // SPS
                *sps = data + naluStart;
                *spsSize = naluLen;
            } else if (nalType == 8 && *pps == NULL) { // PPS
                *pps = data + naluStart;
                *ppsSize = naluLen;
            }
        }
    }

    return (*sps != NULL && *pps != NULL) ? 0 : -1;
}

// Create decoder context
static VTDecoderContext* vtCreateDecoder() {
    VTDecoderContext *ctx = (VTDecoderContext*)calloc(1, sizeof(VTDecoderContext));
    return ctx;
}

// Create or update decompression session based on SPS/PPS
static int vtUpdateSession(VTDecoderContext *ctx, unsigned char *sps, size_t spsSize,
                           unsigned char *pps, size_t ppsSize) {
    if (!ctx) return -1;

    // Release old session
    if (ctx->session) {
        VTDecompressionSessionInvalidate(ctx->session);
        CFRelease(ctx->session);
        ctx->session = NULL;
    }
    if (ctx->formatDesc) {
        CFRelease(ctx->formatDesc);
        ctx->formatDesc = NULL;
    }

    // Create format description
    const uint8_t *parameterSetPointers[2] = { sps, pps };
    size_t parameterSetSizes[2] = { spsSize, ppsSize };

    OSStatus status = CMVideoFormatDescriptionCreateFromH264ParameterSets(
        kCFAllocatorDefault,
        2,
        parameterSetPointers,
        parameterSetSizes,
        4,  // NAL unit length size
        &ctx->formatDesc);

    if (status != noErr) {
        return -1;
    }

    // Create decompression session
    CFMutableDictionaryRef destinationPixelBufferAttributes = CFDictionaryCreateMutable(
        kCFAllocatorDefault, 0,
        &kCFTypeDictionaryKeyCallBacks,
        &kCFTypeDictionaryValueCallBacks);

    SInt32 pixelFormat = kCVPixelFormatType_32BGRA;
    CFNumberRef pixelFormatNumber = CFNumberCreate(kCFAllocatorDefault, kCFNumberSInt32Type, &pixelFormat);
    CFDictionarySetValue(destinationPixelBufferAttributes, kCVPixelBufferPixelFormatTypeKey, pixelFormatNumber);
    CFRelease(pixelFormatNumber);

    VTDecompressionOutputCallbackRecord callbackRecord;
    callbackRecord.decompressionOutputCallback = decompressionOutputCallback;
    callbackRecord.decompressionOutputRefCon = ctx;

    status = VTDecompressionSessionCreate(
        kCFAllocatorDefault,
        ctx->formatDesc,
        NULL,
        destinationPixelBufferAttributes,
        &callbackRecord,
        &ctx->session);

    CFRelease(destinationPixelBufferAttributes);

    return (status == noErr) ? 0 : -1;
}

// Decode a frame (Annex B format)
static int vtDecodeFrame(VTDecoderContext *ctx, unsigned char *data, size_t size) {
    if (!ctx) return -1;

    ctx->outputReady = 0;

    // Check for SPS/PPS and update session if needed
    unsigned char *sps = NULL, *pps = NULL;
    size_t spsSize = 0, ppsSize = 0;

    if (extractSPSPPS(data, size, &sps, &spsSize, &pps, &ppsSize) == 0) {
        if (vtUpdateSession(ctx, sps, spsSize, pps, ppsSize) != 0) {
            return -1;
        }
    }

    if (!ctx->session) {
        return -1;  // Need SPS/PPS first
    }

    // Convert Annex B to AVCC format for decoding
    // Find video NAL units (not SPS/PPS)
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

        // Find next start code or end
        while (offset < size) {
            if (offset + 3 <= size && data[offset] == 0 && data[offset + 1] == 0 &&
                (data[offset + 2] == 1 || (offset + 4 <= size && data[offset + 2] == 0 && data[offset + 3] == 1))) {
                break;
            }
            offset++;
        }

        size_t naluLen = offset - naluStart;
        if (naluLen == 0) continue;

        unsigned char nalType = data[naluStart] & 0x1F;
        if (nalType == 7 || nalType == 8) continue;  // Skip SPS/PPS

        // Create AVCC format buffer (4-byte length prefix)
        size_t avccSize = 4 + naluLen;
        unsigned char *avccData = (unsigned char*)malloc(avccSize);
        if (!avccData) return -1;

        avccData[0] = (naluLen >> 24) & 0xFF;
        avccData[1] = (naluLen >> 16) & 0xFF;
        avccData[2] = (naluLen >> 8) & 0xFF;
        avccData[3] = naluLen & 0xFF;
        memcpy(avccData + 4, data + naluStart, naluLen);

        // Create CMBlockBuffer
        CMBlockBufferRef blockBuffer = NULL;
        OSStatus status = CMBlockBufferCreateWithMemoryBlock(
            kCFAllocatorDefault,
            avccData,
            avccSize,
            kCFAllocatorDefault,
            NULL,
            0,
            avccSize,
            0,
            &blockBuffer);

        if (status != noErr) {
            free(avccData);
            return -1;
        }

        // Create CMSampleBuffer
        CMSampleBufferRef sampleBuffer = NULL;
        size_t sampleSizeArray[] = { avccSize };

        status = CMSampleBufferCreate(
            kCFAllocatorDefault,
            blockBuffer,
            true,
            NULL,
            NULL,
            ctx->formatDesc,
            1,
            0,
            NULL,
            1,
            sampleSizeArray,
            &sampleBuffer);

        CFRelease(blockBuffer);

        if (status != noErr) {
            return -1;
        }

        // Decode
        VTDecodeFrameFlags flags = 0;
        VTDecodeInfoFlags infoFlags;

        status = VTDecompressionSessionDecodeFrame(
            ctx->session,
            sampleBuffer,
            flags,
            NULL,
            &infoFlags);

        CFRelease(sampleBuffer);

        if (status == noErr) {
            VTDecompressionSessionWaitForAsynchronousFrames(ctx->session);
            if (ctx->outputReady) {
                return 0;  // Success
            }
        }
    }

    return -1;
}

static int vtGetOutputWidth(VTDecoderContext *ctx) {
    return ctx ? ctx->outputWidth : 0;
}

static int vtGetOutputHeight(VTDecoderContext *ctx) {
    return ctx ? ctx->outputHeight : 0;
}

static unsigned char* vtGetOutputData(VTDecoderContext *ctx) {
    return ctx ? ctx->outputData : NULL;
}

static void vtDestroyDecoder(VTDecoderContext *ctx) {
    if (!ctx) return;

    if (ctx->session) {
        VTDecompressionSessionInvalidate(ctx->session);
        CFRelease(ctx->session);
    }
    if (ctx->formatDesc) {
        CFRelease(ctx->formatDesc);
    }
    if (ctx->outputData) {
        free(ctx->outputData);
    }
    free(ctx);
}
*/
import "C"

import (
	"image"
	"unsafe"
)

// videoToolboxDecoder implements H.264 decoding using VideoToolbox on macOS.
type videoToolboxDecoder struct {
	ctx *C.VTDecoderContext
}

func newPlatformDecoder() platformDecoder {
	return &videoToolboxDecoder{}
}

func (d *videoToolboxDecoder) init() error {
	d.ctx = C.vtCreateDecoder()
	if d.ctx == nil {
		return ErrPlatformNotSupported
	}
	return nil
}

func (d *videoToolboxDecoder) decodeFrame(data []byte) (image.Image, error) {
	if d.ctx == nil {
		return nil, ErrNotInitialized
	}

	if len(data) == 0 {
		return nil, ErrDecodeFailed
	}

	result := C.vtDecodeFrame(
		d.ctx,
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
	)

	if result != 0 {
		return nil, ErrDecodeFailed
	}

	width := int(C.vtGetOutputWidth(d.ctx))
	height := int(C.vtGetOutputHeight(d.ctx))
	outputData := C.vtGetOutputData(d.ctx)

	if width == 0 || height == 0 || outputData == nil {
		return nil, ErrDecodeFailed
	}

	// Create Go image from output data
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	pixData := C.GoBytes(unsafe.Pointer(outputData), C.int(width*height*4))
	copy(rgba.Pix, pixData)

	return rgba, nil
}

func (d *videoToolboxDecoder) close() {
	if d.ctx != nil {
		C.vtDestroyDecoder(d.ctx)
		d.ctx = nil
	}
}

// checkPlatformAvailability returns true on macOS as VideoToolbox is always available.
func checkPlatformAvailability() bool {
	return true
}
