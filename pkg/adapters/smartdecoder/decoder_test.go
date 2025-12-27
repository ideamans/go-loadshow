package smartdecoder

import (
	"testing"
)

func TestNewForCodecAV1(t *testing.T) {
	decoder, info, err := NewForCodec(CodecAV1, Options{})
	if err != nil {
		t.Fatalf("failed to create AV1 decoder: %v", err)
	}
	if decoder == nil {
		t.Fatal("decoder is nil")
	}
	defer decoder.Close()

	if info.Codec != CodecAV1 {
		t.Errorf("expected codec AV1, got %s", info.Codec)
	}
	if info.Backend != BackendLibaom {
		t.Errorf("expected backend libaom, got %s", info.Backend)
	}
}

func TestNewForCodecH264(t *testing.T) {
	if !IsH264Available() {
		t.Skip("H.264 decoder not available")
	}

	decoder, info, err := NewForCodec(CodecH264, Options{})
	if err != nil {
		t.Fatalf("failed to create H.264 decoder: %v", err)
	}
	if decoder == nil {
		t.Fatal("decoder is nil")
	}
	defer decoder.Close()

	if info.Codec != CodecH264 {
		t.Errorf("expected codec H.264, got %s", info.Codec)
	}

	t.Logf("H.264 decoder backend: %s", info.Backend)
}

func TestNewForCodecUnknown(t *testing.T) {
	_, _, err := NewForCodec(CodecUnknown, Options{})
	if err == nil {
		t.Error("expected error for unknown codec")
	}
	if err != ErrUnsupportedCodec {
		t.Errorf("expected ErrUnsupportedCodec, got %v", err)
	}
}

func TestIsAV1Available(t *testing.T) {
	// AV1 should always be available (libaom is linked)
	if !IsAV1Available() {
		t.Error("AV1 should always be available")
	}
}

func TestAvailabilityChecks(t *testing.T) {
	t.Logf("H.264 available: %v", IsH264Available())
	t.Logf("AV1 available: %v", IsAV1Available())
}
