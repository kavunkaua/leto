package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/formicidae-tracker/hermes"
	"github.com/formicidae-tracker/leto"
)

type ArtemisManager struct {
	incoming, merged, file, broadcast chan *hermes.FrameReadout
	mx                                sync.Mutex
	wg, wgEncode                      sync.WaitGroup
	quitEncode                        chan struct{}
	fileWriter                        *FrameReadoutFileWriter
	trackers                          *RemoteManager
	isMaster                          bool

	artemisCmd    *exec.Cmd
	frameBuffer   *bytes.Buffer
	experimentDir string
	logger        *log.Logger

	experimentName string
	since          time.Time
}

func NewArtemisManager() (*ArtemisManager, error) {
	cmd := exec.Command("artemis", "--version")
	_, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Could not find artemis: %s", err)
	}
	//TODO Check version compatibility"
	//TODO check if slave or master
	cmd = exec.Command("ffmpeg", "-version")
	_, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Could not find ffmpeg: %s", err)
	}

	return &ArtemisManager{
		isMaster: true,
		logger:   log.New(os.Stderr, "[artemis]", log.LstdFlags),
	}, nil
}

func (m *ArtemisManager) Status() (bool, string, time.Time) {
	m.mx.Lock()
	defer m.mx.Unlock()
	if m.incoming == nil {
		return false, "", time.Time{}
	}
	return true, m.experimentName, m.since
}

func (m *ArtemisManager) ExperimentDir(expname string) (string, error) {
	basename := filepath.Join(xdg.DataHome, "fort-experiments", expname)
	basedir, _, err := FilenameWithoutOverwrite(basename)
	return basedir, err
}

func (*ArtemisManager) LinkHostname(address string) error {
	return fmt.Errorf("Work balancing with multiple host is not yet implemented")
}

func (*ArtemisManager) UnlinkHostname(address string) error {
	return fmt.Errorf("Work balancing with multiple host is not yet implemented")
}

func (m *ArtemisManager) Start(config *leto.TrackingStart) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	if m.incoming != nil {
		m.mx.Unlock()
		return fmt.Errorf("ArtemisManager: Start: already started")
	}

	m.incoming = make(chan *hermes.FrameReadout, 10)
	m.merged = make(chan *hermes.FrameReadout, 10)
	m.file = make(chan *hermes.FrameReadout, 200)
	m.broadcast = make(chan *hermes.FrameReadout, 10)

	var err error
	m.experimentDir, err = m.ExperimentDir(config.ExperimentName)
	if err != nil {
		return err
	}
	err = os.MkdirAll(m.experimentDir, 0644)
	if err != nil {
		return err
	}

	m.fileWriter, err = NewFrameReadoutWriter(filepath.Join(m.experimentDir, "tracking.hermes"))
	if err != nil {
		return err
	}

	m.trackers = NewRemoteManager()
	//TODO actually write the workloadbalance and different definitions
	wb := &WorkloadBalance{
		FPS:       config.Camera.FPS,
		Stride:    1,
		IDsByUUID: map[string][]bool{"foo": []bool{true}},
	}
	m.wg.Add(1)
	go func() {
		for i := range m.merged {
			select {
			case m.file <- i:
			default:
			}
			select {
			case m.broadcast <- i:
			default:
			}
		}
		close(m.file)
		close(m.broadcast)
		m.wg.Done()
	}()
	m.wg.Add(1)
	go func() {
		MergeFrameReadout(wb, m.incoming, m.merged)
		m.wg.Done()
	}()
	m.wg.Add(1)
	go func() {
		err := m.trackers.Listen(fmt.Sprintf(":%d", leto.ARTEMIS_IN_PORT), ArtemisOnAccept(m.incoming), func() {
			close(m.incoming)
			m.mx.Lock()
			defer m.mx.Unlock()
			m.incoming = nil
		})
		m.logger.Printf("%s", err)
		m.wg.Done()
	}()
	m.wg.Add(1)
	go func() {
		BroadcastFrameReadout(fmt.Sprintf(":%d", leto.ARTEMIS_OUT_PORT), m.broadcast)
		m.wg.Done()
	}()
	m.wg.Add(1)
	go func() {
		m.fileWriter.WriteAll(m.file)
		m.wg.Done()
	}()
	m.artemisCmd = m.TrackingMasterTrackingCommand("localhost", leto.ARTEMIS_IN_PORT, "foo", config.Camera, config.Tag)
	if m.isMaster == true {
		dirname := filepath.Join(m.experimentDir, "ants")
		err = os.MkdirAll(dirname, 0644)
		if err != nil {
			return err
		}
		m.frameBuffer = bytes.NewBuffer(nil)
		m.artemisCmd.Stdout = m.frameBuffer
		m.quitEncode = make(chan struct{})
		m.wgEncode.Add(1)
		go m.encodeAndStream(config.Camera.FPS, config.BitRateKB, config.StreamHost)
	} else {
		m.artemisCmd.Stdout = nil
	}
	m.logger.Printf("Starting tracking")
	m.experimentName = config.ExperimentName
	m.since = time.Now()
	m.artemisCmd.Start()
	return nil
}

func (m *ArtemisManager) Stop() error {
	m.mx.Lock()
	defer m.mx.Unlock()
	if m.incoming == nil {
		return fmt.Errorf("Already stoppped")
	}
	if m.quitEncode != nil {
		close(m.quitEncode)
	}
	m.wgEncode.Wait()
	m.quitEncode = nil
	m.frameBuffer = nil
	m.artemisCmd.Process.Signal(os.Interrupt)
	m.artemisCmd.Wait()
	m.artemisCmd = nil
	m.frameBuffer = nil
	//Stops the reading of frame readout, it will close all the chain
	if err := m.trackers.Close(); err != nil {
		return err
	}
	m.wg.Wait()
	m.fileWriter.Close()
	m.incoming = nil
	m.merged = nil
	m.file = nil
	m.broadcast = nil
	return nil
}

func (m *ArtemisManager) TrackingMasterTrackingCommand(hostname string, port int, UUID string, camera leto.CameraConfiguration, detection leto.TagDetectionConfiguration) *exec.Cmd {
	args := []string{}
	args = append(args, "--host", hostname)
	args = append(args, "--port", fmt.Sprintf("%d", port))
	args = append(args, "--uuid", UUID)
	args = append(args, "--camera-fps", fmt.Sprintf("%f", camera.FPS))
	args = append(args, "--camera-strobe-us", fmt.Sprintf("%d", camera.StrobeDuration.Nanoseconds()/1000))
	args = append(args, "--camera-strobe-delay-us", fmt.Sprintf("%d", camera.StrobeDelay.Nanoseconds()/1000))
	args = append(args, "--at-family", detection.Family)
	args = append(args, "--at-quad-decimate", fmt.Sprintf("%f", detection.QuadDecimate))
	args = append(args, "--at-quad-sigma", fmt.Sprintf("%f", detection.QuadSigma))
	if detection.RefineEdges == true {
		args = append(args, "--at-refine-edges")
	}
	args = append(args, "--at-quad-min-cluster", fmt.Sprintf("%d", detection.QuadMinClusterPixel))
	args = append(args, "--at-quad-max-n-maxima", fmt.Sprintf("%d", detection.QuadMaxNMaxima))
	args = append(args, "--at-quad-critical-radian", fmt.Sprintf("%f", detection.QuadCriticalRadian))
	args = append(args, "--at-quad-max-line-mse", fmt.Sprintf("%f", detection.QuadMaxLineMSE))
	args = append(args, "--at-quad-min-bw-diff", fmt.Sprintf("%d", detection.QuadMinBWDiff))
	if detection.QuadDeglitch == true {
		args = append(args, "--at-quad-deglitch")
	}

	if m.isMaster == true {
		args = append(args, "--video-to-stdout")
		args = append(args, "--video-output-height", "1080")
		args = append(args, "--video-output-add-header")
	}

	cmd := exec.Command("artemis", args...)
	cmd.Stderr = nil
	cmd.Stdin = nil
	return cmd
}

func (m *ArtemisManager) encodeAndStream(fps float64, bitrate int, streamAddress string) {
	basenameMovie := filepath.Join(m.experimentDir, "stream.mp4")
	basenameFrame := filepath.Join(m.experimentDir, "stream.frame-matching.txt")
	var f *os.File
	var encodeCmd *exec.Cmd
	var streamCmd *exec.Cmd

	rawReader, rawWriter := io.Pipe()
	encodedReader, encodedWriter := io.Pipe()
	defer func() {
		if f != nil {
			f.Close()
		}
		if streamCmd != nil {

			streamCmd.Process.Signal(os.Interrupt)

		}
		if encodeCmd != nil {
			encodeCmd.Process.Signal(os.Interrupt)
		}
		if streamCmd != nil {
			streamCmd.Wait()
		}
		if encodeCmd != nil {
			encodeCmd.Wait()
		}
		rawReader.Close()
		rawWriter.Close()
		encodedReader.Close()
		encodedWriter.Close()
		m.wgEncode.Done()
	}()

	header := make([]byte, 3*8)
	host, err := os.Hostname()
	if err != nil {
		m.logger.Printf("%s", err)
		return
	}
	currentFrame := 0
	defer func() {
	}()

	period := 2 * time.Hour
	nextFile := time.Now().Add(period)

	for {
		//test if we need to quit
		select {
		case <-m.quitEncode:
			return
		default:
		}

		_, err := io.ReadFull(m.frameBuffer, header)
		if err != nil {
			m.logger.Printf("%s", err)
		}
		actual := binary.LittleEndian.Uint64(header)
		width := binary.LittleEndian.Uint64(header[8:])
		height := binary.LittleEndian.Uint64(header[16:])
		if encodeCmd == nil && streamCmd == nil && f == nil {
			mName, _, err := FilenameWithoutOverwrite(basenameMovie)
			cfName, _, err := FilenameWithoutOverwrite(basenameFrame)
			if err != nil {
				m.logger.Printf("%s", err)
				return
			}
			f, err = os.Create(cfName)
			if err != nil {
				m.logger.Printf("%s", err)
			}

			cbr := fmt.Sprintf("%dk", bitrate)
			res := fmt.Sprintf("%dx%d", width, height)
			quality := "ultrafast"
			encodeCmd = exec.Command("ffmpeg",
				"-f", "rawvideo",
				"-vcodec", "rawvideo",
				"-pixel_format", "rgb24",
				"-video_size", res,
				"-framerate", fmt.Sprintf("%f", fps),
				"-i", "-",
				"-c:v:0", "libx264",
				"-g", fmt.Sprintf("%d", int(2*fps)),
				"-keyint_min", fmt.Sprintf("%d", int(fps)),
				"-b:v", cbr,
				"-minrate", cbr,
				"-maxrate", cbr,
				"-pix_fmt",
				"yuv420p",
				"-s", res,
				"-preset", quality,
				"-tune", "film",
				"-f", "flv",
				"-")
			encodeCmd.Stderr = nil
			encodeCmd.Stdin = rawReader
			streamCmd = exec.Command("ffmpeg",
				"-hide_banner",
				"-loglevel", "error",
				"-f", "flv",
				"-i", "-",
				"-vcodec", "copy",
				mName)
			if len(streamAddress) > 0 {
				streamCmd.Args = append(streamCmd.Args,
					"-vcodec", "copy",
					fmt.Sprintf("rtmp://%s/olympus/%s.flv", streamAddress, host))
			}
			streamCmd.Stdout = encodedWriter
			streamCmd.Stdin = encodedReader
			m.logger.Printf("Starting streaming")
			encodeCmd.Start()
			streamCmd.Start()
		}

		fmt.Fprintf(f, "%d %d\n", currentFrame, actual)
		_, err = io.CopyN(rawWriter, m.frameBuffer, int64(3*width*height))
		if err != nil {
			m.logger.Printf("%s", err)
		}
		now := time.Now()
		if now.After(nextFile) == true {
			m.logger.Printf("Resetting streaming after %s", period)
			nextFile = now.Add(period)
			//we stop streaming
			streamCmd.Process.Signal(os.Interrupt)
			encodeCmd.Process.Signal(os.Interrupt)
			streamCmd.Wait()
			encodeCmd.Wait()
			f.Close()
			streamCmd = nil
			encodeCmd = nil
			f = nil
			rawWriter.Close()
			rawReader.Close()
			encodedReader.Close()
			encodedWriter.Close()
			rawReader, rawWriter = io.Pipe()
			encodedReader, encodedWriter = io.Pipe()
		}

	}

}
