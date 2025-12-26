package h264decoder

import (
	"fmt"
	"io"
	"os"

	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/user/loadshow/pkg/ports"
)

// MP4Reader reads and decodes H.264 frames from an MP4 file.
type MP4Reader struct {
	decoder *Decoder
}

// NewMP4Reader creates a new MP4 reader.
func NewMP4Reader() *MP4Reader {
	return &MP4Reader{
		decoder: New(),
	}
}

// ReadFrames reads all frames from an MP4 file.
func (r *MP4Reader) ReadFrames(path string) ([]ports.VideoFrame, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	return r.ReadFramesFromReader(f)
}

// ReadFramesFromReader reads all frames from an io.ReadSeeker.
func (r *MP4Reader) ReadFramesFromReader(reader io.ReadSeeker) ([]ports.VideoFrame, error) {
	// Parse MP4
	mp4File, err := mp4.DecodeFile(reader)
	if err != nil {
		return nil, fmt.Errorf("decode mp4: %w", err)
	}

	// Initialize decoder
	if err := r.decoder.Init(); err != nil {
		return nil, fmt.Errorf("init decoder: %w", err)
	}
	defer r.decoder.Close()

	var frames []ports.VideoFrame

	// Handle fragmented MP4
	if mp4File.IsFragmented() {
		frames, err = r.readFragmentedMP4(mp4File, reader)
		if err != nil {
			return nil, err
		}
	} else {
		frames, err = r.readProgressiveMP4(mp4File, reader)
		if err != nil {
			return nil, err
		}
	}

	return frames, nil
}

func (r *MP4Reader) readFragmentedMP4(mp4File *mp4.File, reader io.ReadSeeker) ([]ports.VideoFrame, error) {
	var frames []ports.VideoFrame

	// Find video track and get trex
	var videoTrackID uint32
	var trex *mp4.TrexBox
	var avcC *mp4.AvcCBox

	if mp4File.Init != nil && mp4File.Init.Moov != nil {
		for _, trak := range mp4File.Init.Moov.Traks {
			if trak.Mdia != nil && trak.Mdia.Hdlr != nil && trak.Mdia.Hdlr.HandlerType == "vide" {
				videoTrackID = trak.Tkhd.TrackID

				// Get avcC box for SPS/PPS
				if trak.Mdia.Minf != nil && trak.Mdia.Minf.Stbl != nil && trak.Mdia.Minf.Stbl.Stsd != nil {
					for _, child := range trak.Mdia.Minf.Stbl.Stsd.Children {
						if avc1, ok := child.(*mp4.VisualSampleEntryBox); ok {
							avcC = avc1.AvcC
						}
					}
				}
				break
			}
		}
		// Find trex for the track
		if mp4File.Init.Moov.Mvex != nil {
			for _, t := range mp4File.Init.Moov.Mvex.Trexs {
				if t.TrackID == videoTrackID {
					trex = t
					break
				}
			}
		}
	}

	if videoTrackID == 0 {
		return nil, fmt.Errorf("no video track found")
	}

	// Get timescale
	var timescale uint32 = 1000
	if mp4File.Init != nil && mp4File.Init.Moov != nil {
		for _, trak := range mp4File.Init.Moov.Traks {
			if trak.Tkhd.TrackID == videoTrackID && trak.Mdia != nil && trak.Mdia.Mdhd != nil {
				timescale = trak.Mdia.Mdhd.Timescale
				break
			}
		}
	}

	// Prepare SPS/PPS in Annex B format
	var spsPPS []byte
	if avcC != nil {
		for _, sps := range avcC.SPSnalus {
			spsPPS = append(spsPPS, 0, 0, 0, 1)
			spsPPS = append(spsPPS, sps...)
		}
		for _, pps := range avcC.PPSnalus {
			spsPPS = append(spsPPS, 0, 0, 0, 1)
			spsPPS = append(spsPPS, pps...)
		}
	}

	// Process fragments
	for _, seg := range mp4File.Segments {
		for _, frag := range seg.Fragments {
			if frag.Moof == nil {
				continue
			}

			for _, traf := range frag.Moof.Trafs {
				if traf.Tfhd.TrackID != videoTrackID {
					continue
				}

				// Get base decode time
				var baseDecodeTime uint64
				if traf.Tfdt != nil {
					baseDecodeTime = traf.Tfdt.BaseMediaDecodeTime()
				}

				// Get samples using trex
				samples, err := frag.GetFullSamples(trex)
				if err != nil {
					return nil, fmt.Errorf("get samples: %w", err)
				}

				currentTime := baseDecodeTime
				for i, sample := range samples {
					// Convert AVCC to Annex B format
					annexB := avccToAnnexB(sample.Data)

					// Prepend SPS/PPS for keyframes
					var frameData []byte
					if sample.Flags == mp4.SyncSampleFlags || i == 0 {
						frameData = make([]byte, len(spsPPS)+len(annexB))
						copy(frameData, spsPPS)
						copy(frameData[len(spsPPS):], annexB)
					} else {
						frameData = annexB
					}

					// Decode frame
					img, err := r.decoder.DecodeFrame(frameData)
					if err != nil {
						// Skip frames that can't be decoded
						currentTime += uint64(sample.Dur)
						continue
					}

					// Skip frames with no output (decoder needs more input)
					if img == nil {
						currentTime += uint64(sample.Dur)
						continue
					}

					timestampMs := int(currentTime * 1000 / uint64(timescale))
					durationMs := int(uint64(sample.Dur) * 1000 / uint64(timescale))

					frames = append(frames, ports.VideoFrame{
						Image:       img,
						TimestampMs: timestampMs,
						Duration:    durationMs,
					})

					currentTime += uint64(sample.Dur)
				}
			}
		}
	}

	return frames, nil
}

func (r *MP4Reader) readProgressiveMP4(mp4File *mp4.File, reader io.ReadSeeker) ([]ports.VideoFrame, error) {
	var frames []ports.VideoFrame

	// Find video track
	if mp4File.Moov == nil {
		return nil, fmt.Errorf("no moov box found")
	}

	var videoTrack *mp4.TrakBox
	var avcC *mp4.AvcCBox

	for _, trak := range mp4File.Moov.Traks {
		if trak.Mdia != nil && trak.Mdia.Hdlr != nil && trak.Mdia.Hdlr.HandlerType == "vide" {
			videoTrack = trak

			// Get avcC box for SPS/PPS
			if trak.Mdia.Minf != nil && trak.Mdia.Minf.Stbl != nil && trak.Mdia.Minf.Stbl.Stsd != nil {
				for _, child := range trak.Mdia.Minf.Stbl.Stsd.Children {
					if avc1, ok := child.(*mp4.VisualSampleEntryBox); ok {
						avcC = avc1.AvcC
					}
				}
			}
			break
		}
	}

	if videoTrack == nil {
		return nil, fmt.Errorf("no video track found")
	}

	// Get timescale
	var timescale uint32 = 1000
	if videoTrack.Mdia != nil && videoTrack.Mdia.Mdhd != nil {
		timescale = videoTrack.Mdia.Mdhd.Timescale
	}

	// Prepare SPS/PPS in Annex B format
	var spsPPS []byte
	if avcC != nil {
		for _, sps := range avcC.SPSnalus {
			spsPPS = append(spsPPS, 0, 0, 0, 1)
			spsPPS = append(spsPPS, sps...)
		}
		for _, pps := range avcC.PPSnalus {
			spsPPS = append(spsPPS, 0, 0, 0, 1)
			spsPPS = append(spsPPS, pps...)
		}
	}

	// Get stbl (sample table)
	if videoTrack.Mdia == nil || videoTrack.Mdia.Minf == nil || videoTrack.Mdia.Minf.Stbl == nil {
		return nil, fmt.Errorf("no sample table found")
	}
	stbl := videoTrack.Mdia.Minf.Stbl

	// Get sample count
	if stbl.Stsz == nil {
		return nil, fmt.Errorf("no stsz box found")
	}
	sampleCount := stbl.Stsz.SampleNumber

	// Build sync sample set (keyframes)
	syncSamples := make(map[uint32]bool)
	if stbl.Stss != nil {
		for _, sampleNr := range stbl.Stss.SampleNumber {
			syncSamples[sampleNr] = true
		}
	}

	// Read samples
	for sampleNr := uint32(1); sampleNr <= sampleCount; sampleNr++ {
		// Get sample data by reading from chunk offset
		sample, err := getSampleData(stbl, reader, sampleNr)
		if err != nil {
			continue // Skip samples that can't be read
		}

		// Get decode time from stts box
		var decodeTime uint64
		var dur uint32
		if stbl.Stts != nil {
			decodeTime, dur = stbl.Stts.GetDecodeTime(sampleNr)
		}
		timestampMs := int(decodeTime * 1000 / uint64(timescale))
		durationMs := int(uint64(dur) * 1000 / uint64(timescale))

		// Check if keyframe
		isKeyframe := syncSamples[sampleNr] || len(syncSamples) == 0

		// Convert AVCC to Annex B format
		annexB := avccToAnnexB(sample)

		// Prepend SPS/PPS for keyframes
		var frameData []byte
		if isKeyframe {
			frameData = make([]byte, len(spsPPS)+len(annexB))
			copy(frameData, spsPPS)
			copy(frameData[len(spsPPS):], annexB)
		} else {
			frameData = annexB
		}

		// Decode frame
		img, err := r.decoder.DecodeFrame(frameData)
		if err != nil {
			continue // Skip frames that can't be decoded
		}

		// Skip frames with no output (decoder needs more input)
		if img == nil {
			continue
		}

		frames = append(frames, ports.VideoFrame{
			Image:       img,
			TimestampMs: timestampMs,
			Duration:    durationMs,
		})
	}

	return frames, nil
}

// getSampleData reads sample data from a progressive MP4 file
func getSampleData(stbl *mp4.StblBox, reader io.ReadSeeker, sampleNr uint32) ([]byte, error) {
	if stbl.Stsc == nil || stbl.Stsz == nil {
		return nil, fmt.Errorf("missing stsc or stsz box")
	}

	// Get chunk number and first sample in chunk
	chunkNr, firstSampleInChunk, err := stbl.Stsc.ChunkNrFromSampleNr(int(sampleNr))
	if err != nil {
		return nil, fmt.Errorf("get chunk nr: %w", err)
	}

	// Get chunk offset
	var chunkOffset uint64
	if stbl.Stco != nil {
		chunkOffset, err = stbl.Stco.GetOffset(chunkNr)
		if err != nil {
			return nil, fmt.Errorf("get chunk offset: %w", err)
		}
	} else if stbl.Co64 != nil {
		if chunkNr < 1 || chunkNr > len(stbl.Co64.ChunkOffset) {
			return nil, fmt.Errorf("chunk nr out of range")
		}
		chunkOffset = stbl.Co64.ChunkOffset[chunkNr-1]
	} else {
		return nil, fmt.Errorf("no stco or co64 box")
	}

	// Calculate offset within chunk
	offset := chunkOffset
	for s := uint32(firstSampleInChunk); s < sampleNr; s++ {
		offset += uint64(stbl.Stsz.GetSampleSize(int(s)))
	}

	// Get sample size
	sampleSize := stbl.Stsz.GetSampleSize(int(sampleNr))

	// Seek and read
	if _, err := reader.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to sample: %w", err)
	}

	data := make([]byte, sampleSize)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, fmt.Errorf("read sample: %w", err)
	}

	return data, nil
}

// avccToAnnexB converts AVCC format (length-prefixed NALUs) to Annex B format (start code prefixed)
func avccToAnnexB(data []byte) []byte {
	var result []byte
	offset := 0

	for offset+4 <= len(data) {
		naluLen := int(data[offset])<<24 | int(data[offset+1])<<16 |
			int(data[offset+2])<<8 | int(data[offset+3])
		offset += 4

		if offset+naluLen > len(data) {
			break
		}

		// Add Annex B start code
		result = append(result, 0, 0, 0, 1)
		result = append(result, data[offset:offset+naluLen]...)
		offset += naluLen
	}

	return result
}

// Close releases resources.
func (r *MP4Reader) Close() {
	if r.decoder != nil {
		r.decoder.Close()
	}
}

// RawFrame represents a raw H.264 frame without decoding.
type RawFrame struct {
	Data        []byte
	TimestampMs int
	Duration    int
	IsKeyframe  bool
}

// ExtractFrames extracts raw H.264 frames from MP4 data without decoding.
func ExtractFrames(mp4Data []byte) ([]RawFrame, error) {
	reader := &bytesReadSeeker{data: mp4Data}

	mp4File, err := mp4.DecodeFile(reader)
	if err != nil {
		return nil, fmt.Errorf("decode mp4: %w", err)
	}

	// Handle fragmented vs progressive MP4
	if mp4File.IsFragmented() {
		return extractFramesFragmented(mp4File, reader)
	}
	return extractFramesProgressive(mp4File, reader)
}

func extractFramesProgressive(mp4File *mp4.File, reader io.ReadSeeker) ([]RawFrame, error) {
	var frames []RawFrame

	// Find video track
	if mp4File.Moov == nil {
		return nil, fmt.Errorf("no moov box found")
	}

	var videoTrack *mp4.TrakBox
	var avcC *mp4.AvcCBox
	var timescale uint32 = 1000

	for _, trak := range mp4File.Moov.Traks {
		if trak.Mdia != nil && trak.Mdia.Hdlr != nil && trak.Mdia.Hdlr.HandlerType == "vide" {
			videoTrack = trak
			if trak.Mdia.Mdhd != nil {
				timescale = trak.Mdia.Mdhd.Timescale
			}

			// Get avcC box for SPS/PPS
			if trak.Mdia.Minf != nil && trak.Mdia.Minf.Stbl != nil && trak.Mdia.Minf.Stbl.Stsd != nil {
				for _, child := range trak.Mdia.Minf.Stbl.Stsd.Children {
					if avc1, ok := child.(*mp4.VisualSampleEntryBox); ok {
						avcC = avc1.AvcC
					}
				}
			}
			break
		}
	}

	if videoTrack == nil {
		return nil, fmt.Errorf("no video track found")
	}

	// Prepare SPS/PPS in Annex B format
	var spsPPS []byte
	if avcC != nil {
		for _, sps := range avcC.SPSnalus {
			spsPPS = append(spsPPS, 0, 0, 0, 1)
			spsPPS = append(spsPPS, sps...)
		}
		for _, pps := range avcC.PPSnalus {
			spsPPS = append(spsPPS, 0, 0, 0, 1)
			spsPPS = append(spsPPS, pps...)
		}
	}

	// Get stbl (sample table)
	if videoTrack.Mdia == nil || videoTrack.Mdia.Minf == nil || videoTrack.Mdia.Minf.Stbl == nil {
		return nil, fmt.Errorf("no sample table found")
	}
	stbl := videoTrack.Mdia.Minf.Stbl

	// Get sample count
	if stbl.Stsz == nil {
		return nil, fmt.Errorf("no stsz box found")
	}
	sampleCount := stbl.Stsz.SampleNumber

	// Build sync sample set (keyframes)
	syncSamples := make(map[uint32]bool)
	if stbl.Stss != nil {
		for _, sampleNr := range stbl.Stss.SampleNumber {
			syncSamples[sampleNr] = true
		}
	}

	// Read samples
	for sampleNr := uint32(1); sampleNr <= sampleCount; sampleNr++ {
		sample, err := getSampleData(stbl, reader, sampleNr)
		if err != nil {
			continue
		}

		// Get decode time from stts box
		var decodeTime uint64
		var dur uint32
		if stbl.Stts != nil {
			decodeTime, dur = stbl.Stts.GetDecodeTime(sampleNr)
		}
		timestampMs := int(decodeTime * 1000 / uint64(timescale))
		durationMs := int(uint64(dur) * 1000 / uint64(timescale))
		isKeyframe := syncSamples[sampleNr] || len(syncSamples) == 0

		annexB := avccToAnnexB(sample)

		var frameData []byte
		if isKeyframe {
			frameData = make([]byte, len(spsPPS)+len(annexB))
			copy(frameData, spsPPS)
			copy(frameData[len(spsPPS):], annexB)
		} else {
			frameData = annexB
		}

		frames = append(frames, RawFrame{
			Data:        frameData,
			TimestampMs: timestampMs,
			Duration:    durationMs,
			IsKeyframe:  isKeyframe,
		})
	}

	return frames, nil
}

func extractFramesFragmented(mp4File *mp4.File, reader io.ReadSeeker) ([]RawFrame, error) {
	var frames []RawFrame

	// Find video track
	var videoTrackID uint32
	var trex *mp4.TrexBox
	var timescale uint32 = 1000
	var avcC *mp4.AvcCBox

	if mp4File.Init != nil && mp4File.Init.Moov != nil {
		for _, trak := range mp4File.Init.Moov.Traks {
			if trak.Mdia != nil && trak.Mdia.Hdlr != nil && trak.Mdia.Hdlr.HandlerType == "vide" {
				videoTrackID = trak.Tkhd.TrackID
				if trak.Mdia.Mdhd != nil {
					timescale = trak.Mdia.Mdhd.Timescale
				}

				// Get avcC box for SPS/PPS
				if trak.Mdia.Minf != nil && trak.Mdia.Minf.Stbl != nil && trak.Mdia.Minf.Stbl.Stsd != nil {
					for _, child := range trak.Mdia.Minf.Stbl.Stsd.Children {
						if avc1, ok := child.(*mp4.VisualSampleEntryBox); ok {
							avcC = avc1.AvcC
						}
					}
				}
				break
			}
		}
		if mp4File.Init.Moov.Mvex != nil {
			for _, t := range mp4File.Init.Moov.Mvex.Trexs {
				if t.TrackID == videoTrackID {
					trex = t
					break
				}
			}
		}
	}

	if videoTrackID == 0 {
		return nil, fmt.Errorf("no video track found")
	}

	// Prepare SPS/PPS in Annex B format
	var spsPPS []byte
	if avcC != nil {
		for _, sps := range avcC.SPSnalus {
			spsPPS = append(spsPPS, 0, 0, 0, 1)
			spsPPS = append(spsPPS, sps...)
		}
		for _, pps := range avcC.PPSnalus {
			spsPPS = append(spsPPS, 0, 0, 0, 1)
			spsPPS = append(spsPPS, pps...)
		}
	}

	// Process fragments
	for _, seg := range mp4File.Segments {
		for _, frag := range seg.Fragments {
			if frag.Moof == nil {
				continue
			}

			for _, traf := range frag.Moof.Trafs {
				if traf.Tfhd.TrackID != videoTrackID {
					continue
				}

				var baseDecodeTime uint64
				if traf.Tfdt != nil {
					baseDecodeTime = traf.Tfdt.BaseMediaDecodeTime()
				}

				samples, err := frag.GetFullSamples(trex)
				if err != nil {
					return nil, fmt.Errorf("get samples: %w", err)
				}

				currentTime := baseDecodeTime
				for _, sample := range samples {
					timestampMs := int(currentTime * 1000 / uint64(timescale))
					durationMs := int(uint64(sample.Dur) * 1000 / uint64(timescale))
					isKeyframe := sample.Flags == mp4.SyncSampleFlags

					// Convert AVCC to Annex B
					annexB := avccToAnnexB(sample.Data)

					// Prepend SPS/PPS for keyframes
					var frameData []byte
					if isKeyframe {
						frameData = make([]byte, len(spsPPS)+len(annexB))
						copy(frameData, spsPPS)
						copy(frameData[len(spsPPS):], annexB)
					} else {
						frameData = annexB
					}

					frames = append(frames, RawFrame{
						Data:        frameData,
						TimestampMs: timestampMs,
						Duration:    durationMs,
						IsKeyframe:  isKeyframe,
					})

					currentTime += uint64(sample.Dur)
				}
			}
		}
	}

	return frames, nil
}

// bytesReadSeeker implements io.ReadSeeker for a byte slice
type bytesReadSeeker struct {
	data   []byte
	offset int64
}

func (b *bytesReadSeeker) Read(p []byte) (n int, err error) {
	if b.offset >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.offset:])
	b.offset += int64(n)
	return n, nil
}

func (b *bytesReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = b.offset + offset
	case io.SeekEnd:
		newOffset = int64(len(b.data)) + offset
	}
	if newOffset < 0 {
		return 0, fmt.Errorf("negative offset")
	}
	b.offset = newOffset
	return newOffset, nil
}
