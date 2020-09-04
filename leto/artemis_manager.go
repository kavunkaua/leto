package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/blang/semver"
	"github.com/formicidae-tracker/hermes"
	"github.com/formicidae-tracker/leto"
	"github.com/google/uuid"
)

type ArtemisManager struct {
	incoming, merged, file, broadcast chan *hermes.FrameReadout
	mx                                sync.Mutex
	wg, artemisWg                     sync.WaitGroup
	quitEncode                        chan struct{}
	fileWriter                        *FrameReadoutFileWriter
	trackers                          *RemoteManager
	nodeConfig                        NodeConfiguration

	artemisCmd    *exec.Cmd
	artemisOut    *io.PipeWriter
	streamIn      *io.PipeReader
	streamManager *StreamManager
	testMode      bool

	experimentDir string
	logger        *log.Logger

	experimentName string
	since          time.Time
}

func CheckArtemisVersion(actual, minimal string) error {
	a, err := semver.ParseTolerant(actual)
	if err != nil {
		return err
	}
	m, err := semver.ParseTolerant(minimal)
	if err != nil {
		return err
	}

	if m.Major == 0 {
		if a.Major != 0 || a.Minor != m.Minor {
			return fmt.Errorf("Unexpected major version v%d.%d (expected: v%d.%d)", a.Major, a.Minor, m.Major, m.Minor)
		}
	} else if m.Major != a.Major {
		return fmt.Errorf("Unexpected major version v%d (expected: v%d)", a.Major, m.Major)
	}

	if a.GE(m) == false {
		return fmt.Errorf("Invalid version v%s (minimal: v%s)", a, m)
	}

	return nil
}

func extractCoaxlinkFirmwareOutput(output []byte) (string, error) {
	rx := regexp.MustCompile(`Firmware variant:\W+[0-9]+\W+\(([0-9a-z\-]+)\)`)
	m := rx.FindStringSubmatch(string(output))
	if len(m) == 0 {
		return "", fmt.Errorf("Could not determine firmware variant in output: '%s'", output)
	}
	return m[1], nil

}

func getFirmwareVariant() (string, error) {
	cmd := exec.Command("coaxlink-firmware")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("Could not check slave firmware variant")
	}

	return extractCoaxlinkFirmwareOutput(output)
}

func CheckFirmwareVariant(c NodeConfiguration, variant string, checkMaster bool) error {
	expected := "1-camera"
	if c.IsMaster() == false {
		expected = "1-df-camera"
	} else if checkMaster == false {
		return nil
	}

	if variant != expected {
		return fmt.Errorf("Unexpected firmware variant %s (expected: %s)", variant, expected)
	}

	return nil
}

func getAndCheckFirmwareVariant(c NodeConfiguration, checkMaster bool) error {
	variant, err := getFirmwareVariant()
	if err != nil {
		if c.IsMaster() && checkMaster == false {
			return nil
		}
		return err
	}
	return CheckFirmwareVariant(c, variant, checkMaster)
}

func NewArtemisManager() (*ArtemisManager, error) {
	cmd := exec.Command("artemis", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Could not find artemis: %s", err)
	}

	artemisVersion := strings.TrimPrefix(strings.TrimSpace(string(output)), "artemis ")
	err = CheckArtemisVersion(artemisVersion, leto.ARTEMIS_MIN_VERSION)
	if err != nil {
		return nil, err
	}

	cmd = exec.Command("ffmpeg", "-version")
	_, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Could not find ffmpeg: %s", err)
	}

	nodeConfig := GetNodeConfiguration()

	err = getAndCheckFirmwareVariant(nodeConfig, false)
	if err != nil {
		return nil, err
	}

	return &ArtemisManager{
		nodeConfig: nodeConfig,
		logger:     log.New(os.Stderr, "[artemis] ", log.LstdFlags),
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
	if m.testMode == false {
		basename := filepath.Join(xdg.DataHome, "fort-experiments", expname)
		basedir, _, err := FilenameWithoutOverwrite(basename)
		return basedir, err
	}
	basename := filepath.Join(os.TempDir(), "fort-tests", expname)
	basedir, _, err := FilenameWithoutOverwrite(basename)
	return basedir, err
}

func (m *ArtemisManager) SetMaster(hostname string) (err error) {
	defer func() {
		if err == nil {
			m.nodeConfig.Save()
		}
	}()

	if len(hostname) == 0 {
		m.nodeConfig.Master = ""
		return
	}

	if len(m.nodeConfig.Slaves) != 0 {
		err = fmt.Errorf("Cannot set node as slave as it has its own slaves (%s)", m.nodeConfig.Slaves)
		return
	}
	m.nodeConfig.Master = hostname
	err = getAndCheckFirmwareVariant(m.nodeConfig, true)
	if err != nil {
		m.nodeConfig.Master = ""
	}
	return
}

func (m *ArtemisManager) AddSlave(hostname string) (err error) {
	defer func() {
		if err == nil {
			m.nodeConfig.Save()
		}
	}()

	err = m.SetMaster("")
	if err != nil {
		return
	}
	err = getAndCheckFirmwareVariant(m.nodeConfig, true)
	if err != nil {
		return
	}

	err = m.nodeConfig.AddSlave(hostname)
	return
}

func (m *ArtemisManager) RemoveSlave(hostname string) (err error) {
	defer func() {
		if err == nil {
			m.nodeConfig.Save()
		}
	}()

	return m.nodeConfig.RemoveSlave(hostname)
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

func GenerateLoadBalancing(c NodeConfiguration) *leto.LoadBalancing {
	if len(c.Slaves) == 0 {
		return &leto.LoadBalancing{
			SelfUUID:     "single-node",
			UUIDs:        map[string]string{"localhost": "single-node"},
			Assignements: map[int]string{0: "single-node"},
		}
	}
	res := &leto.LoadBalancing{
		SelfUUID:     uuid.New().String(),
		UUIDs:        make(map[string]string),
		Assignements: make(map[int]string),
	}
	res.UUIDs["localhost"] = res.SelfUUID
	res.Assignements[0] = res.SelfUUID
	for i, s := range c.Slaves {
		uuid := uuid.New().String()
		res.UUIDs[s] = uuid
		res.Assignements[i+1] = uuid
	}
	return res
}

func BuildWorkloadBalance(lb *leto.LoadBalancing, FPS float64) *WorkloadBalance {
	wb := &WorkloadBalance{
		FPS:        FPS,
		MasterUUID: lb.UUIDs["localhost"],
		Stride:     len(lb.Assignements),
		IDsByUUID:  make(map[string][]bool),
	}

	for id, uuid := range lb.Assignements {
		if _, ok := wb.IDsByUUID[uuid]; ok == false {
			wb.IDsByUUID[uuid] = make([]bool, len(lb.Assignements))
		}
		wb.IDsByUUID[uuid][id] = true

	}
	return wb
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

	if m.nodeConfig.IsMaster() == true {
		config.Loads = GenerateLoadBalancing(m.nodeConfig)
		if len(m.nodeConfig.Slaves) > 0 {
			cmd := exec.Command("artemis", "--fetch-resolution")

			if config.Camera.StubPaths != nil || len(*config.Camera.StubPaths) > 0 {
				cmd.Args = append(cmd.Args, "--stub-image-paths", strings.Join(*config.Camera.StubPaths, ","))
			}

			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("Could not determine camera resolution: %s", err)
			}
			_, err = fmt.Sscanf(string(out), "%d %d", &config.Loads.Width, &config.Loads.Height)
			if err != nil {
				return fmt.Errorf("Could not parse camera resolution in '%s'", out)
			}
		}
	}

	if err := config.CheckAllFieldAreSet(); err != nil {
		return fmt.Errorf("incomplete configuration: %s", err)
	}

	m.testMode = false
	if len(config.ExperimentName) == 0 {
		m.logger.Printf("Starting in test mode")
		m.testMode = true
		// enforces display
		config.ExperimentName = "TEST-MODE"
	} else {
		m.logger.Printf("New experiment '%s'", config.ExperimentName)
	}

	m.incoming = make(chan *hermes.FrameReadout, 10)

	if m.nodeConfig.IsMaster() == true {
		m.merged = make(chan *hermes.FrameReadout, 10)
		m.file = make(chan *hermes.FrameReadout, 200)
		m.broadcast = make(chan *hermes.FrameReadout, 10)
	}

	var err error
	m.experimentDir, err = m.ExperimentDir(config.ExperimentName)
	if err != nil {
		return err
	}
	err = os.MkdirAll(m.experimentDir, 0755)
	if err != nil {
		return err
	}

	//save the config to the experiment dir
	confSaveName := filepath.Join(m.experimentDir, "leto-final-config.yml")
	err = config.WriteConfiguration(confSaveName)
	if err != nil {
		return err
	}

	wb := BuildWorkloadBalance(config.Loads, *config.Camera.FPS)

	if m.nodeConfig.IsMaster() == true {
		m.fileWriter, err = NewFrameReadoutWriter(filepath.Join(m.experimentDir, "tracking.hermes"))
		if err != nil {
			return err
		}

		m.trackers = NewRemoteManager()

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
	}

	logFilePath := filepath.Join(m.experimentDir, "artemis.command")
	artemisCommandLog, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("Could not create artemis log file ('%s'): %s", logFilePath, err)
	}
	defer artemisCommandLog.Close()

	targetHost := "localhost"
	if m.nodeConfig.IsMaster() == false {
		targetHost = strings.TrimPrefix(m.nodeConfig.Master, "leto.") + ".local"
	}
	m.artemisCmd = m.TrackingCommand(targetHost, leto.ARTEMIS_IN_PORT, config.Loads.SelfUUID, config.Camera, config.Detection, *config.LegacyMode, wb)
	m.artemisCmd.Stderr, err = os.Create(filepath.Join(m.experimentDir, "artemis.stderr"))
	m.artemisCmd.Args = append(m.artemisCmd.Args, "--log-output-dir", m.experimentDir)
	m.artemisCmd.Stdin = nil

	tags := make([]string, 0, len(*config.Highlights))
	for _, id := range *config.Highlights {
		tags = append(tags, "0x"+strconv.FormatUint(uint64(id), 16))
	}

	if len(tags) != 0 {
		m.artemisCmd.Args = append(m.artemisCmd.Args, "--highlight-tags", strings.Join(tags, ","))
	}

	if m.nodeConfig.IsMaster() == true {
		dirname := filepath.Join(m.experimentDir, "ants")
		err = os.MkdirAll(dirname, 0755)
		if err != nil {
			return err
		}
		m.artemisCmd.Args = append(m.artemisCmd.Args, "--new-ant-output-dir", dirname,
			"--new-ant-roi-size", fmt.Sprintf("%d", *config.NewAntOutputROISize),
			"--image-renew-period", fmt.Sprintf("%s", config.NewAntRenewPeriod))
		m.streamIn, m.artemisOut = io.Pipe()
		m.artemisCmd.Stdout = m.artemisOut
		m.streamManager, err = NewStreamManager(m.experimentDir, *config.Camera.FPS/float64(wb.Stride), config.Stream)
		if err != nil {
			return err
		}
		go m.streamManager.EncodeAndStreamMuxedStream(m.streamIn)
	} else {
		m.artemisCmd.Stdout = nil
		m.artemisCmd.Args = append(m.artemisCmd.Args,
			"--camera-slave-width", fmt.Sprintf("%d", config.Loads.Width),
			"--camera-slave-height", fmt.Sprintf("%d", config.Loads.Height))
	}

	m.logger.Printf("Starting tracking for '%s'", config.ExperimentName)
	m.experimentName = config.ExperimentName
	m.since = time.Now()
	fmt.Fprintf(artemisCommandLog, "%s %s\n", m.artemisCmd.Path, m.artemisCmd.Args)

	if m.nodeConfig.IsMaster() {
		//Starts all slaves
		for _, s := range m.nodeConfig.Slaves {
			slaveConfig := *config
			slaveConfig.Loads.SelfUUID = slaveConfig.Loads.UUIDs[s]
			resp := leto.Response{}
			_, _, err := leto.RunForHost(s, "Leto.StartTracking", &slaveConfig, &resp)
			if err == nil {
				err = resp.ToError()
			}
			if err != nil {
				m.logger.Printf("Could not start slave %s: %s", s, err)
			}
		}
	}

	m.artemisWg.Add(1)
	go func() {
		defer m.artemisWg.Done()
		err := m.artemisCmd.Run()
		m.mx.Lock()
		defer m.mx.Unlock()
		cleanDir := true
		if err != nil {
			cleanDir = false
			m.logger.Printf("artemis child process exited with error: %s", err)
			if m.testMode == true {
				logs, errF := ioutil.ReadFile(filepath.Join(m.experimentDir, "artemis.stderr"))
				if errF == nil {
					m.logger.Printf("Artemis STDERR:\n%s", logs)
				}
			}
		}
		m.artemisCmd = nil
		//Stops the reading of frame readout, it will close all the chain
		if m.trackers != nil {
			err = m.trackers.Close()
			if err != nil {
				m.logger.Printf("Could not close connections: %s", err)
			}
		}

		log.Printf("Waiting for all connection to be closed")
		m.mx.Unlock()
		m.wg.Wait()
		if m.fileWriter != nil {
			m.fileWriter.Close()
		}
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

		if m.testMode == true && cleanDir == true {
			log.Printf("Cleaning '%s'", m.experimentDir)
			if err := os.RemoveAll(m.experimentDir); err != nil {
				log.Printf("Could not clean '%s': %s", m.experimentDir, err)
			}
		}
	}()

	return nil
}

func (m *ArtemisManager) Stop() error {
	m.mx.Lock()
	defer m.mx.Unlock()

	if m.incoming == nil {
		return fmt.Errorf("Already stoppped")
	}

	if m.artemisCmd != nil {
		if m.nodeConfig.IsMaster() == true {
			for _, s := range m.nodeConfig.Slaves {
				resp := leto.Response{}
				_, _, err := leto.RunForHost(s, "Leto.StopTracking", &leto.TrackingStop{}, &resp)
				if err == nil {
					err = resp.ToError()
				}
				if err != nil {
					m.logger.Printf("Could not stop slave %s: %s", s, err)
				}
			}
		}

		m.artemisCmd.Process.Signal(os.Interrupt)
		m.logger.Printf("Waiting for artemis process to stop")
		m.artemisCmd = nil
	}

	m.mx.Unlock()
	m.artemisWg.Wait()
	m.mx.Lock()
	return nil
}

func (m *ArtemisManager) TrackingCommand(hostname string, port int, UUID string, camera leto.CameraConfiguration, detection leto.TagDetectionConfiguration, legacyMode bool, wb *WorkloadBalance) *exec.Cmd {
	args := []string{}

	if len(*camera.StubPaths) != 0 {
		args = append(args, "--stub-image-paths", strings.Join(*camera.StubPaths, ","))
	}

	if m.testMode == true {
		args = append(args, "--test-mode")
	}
	args = append(args, "--host", hostname)
	args = append(args, "--port", fmt.Sprintf("%d", port))
	args = append(args, "--uuid", UUID)

	if legacyMode == true {
		args = append(args, "--legacy-mode")
	}
	args = append(args, "--camera-fps", fmt.Sprintf("%f", *camera.FPS))
	args = append(args, "--camera-strobe", fmt.Sprintf("%s", camera.StrobeDuration))
	args = append(args, "--camera-strobe-delay", fmt.Sprintf("%s", camera.StrobeDelay))
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

	if m.nodeConfig.IsMaster() == true {
		args = append(args, "--video-output-to-stdout")
		args = append(args, "--video-output-height", "1080")
		args = append(args, "--video-output-add-header")
	}

	if len(wb.IDsByUUID) > 1 {
		args = append(args, "--frame-stride", fmt.Sprintf("%d", len(wb.IDsByUUID)))
		ids := []string{}
		for i, isSet := range wb.IDsByUUID[UUID] {
			if isSet == false {
				continue
			}
			ids = append(ids, fmt.Sprintf("%d", i))
		}
		args = append(args, "--frame-ids", strings.Join(ids, ","))
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
