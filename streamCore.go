package main

import (
	"log"
	"math"
	"time"

	"github.com/deepch/vdk/format/rtspv2"
)

//serveStreams main start
func serveStreams() {
	for k, v := range Config.Streams {
		if v.OnDemand {
			log.Println(k, "OnDemand not supported")
			v.OnDemand = false
		}
		if !v.OnDemand {
			go RTSPWorkerLoop(k, v.URL)
		}
	}
}

//RTSPWorkerLoop work loop
func RTSPWorkerLoop(name, url string) {
	defer Config.RunUnlock(name)
	for {
		log.Println(name, "Stream Try Connect")
		err := RTSPWorker(name, url)
		if err != nil {
			log.Println(name, err)
		}
		//reconnect delay
		time.Sleep(1 * time.Second)
	}
}

func RTSPWorker(name, url string) error {
	FPSMode := Config.FPSMode(name)
	var start bool
	var fps int
	keyTest := time.NewTimer(20 * time.Second)
	/*
		FPS mode fixed
	*/
	if FPSMode == FPSModeFixed {
		fps = 24
	}
	RTSPClient, err := rtspv2.Dial(rtspv2.RTSPClientOptions{URL: url, DisableAudio: true, DialTimeout: 3 * time.Second, ReadWriteTimeout: 3 * time.Second, Debug: false})
	/*
		FPS mode sdp
	*/
	if FPSMode == FPSModeSDP {
		log.Println("fps sdp update new", fps)
		fps = RTSPClient.FPS
	}
	if err != nil {
		return err
	}
	defer RTSPClient.Close()
	if RTSPClient.CodecData != nil {
		Config.coAd(name, RTSPClient.CodecData)
	}
	/*
		FPS mode sps
	*/
	if FPSMode == FPSModeSPS {
		fps = updateGetFPS(fps, RTSPClient.CodecData)
	}
	var AudioOnly bool
	if len(RTSPClient.CodecData) == 1 && RTSPClient.CodecData[0].Type().IsAudio() {
		AudioOnly = true
	}
	var ProbeCount int
	var ProbeFrame int
	var ProbePTS time.Duration
	Config.NewHLSMuxer(name)
	defer Config.HLSMuxerClose(name)
	for {
		select {
		case <-keyTest.C:
			return ErrorStreamExitNoVideoOnStream
		case signals := <-RTSPClient.Signals:
			switch signals {
			case rtspv2.SignalCodecUpdate:
				Config.coAd(name, RTSPClient.CodecData)
				/*
					FPS mode sps
				*/
				if FPSMode == FPSModeSPS {
					fps = updateGetFPS(fps, RTSPClient.CodecData)
				}
			case rtspv2.SignalStreamRTPStop:
				return ErrorStreamExitRtspDisconnect
			}
		case packetAV := <-RTSPClient.OutgoingPacketQueue:
			//wait fist key on start
			if packetAV.IsKeyFrame && !start {
				start = true
			}
			/*
				FPS mode probe
			*/
			if start && FPSMode == FPSModeProbe {
				ProbePTS += packetAV.Duration
				ProbeFrame++
				if packetAV.IsKeyFrame && ProbePTS.Seconds() >= 1 {
					ProbeCount++
					if ProbeCount == 2 {
						fps = int(math.Round(float64(ProbeFrame) / ProbePTS.Seconds()))
					}
					ProbeFrame = 0
					ProbePTS = 0
				}
			}
			if AudioOnly || packetAV.IsKeyFrame {
				keyTest.Reset(20 * time.Second)
			}
			if (start && FPSMode != FPSModeProbe && fps != 0) || (ProbeCount > 2) || (start && (FPSMode == FPSModePTS || FPSMode == FPSModeFixed)) {
				if FPSMode != FPSModePTS {
					//TODO fix it
					packetAV.Duration = time.Duration((float32(1000)/float32(fps))*1000*1000) * time.Nanosecond
				}
				Config.HlsMuxerSetFPS(name, fps)
				Config.HlsMuxerWritePacket(name, packetAV)
			}
		}
	}
}
