package main

import (
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/deepch/vdk/av"
)

//MuxerHLS struct
type MuxerHLS struct {
	mutex             sync.RWMutex
	UUID              string           //Current UUID
	MSN               int              //Current MSN
	FPS               int              //Current FPS
	MediaSequence     int              //Current MediaSequence
	CurrentFragmentID int              //Current fragment id
	CacheM3U8         string           //Current index cache
	CurrentSegment    *Segment         //Current segment link
	Segments          map[int]*Segment //Current segments group
	SessionsIndex     map[string]Sei   //Session wait preload index m3u8
	SessionsPreload   map[string]Sef   //Session wait preload fragment
}

//Sei wait struct client index
type Sei struct {
	Q    chan string
	MSN  int
	Part int
}

//Sef wait struct client fragment
type Sef struct {
	Q    chan bool
	MSN  int
	Part int
}

//NewHLSMuxer Segments
func NewHLSMuxer(uuid string) *MuxerHLS {
	return &MuxerHLS{
		UUID:            uuid,
		MSN:             -1,
		SessionsIndex:   make(map[string]Sei),
		SessionsPreload: make(map[string]Sef),
		Segments:        make(map[int]*Segment),
	}
}

//SetFPS func
func (element *MuxerHLS) SetFPS(fps int) {
	element.FPS = fps
}

//WritePacket func
func (element *MuxerHLS) WritePacket(packet *av.Packet) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	//TODO delete packet.IsKeyFrame if need no EXT-X-INDEPENDENT-SEGMENTS
	if packet.IsKeyFrame && (element.CurrentSegment == nil || element.CurrentSegment.GetDuration().Seconds() >= 4) {
		if element.CurrentSegment != nil {
			element.CurrentSegment.Close()
			if len(element.Segments) > 6 {
				delete(element.Segments, element.MSN-6)
				element.MediaSequence++
			}
		}
		element.CurrentSegment = element.NewSegment()
		element.CurrentSegment.SetFPS(element.FPS)
	}
	element.CurrentSegment.WritePacket(packet)
	CurrentFragmentID := element.CurrentSegment.GetFragmentID()
	if CurrentFragmentID != element.CurrentFragmentID {
		element.UpdateIndexM3u8()
	}
	element.CurrentFragmentID = CurrentFragmentID
}

//GetIndexM3u8 func
func (element *MuxerHLS) GetIndexM3u8(needMSN int, needPart int) (string, bool, chan string, string, error) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if len(element.CacheM3U8) != 0 && ((needMSN == -1 || needPart == -1) || (needMSN < element.MSN) || (needMSN == element.MSN && needPart < element.CurrentFragmentID)) {
		return element.CacheM3U8, true, nil, "", nil
	}
	waitQ := make(chan string, 5)
	session := pseudoUUID()
	element.SessionsIndex[session] = Sei{Q: waitQ, MSN: needMSN, Part: needPart}
	return "", false, waitQ, session, nil
}

//UpdateIndexM3u8 func
func (element *MuxerHLS) UpdateIndexM3u8() {
	var header string
	var body string
	var partTarget time.Duration
	var segmentTarget time.Duration
	segmentTarget = time.Second * 2
	for _, segmentKey := range element.SortSegments(element.Segments) {
		for _, fragmentKey := range element.SortFragment(element.Segments[segmentKey].Fragment) {
			if element.Segments[segmentKey].Fragment[fragmentKey].Finish {
				var independent string
				if element.Segments[segmentKey].Fragment[fragmentKey].Independent {
					independent = ",INDEPENDENT=YES"
				}
				body += "#EXT-X-PART:DURATION=" + strconv.FormatFloat(element.Segments[segmentKey].Fragment[fragmentKey].GetDuration().Seconds(), 'f', 5, 64) + "" + independent + ",URI=\"fragment/" + strconv.Itoa(segmentKey) + "/" + strconv.Itoa(fragmentKey) + "/0qrm9ru6." + strconv.Itoa(fragmentKey) + ".m4s\"\n"
				partTarget = element.Segments[segmentKey].Fragment[fragmentKey].Duration
			} else {
				body += "#EXT-X-PRELOAD-HINT:TYPE=PART,URI=\"fragment/" + strconv.Itoa(segmentKey) + "/" + strconv.Itoa(fragmentKey) + "/0qrm9ru6." + strconv.Itoa(fragmentKey) + ".m4s\"\n"
			}
		}
		if element.Segments[segmentKey].Finish {
			segmentTarget = element.Segments[segmentKey].Duration
			body += "#EXT-X-PROGRAM-DATE-TIME:" + element.Segments[segmentKey].Time.Format("2006-01-02T15:04:05.000000Z") + "\n#EXTINF:" + strconv.FormatFloat(element.Segments[segmentKey].Duration.Seconds(), 'f', 5, 64) + ",\n"
			body += "segment/" + strconv.Itoa(segmentKey) + "/" + element.UUID + "." + strconv.Itoa(segmentKey) + ".m4s\n"
		}
	}
	header += "#EXTM3U\n"
	header += "#EXT-X-TARGETDURATION:" + strconv.Itoa(int(math.Round(segmentTarget.Seconds()))) + "\n"
	header += "#EXT-X-VERSION:7\n"
	header += "#EXT-X-INDEPENDENT-SEGMENTS\n"
	header += "#EXT-X-SERVER-CONTROL:CAN-BLOCK-RELOAD=YES,PART-HOLD-BACK=" + strconv.FormatFloat(partTarget.Seconds()*4, 'f', 5, 64) + ",HOLD-BACK=" + strconv.FormatFloat(segmentTarget.Seconds()*4, 'f', 5, 64) + "\n"
	header += "#EXT-X-MAP:URI=\"init.mp4\"\n"
	header += "#EXT-X-PART-INF:PART-TARGET=" + strconv.FormatFloat(partTarget.Seconds(), 'f', 5, 64) + "\n"
	header += "#EXT-X-MEDIA-SEQUENCE:" + strconv.Itoa(element.MediaSequence) + "\n"
	header += body
	element.CacheM3U8 = header
	element.SendIndex()
	element.SendPart()
}

//SendIndex func
func (element *MuxerHLS) SendIndex() {
	if element.MSN > 0 {
		for i, i2 := range element.SessionsIndex {
			i2.Q <- element.CacheM3U8
			delete(element.SessionsIndex, i)
		}
	}
}

//CloseSessionIndex func
func (element *MuxerHLS) CloseSessionIndex(uuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	delete(element.SessionsIndex, uuid)
}

//CloseSessionPart func
func (element *MuxerHLS) CloseSessionPart(uuid string) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	delete(element.SessionsPreload, uuid)
}

//SendPart func
func (element *MuxerHLS) SendPart() {
	for i, i2 := range element.SessionsPreload {
		i2.Q <- true
		delete(element.SessionsPreload, i)
	}
}

func (element *MuxerHLS) GetSegment(segment int) ([]*av.Packet, error) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if segmentTmp, ok := element.Segments[segment]; ok && len(segmentTmp.Fragment) > 0 {
		var res []*av.Packet
		for _, v := range element.SortFragment(segmentTmp.Fragment) {
			res = append(res, segmentTmp.Fragment[v].Packets...)
		}
		return res, nil
	}
	return nil, ErrorStreamSegmentNotFound
}

func (element *MuxerHLS) GetFragment(segment int, fragment int) ([]*av.Packet, bool, chan bool, string, error) {
	element.mutex.Lock()
	defer element.mutex.Unlock()
	if segmentTmp, segmentTmpOK := element.Segments[segment]; segmentTmpOK {
		if fragmentTmp, fragmentTmpOK := segmentTmp.Fragment[fragment]; fragmentTmpOK {
			if fragmentTmp.Finish {
				//Ret full ready fragment
				return fragmentTmp.Packets, true, nil, "", nil
			} else {
				//Make wait chan fragment
				session := pseudoUUID()
				waitQ := make(chan bool, 5)
				element.SessionsPreload[session] = Sef{Q: waitQ, MSN: segment, Part: fragment}
				return nil, false, waitQ, session, nil
			}
		}
	}
	return nil, true, nil, "", ErrorStreamFragmentNotFound
}

//SortFragment func
func (element *MuxerHLS) SortFragment(val map[int]*Fragment) []int {
	keys := make([]int, len(val))
	i := 0
	for k := range val {
		keys[i] = k
		i++
	}
	sort.Ints(keys)
	return keys
}

//SortSegments fuc
func (element *MuxerHLS) SortSegments(val map[int]*Segment) []int {
	keys := make([]int, len(val))
	i := 0
	for k := range val {
		keys[i] = k
		i++
	}
	sort.Ints(keys)
	return keys
}

func (element *MuxerHLS) Close() {

}
