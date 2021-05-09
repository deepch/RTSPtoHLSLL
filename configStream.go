package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"github.com/deepch/vdk/av"
)

const (
	FPSModeFixed = iota
	FPSModeSDP
	FPSModeSPS
	FPSModeProbe
	FPSModePTS
)

//Config global
var Config = loadConfig()

//ConfigST struct
type ConfigST struct {
	mutex   sync.RWMutex
	Server  ServerST            `json:"server"`
	Streams map[string]StreamST `json:"streams"`
}

//ServerST struct
type ServerST struct {
	HTTPName  string `json:"http_server_name"`
	HTTPPort  string `json:"http_port"`
	HTTPSPort string `json:"https_port"`
}

//StreamST struct
type StreamST struct {
	URL                   string    `json:"url"`
	Status                bool      `json:"status"`
	OnDemand              bool      `json:"on_demand"`
	FPSMode               string    `json:"fps_mode"`
	FPSProbeTime          int       `json:"fps_probe_time"`
	FPS                   int       `json:"fps"`
	HlsSegmentMinDuration int       `json:"hls_segment_min_duration"`
	HlsSegmentMaxSegments int       `json:"hls_segment_max_segments"`
	RunLock               bool      `json:"-"`
	HlsMuxer              *MuxerHLS `json:"-"`
	Codecs                []av.CodecData
}

//loadConfig func
func loadConfig() *ConfigST {
	var tmp ConfigST
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.Fatalln(err)
	}
	err = json.Unmarshal(data, &tmp)
	if err != nil {
		log.Fatalln(err)
	}
	return &tmp
}

//RunIFNotRun if not run
func (element *ConfigST) RunIFNotRun(uuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok {
		if tmp.OnDemand && !tmp.RunLock {
			tmp.RunLock = true
			element.Streams[uuid] = tmp
			go RTSPWorkerLoop(uuid, tmp.URL)
		}
	}
}

//RunUnlock lock run stream
func (element *ConfigST) RunUnlock(uuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok {
		if tmp.OnDemand && tmp.RunLock {
			tmp.RunLock = false
			element.Streams[uuid] = tmp
		}
	}
}

//FPSMode func
func (element *ConfigST) FPSMode(uuid string) int {
	element.mutex.RLock()
	defer element.mutex.RUnlock()
	if tmp, ok := element.Streams[uuid]; ok {
		switch tmp.FPSMode {
		case "fixed":
			return FPSModeFixed
		case "sdp":
			return FPSModeSDP
		case "sps":
			return FPSModeSPS
		case "probe":
			return FPSModeProbe
		case "pts":
			return FPSModePTS
		default:
			return FPSModeProbe
		}
	}
	return FPSModeProbe
}

//ext check stream exists
func (element *ConfigST) ext(uuid string) bool {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	_, ok := element.Streams[uuid]
	return ok
}

//coAd add codec to stream
func (element *ConfigST) coAd(uuid string, codecs []av.CodecData) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	t := element.Streams[uuid]
	t.Codecs = codecs
	element.Streams[uuid] = t
}

//HttpName func
func (element *ConfigST) HttpName() string {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	return element.Server.HTTPName
}

//HttpPort func
func (element *ConfigST) HttpPort() string {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	return element.Server.HTTPPort
}

//HttpsPort func
func (element *ConfigST) HttpsPort() string {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	return element.Server.HTTPSPort
}

//coGe get stream codec
func (element *ConfigST) coGe(uuid string) []av.CodecData {
	for i := 0; i < 100; i++ {
		element.mutex.RLock()
		tmp, ok := element.Streams[uuid]
		element.mutex.RUnlock()
		if !ok {
			return nil
		}
		if tmp.Codecs != nil {
			return tmp.Codecs
		}
		//TODO add ctx wait codec data
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

//list return all stream list
func (element *ConfigST) list() (string, []string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	var res []string
	var fist string
	for k := range element.Streams {
		if fist == "" {
			fist = k
		}
		res = append(res, k)
	}
	return fist, res
}

//NewHLSMuxer new muxer init
func (element *ConfigST) NewHLSMuxer(uuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok {
		tmp.HlsMuxer = NewHLSMuxer(uuid)
		element.Streams[uuid] = tmp
	}
}

//HlsMuxerSetFPS write packet
func (element *ConfigST) HlsMuxerSetFPS(uuid string, fps int) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok {
		tmp.HlsMuxer.SetFPS(fps)
	}
}

//HlsMuxerWritePacket write packet
func (element *ConfigST) HlsMuxerWritePacket(uuid string, packet *av.Packet) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok {
		tmp.HlsMuxer.WritePacket(packet)
	}
}

//HLSMuxerClose close muxer
func (element *ConfigST) HLSMuxerClose(uuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok {
		tmp.HlsMuxer.Close()
	}
}

//HLSMuxerM3U8 get m3u8 list
func (element *ConfigST) HLSMuxerM3U8(uuid string, msn, part int) (string, error) {
	element.mutex.Lock()
	tmp, ok := element.Streams[uuid]
	element.mutex.Unlock()
	if !ok {
		return "", ErrorStreamNotFound
	}
	playlist, got, waitQ, session, err := tmp.HlsMuxer.GetIndexM3u8(msn, part)
	if err != nil {
		return "", err
	}
	if got {
		return playlist, nil
	}
	timer := time.NewTimer(time.Second * 10)
	select {
	case <-timer.C:
		tmp.HlsMuxer.CloseSessionIndex(session)
		return "", ErrorStreamIndexTimeout
	case playlist = <-waitQ:
		return playlist, nil
	}
}

//HLSMuxerSegment get segment
func (element *ConfigST) HLSMuxerSegment(uuid string, segment int) ([]*av.Packet, error) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok {
		return tmp.HlsMuxer.GetSegment(segment)
	}
	return nil, ErrorStreamSegmentNotFound
}

//HLSMuxerFragment get fragment
func (element *ConfigST) HLSMuxerFragment(uuid string, segment, fragment int) ([]*av.Packet, error) {
	element.mutex.Lock()
	tmp, ok := element.Streams[uuid]
	element.mutex.Unlock()
	if !ok {
		return nil, ErrorStreamNotFound
	}
	buf, got, waitQ, session, err := tmp.HlsMuxer.GetFragment(segment, fragment)

	if err != nil {
		return nil, err
	}
	if got {
		return buf, nil
	}
	timer := time.NewTimer(time.Second * 2)
	select {
	case <-timer.C:
		tmp.HlsMuxer.CloseSessionPart(session)
		return nil, ErrorStreamFragmentTimeout
	case <-waitQ:
		buf, got, _, _, err = tmp.HlsMuxer.GetFragment(segment, fragment)
		if err != nil {
			return nil, err
		}
		if got {
			return buf, nil
		}
	}
	return nil, ErrorStreamFragmentNotFound
}
