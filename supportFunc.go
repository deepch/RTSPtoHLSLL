package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/deepch/vdk/av"
	"github.com/deepch/vdk/codec/h264parser"
)

var (
	ErrorStreamNotFound            = errors.New("Stream Not Found")
	ErrorStreamExitNoVideoOnStream = errors.New("Stream Exit No Video On Stream")
	ErrorStreamExitRtspDisconnect  = errors.New("Stream Exit Rtsp Disconnect")
	ErrorStreamIndexTimeout        = errors.New("Stream Index Timeout")
	ErrorStreamSegmentNotFound     = errors.New("Stream Segment Not Found")
	ErrorStreamFragmentNotFound    = errors.New("Stream Fragment Not Found")
	ErrorStreamFragmentTimeout     = errors.New("Stream Fragment Timeout")
)

//pseudoUUID func generate random uuid
func pseudoUUID() (uuid string) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return
}

//stringToInt convert string to int if err to zero
func stringToInt(val string) int {
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return i
}

//UpdateGetFPS func
func updateGetFPS(curFPS int, val []av.CodecData) int {
	log.Println(h264parser.NALU_SPS)
	for _, data := range val {
		if data.Type().IsVideo() && data.Type() == av.H264 {
			newFPS := int(data.(h264parser.CodecData).SPSInfo.FPS)
			if curFPS != newFPS {
				log.Println("fps sps update", curFPS, "new", newFPS)
				curFPS = newFPS
			}
		}
	}
	return curFPS
}
