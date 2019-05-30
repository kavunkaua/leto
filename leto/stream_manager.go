package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type StreamManager struct {
	mx sync.Mutex

	period time.Duration

	baseMovieName, baseFrameMatching string
	encodeCmd, streamCmd             *exec.Cmd

	encodeIn            io.WriteCloser
	encodeOut           *io.PipeWriter
	streamIn            *io.PipeReader
	frameCorrespondance *os.File

	host string

	fps         float64
	bitrate     int
	destAddress string
	resolution  string
	quality     string

	logger *log.Logger
}

func NewStreamManager(basedir string, fps float64, bitrate int, destAddress string) *StreamManager {
	return &StreamManager{
		baseMovieName:     filepath.Join(basedir, "stream.mp4"),
		baseFrameMatching: filepath.Join(basedir, "stream.frame-matching.txt"),
		fps:               fps,
		bitrate:           bitrate,
		destAddress:       destAddress,
		resolution:        "",
		quality:           "ultrafast",
		period:            2 * time.Hour,
		logger:            log.New(os.Stderr, "[stream] ", log.LstdFlags),
	}
}

func (s *StreamManager) waitUnsafe() {
	s.logger.Printf("waiting for encoding to stop")
	if s.encodeCmd != nil {
		s.encodeCmd.Wait()
		s.encodeCmd = nil
	}
	if s.encodeOut != nil {
		s.encodeOut.Close()
		s.encodeOut = nil
	}

	if s.streamCmd != nil {
		s.logger.Printf("waiting for streaming to stop")
		s.streamCmd.Wait()
		s.streamCmd = nil
	}

	if s.streamIn != nil {
		s.streamIn.Close()
		s.streamIn = nil
	}

	if s.frameCorrespondance != nil {
		s.frameCorrespondance.Close()
		s.frameCorrespondance = nil
	}
}

func (s *StreamManager) Wait() {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.waitUnsafe()
}

func (s *StreamManager) startTasks() error {

	mName, _, err := FilenameWithoutOverwrite(s.baseMovieName)
	if err != nil {
		return err
	}

	cfName, _, err := FilenameWithoutOverwrite(s.baseFrameMatching)
	if err != nil {
		return err
	}
	s.frameCorrespondance, err = os.Create(cfName)
	if err != nil {
		return err
	}
	s.streamIn, s.encodeOut = io.Pipe()

	s.encodeCmd = s.buildEncodeCommand()
	s.encodeIn, err = s.encodeCmd.StdinPipe()
	if err != nil {
		return err
	}
	s.encodeCmd.Stderr = nil
	s.encodeCmd.Stdout = s.encodeOut

	s.streamCmd = s.buildStreamCommand(mName)
	s.streamCmd.Stdout = nil
	s.streamCmd.Stderr = nil
	s.streamCmd.Stdin = s.streamIn

	s.logger.Printf("Starting streaming to %s and %s", mName, s.destAddress)
	err = s.encodeCmd.Start()
	if err != nil {
		return err
	}
	err = s.streamCmd.Start()
	return err
}

func (s *StreamManager) stopTasks() {
	s.logger.Printf("Stopping streaming tasks")

	if s.encodeIn != nil {
		s.encodeIn.Close()
	}
	if s.encodeCmd != nil {
		s.encodeCmd.Process.Signal(os.Interrupt)
	}

}

func (s *StreamManager) EncodeAndStreamMuxedStream(muxed io.Reader) {
	header := make([]byte, 3*8)
	var err error
	s.host, err = os.Hostname()
	if err != nil {
		s.logger.Printf("cannot get hostname: %s", err)
		return
	}
	currentFrame := 0

	nextFile := time.Now().Add(s.period)

	for {
		_, err := io.ReadFull(muxed, header)
		if err != nil {
			s.logger.Printf("cannot read header: %s", err)
			if err == io.EOF {
				s.stopTasks()
				return
			}
		}
		actual := binary.LittleEndian.Uint64(header)
		width := binary.LittleEndian.Uint64(header[8:])
		height := binary.LittleEndian.Uint64(header[16:])
		if len(s.resolution) == 0 {
			s.resolution = fmt.Sprintf("%dx%d", width, height)
		}
		if s.encodeCmd == nil && s.streamCmd == nil && s.frameCorrespondance == nil {
			s.mx.Lock()
			if err := s.startTasks(); err != nil {
				s.mx.Unlock()
				s.logger.Printf("Could not start stream tasks: %s", err)
				return
			}
			s.mx.Unlock()
		}

		fmt.Fprintf(s.frameCorrespondance, "%d %d\n", currentFrame, actual)
		_, err = io.CopyN(s.encodeIn, muxed, int64(3*width*height))
		if err != nil {
			s.logger.Printf("cannot copy frame: %s", err)
		}
		currentFrame += 1

		now := time.Now()
		if now.After(nextFile) == true {
			log.Printf("Creating new film segment after %s", s.period)
			s.mx.Lock()
			s.stopTasks()
			s.waitUnsafe()
			s.mx.Unlock()
			nextFile = now.Add(s.period)
		}
	}

}

func (s *StreamManager) buildEncodeCommand() *exec.Cmd {
	cbr := fmt.Sprintf("%dk", s.bitrate)
	return exec.Command("ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-f", "rawvideo",
		"-vcodec", "rawvideo",
		"-pixel_format", "rgb24",
		"-video_size", s.resolution,
		"-framerate", fmt.Sprintf("%f", s.fps),
		"-i", "-",
		"-c:v:0", "libx264",
		"-g", fmt.Sprintf("%d", int(2*s.fps)),
		"-keyint_min", fmt.Sprintf("%d", int(s.fps)),
		"-b:v", cbr,
		"-minrate", cbr,
		"-maxrate", cbr,
		"-pix_fmt",
		"yuv420p",
		"-s", s.resolution,
		"-preset", s.quality,
		"-tune", "film",
		"-f", "flv",
		"-")
}

func (s *StreamManager) buildStreamCommand(file string) *exec.Cmd {
	res := exec.Command("ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-f", "flv",
		"-i", "-",
		"-vcodec", "copy",
		file)
	if len(s.destAddress) > 0 {
		res.Args = append(res.Args,
			"-vcodec", "copy",
			fmt.Sprintf("rtmp://%s/olympus/%s.flv", s.destAddress, s.host))
	}
	return res
}