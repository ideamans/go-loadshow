// Package codecdetect provides utilities for detecting video codec from MP4 files.
package codecdetect

import (
	"fmt"
	"io"
	"os"

	"github.com/Eyevinn/mp4ff/mp4"
)

// Codec represents a video codec type.
type Codec string

const (
	CodecH264    Codec = "h264"
	CodecAV1     Codec = "av1"
	CodecUnknown Codec = "unknown"
)

// DetectFromFile detects the video codec used in an MP4 file.
func DetectFromFile(path string) (Codec, error) {
	f, err := os.Open(path)
	if err != nil {
		return CodecUnknown, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	return DetectFromReader(f)
}

// DetectFromReader detects the video codec from an io.ReadSeeker.
func DetectFromReader(reader io.ReadSeeker) (Codec, error) {
	mp4File, err := mp4.DecodeFile(reader)
	if err != nil {
		return CodecUnknown, fmt.Errorf("decode mp4: %w", err)
	}

	// Reset reader position for subsequent reads
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return CodecUnknown, fmt.Errorf("seek: %w", err)
	}

	return detectFromMP4File(mp4File)
}

// DetectFromBytes detects the video codec from MP4 data bytes.
func DetectFromBytes(data []byte) (Codec, error) {
	reader := &bytesReadSeeker{data: data}
	return DetectFromReader(reader)
}

func detectFromMP4File(mp4File *mp4.File) (Codec, error) {
	// Check fragmented MP4
	if mp4File.IsFragmented() {
		if mp4File.Init != nil && mp4File.Init.Moov != nil {
			for _, trak := range mp4File.Init.Moov.Traks {
				codec := detectCodecFromTrack(trak)
				if codec != CodecUnknown {
					return codec, nil
				}
			}
		}
	}

	// Check progressive MP4
	if mp4File.Moov != nil {
		for _, trak := range mp4File.Moov.Traks {
			codec := detectCodecFromTrack(trak)
			if codec != CodecUnknown {
				return codec, nil
			}
		}
	}

	return CodecUnknown, fmt.Errorf("no video track found")
}

func detectCodecFromTrack(trak *mp4.TrakBox) Codec {
	if trak.Mdia == nil || trak.Mdia.Hdlr == nil {
		return CodecUnknown
	}

	// Only process video tracks
	if trak.Mdia.Hdlr.HandlerType != "vide" {
		return CodecUnknown
	}

	if trak.Mdia.Minf == nil || trak.Mdia.Minf.Stbl == nil || trak.Mdia.Minf.Stbl.Stsd == nil {
		return CodecUnknown
	}

	stsd := trak.Mdia.Minf.Stbl.Stsd

	for _, child := range stsd.Children {
		switch child.Type() {
		case "avc1", "avc3":
			// H.264/AVC
			return CodecH264
		case "av01":
			// AV1
			return CodecAV1
		case "hvc1", "hev1":
			// H.265/HEVC - not supported but detect it
			return CodecUnknown
		}
	}

	return CodecUnknown
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
