package gohlslib

import (
	"io"
	"strconv"
	"time"

	"github.com/bluenviron/gohlslib/pkg/fmp4"
	"github.com/bluenviron/gohlslib/pkg/storage"
)

func fmp4PartName(id uint64) string {
	return "part" + strconv.FormatUint(id, 10)
}

type muxerPart struct {
	startDTS            time.Duration
	videoTrack          *Track
	audioTrack          *Track
	audioTrackTimeScale uint32
	id                  uint64
	storage             storage.Part

	isIndependent       bool
	videoSamples        []*fmp4.PartSample
	audioSamples        []*fmp4.PartSample
	finalDuration       time.Duration
	videoStartDTSFilled bool
	videoStartDTS       time.Duration
	audioStartDTSFilled bool
	audioStartDTS       time.Duration
}

func newMuxerPart(
	startDTS time.Duration,
	videoTrack *Track,
	audioTrack *Track,
	audioTrackTimeScale uint32,
	id uint64,
	storage storage.Part,
) *muxerPart {
	p := &muxerPart{
		startDTS:            startDTS,
		videoTrack:          videoTrack,
		audioTrack:          audioTrack,
		audioTrackTimeScale: audioTrackTimeScale,
		id:                  id,
		storage:             storage,
	}

	if videoTrack == nil {
		p.isIndependent = true
	}

	return p
}

func (p *muxerPart) name() string {
	return fmp4PartName(p.id)
}

func (p *muxerPart) reader() (io.ReadCloser, error) {
	return p.storage.Reader()
}

func (p *muxerPart) computeDuration(nextDTS time.Duration) time.Duration {
	return nextDTS - p.startDTS
}

func (p *muxerPart) finalize(nextDTS time.Duration) error {
	part := fmp4.Part{}

	if p.videoSamples != nil {
		part.Tracks = append(part.Tracks, &fmp4.PartTrack{
			ID:       1,
			BaseTime: durationGoToMp4(p.videoStartDTS, 90000),
			Samples:  p.videoSamples,
			IsVideo:  true,
		})
	}

	if p.audioSamples != nil {
		part.Tracks = append(part.Tracks, &fmp4.PartTrack{
			ID:       1 + len(part.Tracks),
			BaseTime: durationGoToMp4(p.audioStartDTS, p.audioTrackTimeScale),
			Samples:  p.audioSamples,
		})
	}

	err := part.Marshal(p.storage.Writer())
	if err != nil {
		return err
	}

	p.finalDuration = p.computeDuration(nextDTS)

	p.videoSamples = nil
	p.audioSamples = nil

	return nil
}

func (p *muxerPart) writeH264(sample *augmentedVideoSample) {
	if !p.videoStartDTSFilled {
		p.videoStartDTSFilled = true
		p.videoStartDTS = sample.dts
	}

	if !sample.IsNonSyncSample {
		p.isIndependent = true
	}

	p.videoSamples = append(p.videoSamples, &sample.PartSample)
}

func (p *muxerPart) writeAudio(sample *augmentedAudioSample) {
	if !p.audioStartDTSFilled {
		p.audioStartDTSFilled = true
		p.audioStartDTS = sample.dts
	}

	p.audioSamples = append(p.audioSamples, &sample.PartSample)
}
