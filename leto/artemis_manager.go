package main

import (
	"fmt"
	"io"
	"log"
	"net"
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
	wg                                sync.WaitGroup
	quitEncode                        chan struct{}
	fileWriter                        *FrameReadoutFileWriter
	trackers                          *RemoteManager
	isMaster                          bool

	artemisCmd    *exec.Cmd
	artemisOut    *io.PipeWriter
	streamIn      *io.PipeReader
	streamManager *StreamManager
	artemisLog    *os.File

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
		logger:   log.New(os.Stderr, "[artemis] ", log.LstdFlags),
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

func (m *ArtemisManager) ExperimentDir(expname string, testMode bool) (string, error) {
	if testMode == false {
		basename := filepath.Join(xdg.DataHome, "fort-experiments", expname)
		basedir, _, err := FilenameWithoutOverwrite(basename)
		return basedir, err
	}
	basename := filepath.Join(os.TempDir(), "fort-tests", expname)
	basedir, _, err := FilenameWithoutOverwrite(basename)
	return basedir, err
}

func (*ArtemisManager) LinkHostname(address string) error {
	return fmt.Errorf("Work balancing with multiple host is not yet implemented")
}

func (*ArtemisManager) UnlinkHostname(address string) error {
	return fmt.Errorf("Work balancing with multiple host is not yet implemented")
}

func (m *ArtemisManager) LoadDefaultConfig() *leto.TrackingConfiguration {
	res := leto.RecommendedTrackingConfiguration()
	systemConfig, err := leto.ReadConfiguration("/etc/default/leto.yml")
	if err != nil {
		m.logger.Printf("Could not load system configuration: %s", err)
		return &res
	}

	err = res.Merge(systemConfig)
	if err != nil {
		m.logger.Printf("Could not merge system configuration: %s", err)
		m.logger.Printf("Reverting to library default configuration")
		res = leto.RecommendedTrackingConfiguration()
	}

	return &res
}

func (m *ArtemisManager) Start(userConfig *leto.TrackingConfiguration) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	if m.incoming != nil {
		return fmt.Errorf("ArtemisManager: Start: already started")
	}

	config := m.LoadDefaultConfig()

	if err := config.Merge(userConfig); err != nil {
		return fmt.Errorf("could not merge user configuration: %s", err)
	}

	if err := config.CheckAllFieldAreSet(); err != nil {
		return fmt.Errorf("incomplete configuration: %s", err)
	}

	testMode := false
	if len(config.ExperimentName) == 0 {
		m.logger.Printf("Starting in test mode")
		testMode = true
		//disable streaming
		*config.Stream.Host = ""
		// enforces display
		*config.DisplayOnHost = true

		config.ExperimentName = "!!IN TEST MODE!!"
	} else {
		m.logger.Printf("New experiment '%s'", config.ExperimentName)
	}

	m.incoming = make(chan *hermes.FrameReadout, 10)
	m.merged = make(chan *hermes.FrameReadout, 10)
	m.file = make(chan *hermes.FrameReadout, 200)
	m.broadcast = make(chan *hermes.FrameReadout, 10)
	var err error
	m.experimentDir, err = m.ExperimentDir(config.ExperimentName, testMode)
	if err != nil {
		return err
	}
	err = os.MkdirAll(m.experimentDir, 0755)
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
		FPS:       *config.Camera.FPS,
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
		err := m.trackers.Listen(fmt.Sprintf(":%d", leto.ARTEMIS_IN_PORT), m.OnAccept(), func() {
			m.logger.Printf("All connection closed, cleaning up experiment")
			close(m.incoming)
			m.mx.Lock()
			defer m.mx.Unlock()
			m.incoming = nil
		})
		if err != nil {
			m.logger.Printf("listening for tracker unhandled error: %s", err)
		}
		m.wg.Done()
	}()
	m.wg.Add(1)
	go func() {
		BroadcastFrameReadout(fmt.Sprintf(":%d", leto.ARTEMIS_OUT_PORT),
			m.broadcast,
			3*time.Duration(1.0e6/(*config.Camera.FPS))*time.Microsecond)
		m.wg.Done()
	}()
	m.wg.Add(1)
	go func() {
		m.fileWriter.WriteAll(m.file)
		m.wg.Done()
	}()

	logFilePath := filepath.Join(m.experimentDir, "artemis.log")
	m.artemisLog, err = os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("Could not create artemis log file ('%s'): %s", logFilePath, err)
	}

	m.artemisCmd = m.TrackingMasterTrackingCommand("localhost", leto.ARTEMIS_IN_PORT, "foo", config.Camera, config.Detection, testMode, *config.LegacyMode)
	m.artemisCmd.Stderr = m.artemisLog
	m.artemisCmd.Stdin = nil
	if *config.DisplayOnHost == true {
		m.artemisCmd.Args = append(m.artemisCmd.Args, "-d")
	}
	if m.isMaster == true {
		dirname := filepath.Join(m.experimentDir, "ants")
		err = os.MkdirAll(dirname, 0755)
		if err != nil {
			return err
		}
		m.artemisCmd.Args = append(m.artemisCmd.Args, "--new-ant-output-dir", dirname,
			"--new-ant-roi-size", fmt.Sprintf("%d", *config.NewAntOutputROISize))
		m.streamIn, m.artemisOut = io.Pipe()
		m.artemisCmd.Stdout = m.artemisOut
		m.streamManager, err = NewStreamManager(m.experimentDir, *config.Camera.FPS, config.Stream)
		if err != nil {
			return err
		}
		go m.streamManager.EncodeAndStreamMuxedStream(m.streamIn)
	} else {
		m.artemisCmd.Stdout = nil
	}
	m.logger.Printf("Starting tracking for '%s'", config.ExperimentName)
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

	m.artemisCmd.Process.Signal(os.Interrupt)
	m.logger.Printf("Waiting for artemis process to stop")
	m.artemisCmd.Wait()
	m.artemisCmd = nil

	err := m.artemisLog.Close()
	if err != nil {
		m.logger.Printf("Could no close artemis log file: %s", err)
	}

	//Stops the reading of frame readout, it will close all the chain
	if err := m.trackers.Close(); err != nil {
		return err
	}
	log.Printf("Waiting for all connection to be closed")
	m.mx.Unlock()
	m.wg.Wait()
	m.fileWriter.Close()
	m.mx.Lock()

	if m.streamManager != nil {
		m.logger.Printf("Waiting for stream tasks to stop")
		m.artemisOut.Close()
		m.streamManager.Wait()
		m.streamManager = nil
		m.streamIn.Close()
		m.artemisOut = nil
		m.streamIn = nil
	}
	m.incoming = nil
	m.merged = nil
	m.file = nil
	m.broadcast = nil
	m.logger.Printf("Experiment '%s' done", m.experimentName)
	return nil
}

func (m *ArtemisManager) TrackingMasterTrackingCommand(hostname string, port int, UUID string, camera leto.CameraConfiguration, detection leto.TagDetectionConfiguration, testMode, legacyMode bool) *exec.Cmd {
	args := []string{}
	if testMode == false {
		args = append(args, "--host", hostname)
		args = append(args, "--port", fmt.Sprintf("%d", port))
		args = append(args, "--uuid", UUID)
	}
	if legacyMode == true {
		args = append(args, "--legacy-mode")
	}
	args = append(args, "--camera-fps", fmt.Sprintf("%f", *camera.FPS))
	args = append(args, "--camera-strobe-us", fmt.Sprintf("%d", camera.StrobeDuration.Nanoseconds()/1000))
	args = append(args, "--camera-strobe-delay-us", fmt.Sprintf("%d", camera.StrobeDelay.Nanoseconds()/1000))
	args = append(args, "--at-family", *detection.Family)
	args = append(args, "--at-quad-decimate", fmt.Sprintf("%f", *detection.Quad.Decimate))
	args = append(args, "--at-quad-sigma", fmt.Sprintf("%f", *detection.Quad.Sigma))
	if *detection.Quad.RefineEdges == true {
		args = append(args, "--at-refine-edges")
	}
	args = append(args, "--at-quad-min-cluster", fmt.Sprintf("%d", *detection.Quad.MinClusterPixel))
	args = append(args, "--at-quad-max-n-maxima", fmt.Sprintf("%d", *detection.Quad.MaxNMaxima))
	args = append(args, "--at-quad-critical-radian", fmt.Sprintf("%f", *detection.Quad.CriticalRadian))
	args = append(args, "--at-quad-max-line-mse", fmt.Sprintf("%f", *detection.Quad.MaxLineMSE))
	args = append(args, "--at-quad-min-bw-diff", fmt.Sprintf("%d", *detection.Quad.MinBWDiff))
	if *detection.Quad.Deglitch == true {
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

func (m *ArtemisManager) OnAccept() func(c net.Conn) {
	return func(c net.Conn) {
		errors := make(chan error)
		logger := log.New(os.Stderr, fmt.Sprintf("[artemis/%s] ", c.RemoteAddr().String()), log.LstdFlags)
		logger.Printf("new connection from %s", c.RemoteAddr().String())
		go func() {
			for e := range errors {
				logger.Printf("unhandled error: %s", e)
			}
		}()
		FrameReadoutReadAll(c, m.incoming, errors)
	}
}
