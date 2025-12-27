package smartencoder

import (
	"testing"
)

func TestNewAV1Encoder(t *testing.T) {
	encoder, info, err := New(CodecAV1, Options{})
	if err != nil {
		t.Fatalf("failed to create AV1 encoder: %v", err)
	}
	if encoder == nil {
		t.Fatal("encoder is nil")
	}
	if info.Codec != CodecAV1 {
		t.Errorf("expected codec AV1, got %s", info.Codec)
	}
	if info.Backend != BackendLibaom {
		t.Errorf("expected backend libaom, got %s", info.Backend)
	}
	if info.FallbackUsed {
		t.Error("fallback should not be used for AV1")
	}
}

func TestNewH264Encoder(t *testing.T) {
	encoder, info, err := New(CodecH264, Options{
		AllowFallback: true,
	})
	if err != nil {
		t.Fatalf("failed to create H.264 encoder: %v", err)
	}
	if encoder == nil {
		t.Fatal("encoder is nil")
	}

	// Either H.264 or AV1 (fallback) should be selected
	if info.Codec != CodecH264 && info.Codec != CodecAV1 {
		t.Errorf("expected codec H.264 or AV1, got %s", info.Codec)
	}

	// Verify requested codec is H.264
	if info.RequestedCodec != CodecH264 {
		t.Errorf("expected requested codec H.264, got %s", info.RequestedCodec)
	}

	t.Logf("Selected encoder: codec=%s, backend=%s, fallback=%v",
		info.Codec, info.Backend, info.FallbackUsed)
}

func TestIsAV1Available(t *testing.T) {
	// AV1 should always be available (libaom is linked)
	if !IsAV1Available() {
		t.Error("AV1 should always be available")
	}
}

func TestAvailabilityChecks(t *testing.T) {
	// Just log availability status
	t.Logf("H.264 available: %v", IsH264Available())
	t.Logf("H.264 native available: %v", IsH264NativeAvailable())
	t.Logf("H.264 FFmpeg available: %v", IsH264FFmpegAvailable())
	t.Logf("AV1 available: %v", IsAV1Available())
}
