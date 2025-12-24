package av1decoder

import (
	"fmt"
	"image"
	"io"
	"os"

	"github.com/Eyevinn/mp4ff/mp4"
)

// VideoFrame represents a decoded video frame with timing.
type VideoFrame struct {
	Image       image.Image
	TimestampMs int
	Duration    int // Duration in milliseconds
}

// MP4Reader reads and decodes AV1 frames from an MP4 file.
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
func (r *MP4Reader) ReadFrames(path string) ([]VideoFrame, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	return r.ReadFramesFromReader(f)
}

// ReadFramesFromReader reads all frames from an io.ReadSeeker.
func (r *MP4Reader) ReadFramesFromReader(reader io.ReadSeeker) ([]VideoFrame, error) {
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

	var frames []VideoFrame

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

func (r *MP4Reader) readFragmentedMP4(mp4File *mp4.File, reader io.ReadSeeker) ([]VideoFrame, error) {
	var frames []VideoFrame

	// Find video track and get trex
	var videoTrackID uint32
	var trex *mp4.TrexBox
	if mp4File.Init != nil && mp4File.Init.Moov != nil {
		for _, trak := range mp4File.Init.Moov.Traks {
			if trak.Mdia != nil && trak.Mdia.Hdlr != nil && trak.Mdia.Hdlr.HandlerType == "vide" {
				videoTrackID = trak.Tkhd.TrackID
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
				for _, sample := range samples {
					// Decode frame
					img, err := r.decoder.DecodeFrame(sample.Data)
					if err != nil {
						// Skip frames that can't be decoded (might need reference frames)
						currentTime += uint64(sample.Dur)
						continue
					}

					timestampMs := int(currentTime * 1000 / uint64(timescale))
					durationMs := int(uint64(sample.Dur) * 1000 / uint64(timescale))

					frames = append(frames, VideoFrame{
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

func (r *MP4Reader) readProgressiveMP4(mp4File *mp4.File, reader io.ReadSeeker) ([]VideoFrame, error) {
	// Progressive MP4 is not commonly used for our generated videos
	// Return empty for now - our generated videos are fragmented
	return nil, fmt.Errorf("progressive MP4 not supported, use fragmented MP4")
}

// Close releases resources.
func (r *MP4Reader) Close() {
	if r.decoder != nil {
		r.decoder.Close()
	}
}
