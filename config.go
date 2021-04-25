package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/deepch/vdk/av"
)

var (
	Success                         = "success"
	ErrorStreamNotFound             = errors.New("stream not found")
	ErrorStreamAlreadyExists        = errors.New("stream already exists")
	ErrorStreamChannelAlreadyExists = errors.New("stream channel already exists")
	ErrorStreamNotHLSSegments       = errors.New("stream hls not ts seq found")
	ErrorStreamNoVideo              = errors.New("stream no video")
	ErrorStreamNoClients            = errors.New("stream no clients")
	ErrorStreamRestart              = errors.New("stream restart")
	ErrorStreamStopCoreSignal       = errors.New("stream stop core signal")
	ErrorStreamStopRTSPSignal       = errors.New("stream stop rtsp signal")
	ErrorStreamChannelNotFound      = errors.New("stream channel not found")
	ErrorStreamChannelCodecNotFound = errors.New("stream channel codec not ready, possible stream offline")
	ErrorStreamsLen0                = errors.New("streams len zero")
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
	HTTPPort string `json:"http_port"`
}

//StreamST struct
type StreamST struct {
	URL                    string              `json:"url"`
	Status                 bool                `json:"status"`
	OnDemand               bool                `json:"on_demand"`
	RunLock                bool                `json:"-"`
	hlsCursorMSN           int                 `json:"-"`
	hlsCursorPacket        int                 `json:"-"`
	hlsCursorPart          int                 `json:"-"`
	hlsCursorMediaSequence int                 `json:"-"`
	HlsTD                  float64             `json:"-"`
	hlsWaitPart            map[string]WaitPart `json:"-"`
	HlsFD                  float64             `json:"-"`
	hlsSegmentNumber       int                 `json:"-"`
	hlsFragmentNumber      int                 `json:"-"`
	hlsPckNumber           int                 `json:"-"`
	hlsLastSeq             time.Time           `json:"-"`
	hlsSegmentBuffer       map[int]Segment     `json:"-"`
	Codecs                 []av.CodecData
	Cl                     map[string]viewer
	hlsRealPart            int
}

//Segment element
type Segment struct {
	Finish    bool
	duration  time.Duration
	fragments []Fragment
	Time      string
}

//Fragment element
type Fragment struct {
	Finish      bool
	duration    time.Duration
	packets     []*av.Packet
	programTime string
}

//WaitPart element
type WaitPart struct {
	ch   chan string
	msn  int
	part int
}
type viewer struct {
	c chan av.Packet
}

func (element *ConfigST) RunIFNotRun(uuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok {
		if tmp.OnDemand && !tmp.RunLock {
			tmp.RunLock = true
			element.Streams[uuid] = tmp
			go RTSPWorkerLoop(uuid, tmp.URL, tmp.OnDemand)
		}
	}
}

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

func (element *ConfigST) HasViewer(uuid string) bool {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok && len(tmp.Cl) > 0 {
		return true
	}
	return false
}

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
	for i, v := range tmp.Streams {
		v.Cl = make(map[string]viewer)
		v.hlsWaitPart = make(map[string]WaitPart)
		v.hlsSegmentBuffer = make(map[int]Segment)
		v.hlsCursorMSN = -1
		tmp.Streams[i] = v
	}
	return &tmp
}

func (element *ConfigST) cast(uuid string, pck av.Packet) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	for _, v := range element.Streams[uuid].Cl {
		if len(v.c) < cap(v.c) {
			v.c <- pck
		}
	}
}

func (element *ConfigST) ext(suuid string) bool {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	_, ok := element.Streams[suuid]
	return ok
}

func (element *ConfigST) coAd(suuid string, codecs []av.CodecData) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	t := element.Streams[suuid]
	t.Codecs = codecs
	element.Streams[suuid] = t
}

func (element *ConfigST) coGe(suuid string) []av.CodecData {
	for i := 0; i < 100; i++ {
		element.mutex.RLock()
		tmp, ok := element.Streams[suuid]
		element.mutex.RUnlock()
		if !ok {
			return nil
		}
		if tmp.Codecs != nil {
			return tmp.Codecs
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func (element *ConfigST) clAd(suuid string) (string, chan av.Packet) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	cuuid := pseudoUUID()
	ch := make(chan av.Packet, 100)
	element.Streams[suuid].Cl[cuuid] = viewer{c: ch}
	return cuuid, ch
}

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

func (element *ConfigST) clDe(suuid, cuuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	delete(element.Streams[suuid].Cl, cuuid)
}

//StreamHLSAdd add hls seq to buffer
func (element *ConfigST) StreamHLSm3u8Update(uuid string) {
	element.mutex.Lock()
	element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok && len(tmp.hlsWaitPart) > 0 {
		index, _, _, _ := element.StreamHLSm3u8(uuid, false)
		for i, i2 := range tmp.hlsWaitPart {
			if i2.msn < tmp.hlsCursorMSN || i2.part <= tmp.hlsCursorPart-1 {
				i2.ch <- index
				delete(tmp.hlsWaitPart, i)
			}
		}
		element.Streams[uuid] = tmp
	}
}

//StreamHLSAdd add hls seq to buffer
func (element *ConfigST) StreamHLSAdd(uuid string, val *av.Packet) {
	element.mutex.Lock()
	defer func() {
		element.mutex.Unlock()
		element.StreamHLSm3u8Update(uuid)
	}()
	if tmp, ok := element.Streams[uuid]; ok {
		if val.IsKeyFrame && time.Now().Sub(tmp.hlsLastSeq).Seconds() > 1 {
			if tmpSegment, ok := tmp.hlsSegmentBuffer[tmp.hlsCursorMSN]; ok {
				tmpSegment.Finish = true
				tmp.HlsTD = tmpSegment.duration.Seconds()
				tmpSegment.Time = time.Now().Format("2006-01-02T15:04:05.000000Z")
				tmp.hlsSegmentBuffer[tmp.hlsCursorMSN] = tmpSegment
			}
			tmp.hlsCursorMSN++

			tmp.hlsCursorPart = -1
		}
		if tmp.hlsCursorPacket == 0 {
			tmp.hlsCursorPart++
			if tmp.hlsCursorPart-1 >= 0 {
				tmp.hlsSegmentBuffer[tmp.hlsCursorMSN].fragments[tmp.hlsCursorPart-1].Finish = true
			} else if tmp.hlsCursorMSN > 0 {
				tmp.hlsSegmentBuffer[tmp.hlsCursorMSN-1].fragments[9].Finish = true
			}
		}
		tmp.hlsCursorPacket++
		if tmp.hlsCursorPacket == 5 {
			tmp.hlsCursorPacket = 0
		}
		tmpSegment, _ := tmp.hlsSegmentBuffer[tmp.hlsCursorMSN]
		tmpSegment.duration += val.Duration
		if len(tmpSegment.fragments) < tmp.hlsCursorPart+1 {
			tmpSegment.fragments = append(tmpSegment.fragments, Fragment{})
		}
		tmpSegment.fragments[tmp.hlsCursorPart].duration += val.Duration
		tmpSegment.fragments[tmp.hlsCursorPart].packets = append(tmpSegment.fragments[tmp.hlsCursorPart].packets, val)
		tmp.hlsSegmentBuffer[tmp.hlsCursorMSN] = tmpSegment
		if len(tmp.hlsSegmentBuffer) > 6 {
			delete(tmp.hlsSegmentBuffer, tmp.hlsCursorMSN-6)
			tmp.hlsCursorMediaSequence++
		}
		element.Streams[uuid] = tmp
	}
}
func (element *ConfigST) WaitPart(uuid string, msn int, part int) chan string {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	ch := make(chan string, 1)
	if tmp, ok := element.Streams[uuid]; ok {
		tmp.hlsWaitPart[pseudoUUID()] = WaitPart{ch: ch, msn: msn, part: part}
		element.Streams[uuid] = tmp
	}
	return ch
}

//StreamHLSm3u8 get hls m3u8 list
func (element *ConfigST) StreamHLSm3u8(uuid string, lock bool) (string, int, int, error) {
	if lock {
		element.mutex.RLock()
		defer element.mutex.RUnlock()
	}
	if tmp, ok := element.Streams[uuid]; ok {
		var out string
		out += "#EXTM3U\n"
		out += "##INFO:MSN=" + strconv.Itoa(tmp.hlsCursorMSN) + ",PART=" + strconv.Itoa(tmp.hlsCursorPart+1) + "\n"
		out += "#EXT-X-TARGETDURATION:" + strconv.Itoa(int(math.Round(tmp.HlsTD))) + "\n"
		out += "#EXT-X-VERSION:7\n"
		out += "#EXT-X-INDEPENDENT-SEGMENTS\n"
		out += "#EXT-X-SERVER-CONTROL:CAN-BLOCK-RELOAD=YES,PART-HOLD-BACK=0.8000,HOLD-BACK=12.0000\n"
		out += "#EXT-X-MAP:URI=\"init.mp4\"\n"
		out += "#EXT-X-PART-INF:PART-TARGET=0.20000\n"
		out += "#EXT-X-MEDIA-SEQUENCE:" + strconv.Itoa(tmp.hlsCursorMediaSequence) + "\n"
		var keys []int
		for k := range tmp.hlsSegmentBuffer {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		CurrentPart := tmp.hlsCursorPart
		count := 0
		for _, k := range keys {
			count++
			if count >= len(keys)-1 {
				for keyFragment, valFragment := range tmp.hlsSegmentBuffer[k].fragments {
					if valFragment.Finish {
						var independ string
						if keyFragment == 0 {
							independ = ",INDEPENDENT=YES"
						}
						out += "#EXT-X-PART:DURATION=" + strconv.FormatFloat(valFragment.duration.Seconds(), 'f', 5, 64) + "" + independ + ",URI=\"segmentpart/" + strconv.Itoa(k) + "/" + strconv.Itoa(keyFragment) + "/0qrm9ru6." + strconv.Itoa(keyFragment) + ".m4s\"\n"
					} else {
						out += "#EXT-X-PRELOAD-HINT:TYPE=PART,URI=\"segmentpart/" + strconv.Itoa(k) + "/" + strconv.Itoa(keyFragment) + "/0qrm9ru6." + strconv.Itoa(keyFragment) + ".m4s\"\n"

					}
				}
			}
			if tmp.hlsSegmentBuffer[k].Finish {
				out += "#EXT-X-PROGRAM-DATE-TIME:" + tmp.hlsSegmentBuffer[k].Time + "\n#EXTINF:" + strconv.FormatFloat(tmp.hlsSegmentBuffer[k].duration.Seconds(), 'f', 5, 64) + ",\n"
				out += "segment/" + strconv.Itoa(k) + "/" + uuid + "." + strconv.Itoa(k) + ".m4s\n"
			}
		}
		return out, tmp.hlsCursorMSN, CurrentPart, nil
	}
	return "", -1, -1, ErrorStreamNotFound
}

//StreamHLSTS send hls segment buffer to clients
func (element *ConfigST) StreamHLSTS(uuid string, seq int) ([]*av.Packet, error) {
	element.mutex.RLock()
	defer element.mutex.RUnlock()
	if tmp, ok := element.Streams[uuid]; ok {
		if tmps, ok := tmp.hlsSegmentBuffer[seq]; ok {
			if tmps.fragments != nil {
				var res []*av.Packet

				keys := make([]int, len(tmps.fragments))
				i := 0
				for k := range tmps.fragments {
					keys[i] = k
					i++
				}
				sort.Ints(keys)
				for _, fragmentN := range keys {
					res = append(res, tmps.fragments[fragmentN].packets...)
				}
				return res, nil
			}

		}
	}
	return nil, ErrorStreamNotFound
}

//StreamHLSTS send hls segment buffer to clients
func (element *ConfigST) StreamHLSTSPart(uuid string, seq int, part int) ([]*av.Packet, error, bool) {
	element.mutex.RLock()
	defer element.mutex.RUnlock()
	if tmp, ok := element.Streams[uuid]; ok {
		if tmps, ok := tmp.hlsSegmentBuffer[seq]; ok {
			if tmps.fragments != nil && tmps.fragments[part].Finish {
				return tmps.fragments[part].packets, nil, true
			} else if !tmps.fragments[part].Finish {
				return nil, nil, false
			}
		}
	}
	return nil, ErrorStreamNotFound, false
}

//StreamHLSFlush delete hls cache
func (element *ConfigST) StreamHLSFlush(uuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if tmp, ok := element.Streams[uuid]; ok {
		tmp.hlsSegmentBuffer = make(map[int]Segment)
		tmp.hlsSegmentNumber = 0
		element.Streams[uuid] = tmp
	}
}

//stringToInt convert string to int if err to zero
func stringToInt(val string) int {
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return i
}
