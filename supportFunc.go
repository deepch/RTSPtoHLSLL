package main

import (
	"errors"
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
