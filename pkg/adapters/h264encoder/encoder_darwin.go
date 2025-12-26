//go:build darwin

package h264encoder

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework VideoToolbox -framework CoreMedia -framework CoreFoundation -framework CoreVideo

#include <VideoToolbox/VideoToolbox.h>
#include <CoreMedia/CoreMedia.h>
#include <CoreFoundation/CoreFoundation.h>
#include <CoreVideo/CoreVideo.h>
#include <stdlib.h>
#include <string.h>

// Encoded frame data
typedef struct {
    unsigned char *data;
    size_t size;
    int64_t timestampUs;
    int isKeyframe;
} VTEncodedFrame;

// Frame storage
typedef struct {
    VTEncodedFrame *frames;
    int count;
    int capacity;
} VTFrameStorage;

// Encoder context
typedef struct {
    VTCompressionSessionRef session;
    VTFrameStorage storage;
    int width;
    int height;
    double fps;
} VTEncoderContext;

// Forward declaration of callback
static void compressionOutputCallback(void *outputCallbackRefCon,
                                      void *sourceFrameRefCon,
                                      OSStatus status,
                                      VTEncodeInfoFlags infoFlags,
                                      CMSampleBufferRef sampleBuffer);

// Add frame to storage
static void addFrameToStorage(VTFrameStorage *storage, unsigned char *data, size_t size,
                              int64_t timestampUs, int isKeyframe) {
    if (storage->count >= storage->capacity) {
        int newCapacity = storage->capacity * 2;
        if (newCapacity < 100) newCapacity = 100;
        VTEncodedFrame *newFrames = (VTEncodedFrame*)realloc(storage->frames, newCapacity * sizeof(VTEncodedFrame));
        if (!newFrames) return;
        storage->frames = newFrames;
        storage->capacity = newCapacity;
    }

    VTEncodedFrame *frame = &storage->frames[storage->count];
    frame->data = (unsigned char*)malloc(size);
    if (!frame->data) return;
    memcpy(frame->data, data, size);
    frame->size = size;
    frame->timestampUs = timestampUs;
    frame->isKeyframe = isKeyframe;
    storage->count++;
}

// Get SPS/PPS from format description and write to buffer
static int extractSPSPPSAndData(CMSampleBufferRef sampleBuffer, unsigned char **outData, size_t *outSize,
                                int *isKeyframe, int64_t *timestampUs) {
    // Get format description
    CMFormatDescriptionRef formatDesc = CMSampleBufferGetFormatDescription(sampleBuffer);

    // Check if keyframe
    CFArrayRef attachments = CMSampleBufferGetSampleAttachmentsArray(sampleBuffer, false);
    *isKeyframe = 1;
    if (attachments != NULL && CFArrayGetCount(attachments) > 0) {
        CFDictionaryRef attachment = CFArrayGetValueAtIndex(attachments, 0);
        CFBooleanRef notSync = CFDictionaryGetValue(attachment, kCMSampleAttachmentKey_NotSync);
        if (notSync != NULL && CFBooleanGetValue(notSync)) {
            *isKeyframe = 0;
        }
    }

    // Get timestamp
    CMTime pts = CMSampleBufferGetPresentationTimeStamp(sampleBuffer);
    *timestampUs = (int64_t)(CMTimeGetSeconds(pts) * 1000000);

    // Get data buffer
    CMBlockBufferRef blockBuffer = CMSampleBufferGetDataBuffer(sampleBuffer);
    if (blockBuffer == NULL) {
        return -1;
    }

    size_t totalLength = 0;
    char *dataPtr = NULL;
    OSStatus status = CMBlockBufferGetDataPointer(blockBuffer, 0, NULL, &totalLength, &dataPtr);
    if (status != noErr || dataPtr == NULL) {
        return -1;
    }

    // For keyframes, prepend SPS and PPS
    size_t spsSize = 0, ppsSize = 0;
    const uint8_t *sps = NULL, *pps = NULL;

    if (*isKeyframe && formatDesc != NULL) {
        size_t paramCount = 0;
        CMVideoFormatDescriptionGetH264ParameterSetAtIndex(formatDesc, 0, NULL, NULL, &paramCount, NULL);

        for (size_t i = 0; i < paramCount; i++) {
            const uint8_t *param = NULL;
            size_t paramSize = 0;
            status = CMVideoFormatDescriptionGetH264ParameterSetAtIndex(formatDesc, i, &param, &paramSize, NULL, NULL);
            if (status == noErr && param != NULL && paramSize > 0) {
                uint8_t nalType = param[0] & 0x1F;
                if (nalType == 7) { // SPS
                    sps = param;
                    spsSize = paramSize;
                } else if (nalType == 8) { // PPS
                    pps = param;
                    ppsSize = paramSize;
                }
            }
        }
    }

    // Calculate output size
    size_t extraSize = 0;
    if (*isKeyframe && sps != NULL && pps != NULL) {
        extraSize = 4 + spsSize + 4 + ppsSize;
    }

    // Count NALUs in AVCC format and calculate size
    size_t offset = 0;
    size_t naluBytes = 0;
    while (offset + 4 <= totalLength) {
        uint32_t naluLen = ((uint8_t)dataPtr[offset] << 24) |
                          ((uint8_t)dataPtr[offset + 1] << 16) |
                          ((uint8_t)dataPtr[offset + 2] << 8) |
                          (uint8_t)dataPtr[offset + 3];
        offset += 4;
        if (offset + naluLen > totalLength) break;
        naluBytes += 4 + naluLen;  // 4-byte start code + NALU
        offset += naluLen;
    }

    size_t outputSize = extraSize + naluBytes;
    unsigned char *output = (unsigned char*)malloc(outputSize);
    if (output == NULL) {
        return -1;
    }

    size_t outOffset = 0;

    // Write SPS and PPS with start codes (for keyframes)
    if (*isKeyframe && sps != NULL && pps != NULL) {
        output[outOffset++] = 0x00;
        output[outOffset++] = 0x00;
        output[outOffset++] = 0x00;
        output[outOffset++] = 0x01;
        memcpy(output + outOffset, sps, spsSize);
        outOffset += spsSize;

        output[outOffset++] = 0x00;
        output[outOffset++] = 0x00;
        output[outOffset++] = 0x00;
        output[outOffset++] = 0x01;
        memcpy(output + outOffset, pps, ppsSize);
        outOffset += ppsSize;
    }

    // Convert AVCC to Annex B
    offset = 0;
    while (offset + 4 <= totalLength) {
        uint32_t naluLen = ((uint8_t)dataPtr[offset] << 24) |
                          ((uint8_t)dataPtr[offset + 1] << 16) |
                          ((uint8_t)dataPtr[offset + 2] << 8) |
                          (uint8_t)dataPtr[offset + 3];
        offset += 4;
        if (offset + naluLen > totalLength) break;

        output[outOffset++] = 0x00;
        output[outOffset++] = 0x00;
        output[outOffset++] = 0x00;
        output[outOffset++] = 0x01;
        memcpy(output + outOffset, dataPtr + offset, naluLen);
        outOffset += naluLen;
        offset += naluLen;
    }

    *outData = output;
    *outSize = outOffset;
    return 0;
}

// Compression output callback
static void compressionOutputCallback(void *outputCallbackRefCon,
                                      void *sourceFrameRefCon,
                                      OSStatus status,
                                      VTEncodeInfoFlags infoFlags,
                                      CMSampleBufferRef sampleBuffer) {
    if (status != noErr) {
        return;
    }

    VTEncoderContext *ctx = (VTEncoderContext*)outputCallbackRefCon;
    if (ctx == NULL || sampleBuffer == NULL) {
        return;
    }

    unsigned char *data = NULL;
    size_t size = 0;
    int isKeyframe = 0;
    int64_t timestampUs = 0;

    if (extractSPSPPSAndData(sampleBuffer, &data, &size, &isKeyframe, &timestampUs) == 0) {
        addFrameToStorage(&ctx->storage, data, size, timestampUs, isKeyframe);
        free(data);
    }
}

// Create encoder
static VTEncoderContext* vtCreateEncoder(int width, int height, float fps, int bitrate, int quality) {
    VTEncoderContext *ctx = (VTEncoderContext*)calloc(1, sizeof(VTEncoderContext));
    if (!ctx) return NULL;

    ctx->width = width;
    ctx->height = height;
    ctx->fps = fps;
    ctx->storage.capacity = 1000;
    ctx->storage.frames = (VTEncodedFrame*)calloc(ctx->storage.capacity, sizeof(VTEncodedFrame));
    if (!ctx->storage.frames) {
        free(ctx);
        return NULL;
    }

    CFMutableDictionaryRef encoderSpec = CFDictionaryCreateMutable(
        kCFAllocatorDefault, 0,
        &kCFTypeDictionaryKeyCallBacks,
        &kCFTypeDictionaryValueCallBacks);

    // Prefer hardware encoding
    CFDictionarySetValue(encoderSpec,
        kVTVideoEncoderSpecification_EnableHardwareAcceleratedVideoEncoder,
        kCFBooleanTrue);

    OSStatus status = VTCompressionSessionCreate(
        kCFAllocatorDefault,
        width,
        height,
        kCMVideoCodecType_H264,
        encoderSpec,
        NULL,
        kCFAllocatorDefault,
        compressionOutputCallback,
        ctx,
        &ctx->session);

    CFRelease(encoderSpec);

    if (status != noErr) {
        free(ctx->storage.frames);
        free(ctx);
        return NULL;
    }

    // Configure session
    VTSessionSetProperty(ctx->session, kVTCompressionPropertyKey_RealTime, kCFBooleanTrue);
    VTSessionSetProperty(ctx->session, kVTCompressionPropertyKey_AllowFrameReordering, kCFBooleanFalse);
    VTSessionSetProperty(ctx->session, kVTCompressionPropertyKey_ProfileLevel,
        kVTProfileLevel_H264_Baseline_AutoLevel);

    // Frame rate
    CFNumberRef fpsRef = CFNumberCreate(kCFAllocatorDefault, kCFNumberFloat32Type, &fps);
    VTSessionSetProperty(ctx->session, kVTCompressionPropertyKey_ExpectedFrameRate, fpsRef);
    CFRelease(fpsRef);

    // Bitrate
    if (bitrate > 0) {
        int avgBitrate = bitrate * 1000;
        CFNumberRef bitrateRef = CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &avgBitrate);
        VTSessionSetProperty(ctx->session, kVTCompressionPropertyKey_AverageBitRate, bitrateRef);
        CFRelease(bitrateRef);
    }

    // Quality (convert CRF 0-63 to VT quality 0.0-1.0)
    if (quality > 0) {
        float vtQuality = 1.0f - ((float)quality / 63.0f);
        if (vtQuality < 0.0f) vtQuality = 0.0f;
        if (vtQuality > 1.0f) vtQuality = 1.0f;
        CFNumberRef qualityRef = CFNumberCreate(kCFAllocatorDefault, kCFNumberFloat32Type, &vtQuality);
        VTSessionSetProperty(ctx->session, kVTCompressionPropertyKey_Quality, qualityRef);
        CFRelease(qualityRef);
    }

    // Keyframe interval (2 seconds)
    int keyframeInterval = (int)(fps * 2);
    CFNumberRef kfRef = CFNumberCreate(kCFAllocatorDefault, kCFNumberIntType, &keyframeInterval);
    VTSessionSetProperty(ctx->session, kVTCompressionPropertyKey_MaxKeyFrameInterval, kfRef);
    CFRelease(kfRef);

    VTCompressionSessionPrepareToEncodeFrames(ctx->session);

    return ctx;
}

// Encode a frame
static int vtEncodeFrame(VTEncoderContext *ctx, unsigned char *rgbaData, int64_t timestampUs, int forceKeyframe) {
    if (!ctx || !ctx->session) return -1;

    // Create pixel buffer
    CVPixelBufferRef pixelBuffer = NULL;

    CFMutableDictionaryRef attrs = CFDictionaryCreateMutable(
        kCFAllocatorDefault, 0,
        &kCFTypeDictionaryKeyCallBacks,
        &kCFTypeDictionaryValueCallBacks);

    CFDictionaryRef ioSurfaceProps = CFDictionaryCreate(kCFAllocatorDefault, NULL, NULL, 0,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    CFDictionarySetValue(attrs, kCVPixelBufferIOSurfacePropertiesKey, ioSurfaceProps);
    CFRelease(ioSurfaceProps);

    CVReturn result = CVPixelBufferCreate(
        kCFAllocatorDefault,
        ctx->width,
        ctx->height,
        kCVPixelFormatType_32BGRA,
        attrs,
        &pixelBuffer);

    CFRelease(attrs);

    if (result != kCVReturnSuccess) {
        return -1;
    }

    // Copy and convert RGBA to BGRA
    CVPixelBufferLockBaseAddress(pixelBuffer, 0);
    void *baseAddress = CVPixelBufferGetBaseAddress(pixelBuffer);
    size_t bytesPerRow = CVPixelBufferGetBytesPerRow(pixelBuffer);

    for (int y = 0; y < ctx->height; y++) {
        unsigned char *src = rgbaData + y * ctx->width * 4;
        unsigned char *dst = (unsigned char *)baseAddress + y * bytesPerRow;
        for (int x = 0; x < ctx->width; x++) {
            dst[x * 4 + 0] = src[x * 4 + 2]; // B
            dst[x * 4 + 1] = src[x * 4 + 1]; // G
            dst[x * 4 + 2] = src[x * 4 + 0]; // R
            dst[x * 4 + 3] = src[x * 4 + 3]; // A
        }
    }

    CVPixelBufferUnlockBaseAddress(pixelBuffer, 0);

    // Encode
    CMTime presentationTime = CMTimeMake(timestampUs, 1000000);

    CFMutableDictionaryRef frameProps = NULL;
    if (forceKeyframe) {
        frameProps = CFDictionaryCreateMutable(
            kCFAllocatorDefault, 1,
            &kCFTypeDictionaryKeyCallBacks,
            &kCFTypeDictionaryValueCallBacks);
        CFDictionarySetValue(frameProps, kVTEncodeFrameOptionKey_ForceKeyFrame, kCFBooleanTrue);
    }

    OSStatus status = VTCompressionSessionEncodeFrame(
        ctx->session,
        pixelBuffer,
        presentationTime,
        kCMTimeInvalid,
        frameProps,
        NULL,
        NULL);

    if (frameProps) CFRelease(frameProps);
    CVPixelBufferRelease(pixelBuffer);

    return (status == noErr) ? 0 : -1;
}

// Flush encoder
static int vtFlushEncoder(VTEncoderContext *ctx) {
    if (!ctx || !ctx->session) return -1;
    VTCompressionSessionCompleteFrames(ctx->session, kCMTimeInvalid);
    return 0;
}

// Get frame count
static int vtGetFrameCount(VTEncoderContext *ctx) {
    return ctx ? ctx->storage.count : 0;
}

// Get frame data
static VTEncodedFrame* vtGetFrame(VTEncoderContext *ctx, int index) {
    if (!ctx || index < 0 || index >= ctx->storage.count) return NULL;
    return &ctx->storage.frames[index];
}

// Destroy encoder
static void vtDestroyEncoder(VTEncoderContext *ctx) {
    if (!ctx) return;

    if (ctx->session) {
        VTCompressionSessionInvalidate(ctx->session);
        CFRelease(ctx->session);
    }

    if (ctx->storage.frames) {
        for (int i = 0; i < ctx->storage.count; i++) {
            free(ctx->storage.frames[i].data);
        }
        free(ctx->storage.frames);
    }

    free(ctx);
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

// videoToolboxEncoder implements H.264 encoding using VideoToolbox on macOS.
type videoToolboxEncoder struct {
	ctx *C.VTEncoderContext

	mu          sync.Mutex
	width       int
	height      int
	fps         float64
	firstFrame  bool
	initialized bool
}

func newPlatformEncoder() platformEncoder {
	return &videoToolboxEncoder{}
}

func (e *videoToolboxEncoder) init(width, height int, fps float64, opts ports.EncoderOptions) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.width = width
	e.height = height
	e.fps = fps
	e.firstFrame = true

	ctx := C.vtCreateEncoder(
		C.int(width),
		C.int(height),
		C.float(fps),
		C.int(opts.Bitrate),
		C.int(opts.Quality),
	)

	if ctx == nil {
		return fmt.Errorf("failed to create VideoToolbox encoder")
	}

	e.ctx = ctx
	e.initialized = true

	return nil
}

func (e *videoToolboxEncoder) encodeFrame(img image.Image, timestampMs int) ([]encodedFrame, error) {
	if !e.initialized || e.ctx == nil {
		return nil, ErrNotInitialized
	}

	// Convert image to RGBA
	bounds := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, e.width, e.height))
	draw.Draw(rgba, rgba.Bounds(), img, bounds.Min, draw.Src)

	timestampUs := int64(timestampMs) * 1000
	forceKeyframe := 0
	if e.firstFrame {
		forceKeyframe = 1
		e.firstFrame = false
	}

	result := C.vtEncodeFrame(
		e.ctx,
		(*C.uchar)(unsafe.Pointer(&rgba.Pix[0])),
		C.int64_t(timestampUs),
		C.int(forceKeyframe),
	)

	if result != 0 {
		return nil, ErrEncodingFailed
	}

	return nil, nil
}

func (e *videoToolboxEncoder) flush() ([]encodedFrame, error) {
	if !e.initialized || e.ctx == nil {
		return nil, nil
	}

	C.vtFlushEncoder(e.ctx)

	// Collect all frames
	frameCount := int(C.vtGetFrameCount(e.ctx))
	frames := make([]encodedFrame, 0, frameCount)

	for i := 0; i < frameCount; i++ {
		cFrame := C.vtGetFrame(e.ctx, C.int(i))
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

func (e *videoToolboxEncoder) close() {
	if e.ctx != nil {
		C.vtDestroyEncoder(e.ctx)
		e.ctx = nil
	}
	e.initialized = false
}
