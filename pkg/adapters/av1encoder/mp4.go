package av1encoder

import (
	"bytes"
	"fmt"

	"github.com/Eyevinn/mp4ff/av1"
	"github.com/Eyevinn/mp4ff/mp4"
)

// buildMP4 creates an MP4 container from encoded AV1 frames.
func (e *Encoder) buildMP4() ([]byte, error) {
	if len(e.frames) == 0 {
		return nil, fmt.Errorf("no frames to encode")
	}

	timescale := uint32(e.fps * 1000)
	trackID := uint32(1)

	// Create initialization segment
	init := mp4.CreateEmptyInit()
	init.AddEmptyTrack(timescale, "video", "en")

	trak := init.Moov.Trak

	// Set video dimensions
	width := uint16(e.width)
	height := uint16(e.height)

	// Create AV1 codec configuration
	av1C := createAV1ConfigRecord(e.frames)

	// Create av01 sample entry
	av01 := mp4.CreateVisualSampleEntryBox("av01", width, height, av1C)
	trak.Mdia.Minf.Stbl.Stsd.AddChild(av01)

	// Set track header dimensions
	trak.Tkhd.Width = mp4.Fixed32(e.width << 16)
	trak.Tkhd.Height = mp4.Fixed32(e.height << 16)

	// Create fragment
	frag, err := mp4.CreateFragment(1, trackID)
	if err != nil {
		return nil, fmt.Errorf("create fragment: %w", err)
	}

	// Add samples to fragment
	for i, frame := range e.frames {
		// Calculate duration in timescale units
		var dur uint32
		if i < len(e.frames)-1 {
			nextTs := e.frames[i+1].timestampUs
			dur = uint32((nextTs - frame.timestampUs) * int64(timescale) / 1000000)
		} else {
			dur = uint32(timescale) / uint32(e.fps)
		}
		if dur == 0 {
			dur = uint32(timescale) / uint32(e.fps)
		}

		// Decode time in timescale units
		decodeTime := uint64(frame.timestampUs) * uint64(timescale) / 1000000

		flags := mp4.NonSyncSampleFlags
		if frame.isKeyframe {
			flags = mp4.SyncSampleFlags
		}

		frag.AddFullSample(mp4.FullSample{
			Sample: mp4.Sample{
				Flags: flags,
				Size:  uint32(len(frame.data)),
				Dur:   dur,
			},
			DecodeTime: decodeTime,
			Data:       frame.data,
		})
	}

	// Write to buffer
	var buf bytes.Buffer

	// Write ftyp
	ftyp := mp4.NewFtyp("isom", 0x200, []string{"isom", "iso2", "av01", "mp41"})
	if err := ftyp.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode ftyp: %w", err)
	}

	// Write moov (from init segment)
	if err := init.Moov.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode moov: %w", err)
	}

	// Write fragment (moof + mdat)
	if err := frag.Encode(&buf); err != nil {
		return nil, fmt.Errorf("encode fragment: %w", err)
	}

	return buf.Bytes(), nil
}

// createAV1ConfigRecord creates an AV1CodecConfigurationRecord box
func createAV1ConfigRecord(frames []encodedFrame) *mp4.Av1CBox {
	// Find first keyframe to extract sequence header
	var seqHdr []byte
	for _, f := range frames {
		if f.isKeyframe && len(f.data) > 0 {
			// Extract sequence header OBU from the first keyframe
			seqHdr = extractSequenceHeader(f.data)
			break
		}
	}

	return &mp4.Av1CBox{
		CodecConfRec: av1.CodecConfRec{
			Version:              1,
			SeqProfile:           0,
			SeqLevelIdx0:         8, // Level 4.0
			SeqTier0:             0,
			HighBitdepth:         0,
			TwelveBit:            0,
			MonoChrome:           0,
			ChromaSubsamplingX:   1, // 4:2:0
			ChromaSubsamplingY:   1,
			ChromaSamplePosition: 0,
			ConfigOBUs:           seqHdr,
		},
	}
}

// extractSequenceHeader extracts the sequence header OBU from AV1 bitstream
func extractSequenceHeader(data []byte) []byte {
	// AV1 OBU header parsing
	// First byte: forbidden (1) + type (4) + extension flag (1) + has_size (1) + reserved (1)
	if len(data) < 2 {
		return nil
	}

	offset := 0
	for offset < len(data) {
		if offset >= len(data) {
			break
		}

		header := data[offset]
		obuType := (header >> 3) & 0x0F
		hasExtension := (header >> 2) & 0x01
		hasSizeField := (header >> 1) & 0x01

		offset++

		// Skip extension header if present
		if hasExtension == 1 && offset < len(data) {
			offset++
		}

		// Read size if present
		var obuSize int
		if hasSizeField == 1 {
			obuSize, offset = readLeb128(data, offset)
		} else {
			// No size field - rest of data is this OBU
			obuSize = len(data) - offset
		}

		// Check if this is sequence header (type 1)
		if obuType == 1 {
			// Return the entire OBU including header
			startOffset := offset - 1
			if hasExtension == 1 {
				startOffset--
			}
			endOffset := offset + obuSize
			if endOffset > len(data) {
				endOffset = len(data)
			}
			return data[startOffset:endOffset]
		}

		offset += obuSize
	}

	return nil
}

// readLeb128 reads a LEB128 encoded value
func readLeb128(data []byte, offset int) (int, int) {
	value := 0
	for i := 0; i < 8 && offset < len(data); i++ {
		b := data[offset]
		offset++
		value |= int(b&0x7F) << (i * 7)
		if b&0x80 == 0 {
			break
		}
	}
	return value, offset
}
