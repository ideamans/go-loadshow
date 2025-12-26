package h264encoder

import (
	"bytes"
	"fmt"

	"github.com/Eyevinn/mp4ff/mp4"
)

// buildMP4 creates an MP4 container from encoded H.264 frames.
func (e *Encoder) buildMP4() ([]byte, error) {
	if len(e.frames) == 0 {
		return nil, ErrNoFrames
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

	// Extract SPS and PPS from first keyframe
	sps, pps, err := extractSPSPPS(e.frames)
	if err != nil {
		return nil, fmt.Errorf("extract SPS/PPS: %w", err)
	}

	// Create AVC decoder configuration record
	avcC, err := createAVCConfigRecord(sps, pps)
	if err != nil {
		return nil, fmt.Errorf("create avcC: %w", err)
	}

	// Create avc1 sample entry
	avc1 := mp4.CreateVisualSampleEntryBox("avc1", width, height, avcC)
	trak.Mdia.Minf.Stbl.Stsd.AddChild(avc1)

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

		// Convert Annex B to AVCC format if needed
		avccData := convertToAVCC(frame.data)

		frag.AddFullSample(mp4.FullSample{
			Sample: mp4.Sample{
				Flags: flags,
				Size:  uint32(len(avccData)),
				Dur:   dur,
			},
			DecodeTime: decodeTime,
			Data:       avccData,
		})
	}

	// Write to buffer
	var buf bytes.Buffer

	// Write ftyp
	ftyp := mp4.NewFtyp("isom", 0x200, []string{"isom", "iso2", "avc1", "mp41"})
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

// extractSPSPPS extracts SPS and PPS NAL units from the first keyframe.
func extractSPSPPS(frames []encodedFrame) (sps, pps []byte, err error) {
	for _, f := range frames {
		if !f.isKeyframe || len(f.data) == 0 {
			continue
		}

		// Parse NAL units from Annex B format
		nalus := parseAnnexB(f.data)
		for _, nalu := range nalus {
			if len(nalu) == 0 {
				continue
			}
			nalType := nalu[0] & 0x1F
			switch nalType {
			case 7: // SPS
				if sps == nil {
					sps = make([]byte, len(nalu))
					copy(sps, nalu)
				}
			case 8: // PPS
				if pps == nil {
					pps = make([]byte, len(nalu))
					copy(pps, nalu)
				}
			}
		}

		if sps != nil && pps != nil {
			return sps, pps, nil
		}
	}

	if sps == nil {
		return nil, nil, fmt.Errorf("SPS not found")
	}
	if pps == nil {
		return nil, nil, fmt.Errorf("PPS not found")
	}
	return sps, pps, nil
}

// createAVCConfigRecord creates an AVCDecoderConfigurationRecord box.
func createAVCConfigRecord(sps, pps []byte) (*mp4.AvcCBox, error) {
	// Use mp4ff's CreateAvcC which handles all the parsing internally
	avcC, err := mp4.CreateAvcC([][]byte{sps}, [][]byte{pps}, true)
	if err != nil {
		return nil, fmt.Errorf("create avcC: %w", err)
	}

	return avcC, nil
}

// parseAnnexB parses Annex B byte stream into individual NAL units.
func parseAnnexB(data []byte) [][]byte {
	var nalus [][]byte
	start := 0
	i := 0

	for i < len(data) {
		// Look for start code (0x00 0x00 0x01 or 0x00 0x00 0x00 0x01)
		if i+2 < len(data) && data[i] == 0 && data[i+1] == 0 {
			startCodeLen := 0
			if data[i+2] == 1 {
				startCodeLen = 3
			} else if i+3 < len(data) && data[i+2] == 0 && data[i+3] == 1 {
				startCodeLen = 4
			}

			if startCodeLen > 0 {
				// Save previous NAL unit if any
				if i > start {
					nalus = append(nalus, data[start:i])
				}
				i += startCodeLen
				start = i
				continue
			}
		}
		i++
	}

	// Add last NAL unit
	if start < len(data) {
		nalus = append(nalus, data[start:])
	}

	return nalus
}

// convertToAVCC converts Annex B format to AVCC format (length-prefixed).
func convertToAVCC(data []byte) []byte {
	nalus := parseAnnexB(data)
	if len(nalus) == 0 {
		return data
	}

	// Calculate total size
	totalSize := 0
	for _, nalu := range nalus {
		totalSize += 4 + len(nalu) // 4-byte length prefix
	}

	result := make([]byte, totalSize)
	offset := 0

	for _, nalu := range nalus {
		// Skip SPS and PPS in sample data (they're in avcC box)
		if len(nalu) > 0 {
			nalType := nalu[0] & 0x1F
			if nalType == 7 || nalType == 8 {
				continue
			}
		}

		// Write 4-byte length prefix (big endian)
		length := len(nalu)
		result[offset] = byte(length >> 24)
		result[offset+1] = byte(length >> 16)
		result[offset+2] = byte(length >> 8)
		result[offset+3] = byte(length)
		offset += 4

		// Write NAL unit data
		copy(result[offset:], nalu)
		offset += length
	}

	return result[:offset]
}
