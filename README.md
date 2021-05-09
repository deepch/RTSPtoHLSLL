# RTSPtoHLSLL

### Description


RTSP Stream to WebBrowser over HLS Low Latency

full native! not use ffmpeg or gstreamer

### 1. you need https!!!
### 2. work in progress!!!
### 3. unstable!!!

### Limitations

1) This is a temporary project that's not finished yet.
2) For low latency hls to work well on IOS, you need https without it, you get regular HLS!
3) Remember this is not a production project this is an example.

Video Codecs Supported: H264 / H265 (H265 only IE or Safari)

Audio Codecs Supported: AAC
### Download Source

1. Download source (ignore error)
   ```bash 
   $ GO111MODULE=off go get github.com/deepch/RTSPtoHLSLL  
   ```

2. Enable go module
   ```bash
   $ export GO111MODULE=on
   ```
3. CD to Directory
   ```bash
    $ cd $GOPATH/src/github.com/deepch/RTSPtoHLSLL/
   ```
4. Test Run
   ```bash
    $ go run *.go
   ```
   
### Get started (configure)
Open config file and edit

1) Go to source directory 
   ```bash
   $GOPATH/src/github.com/deepch/RTSPtoHLSLL/
   ```
2) open file config.json to edit mcedit nano or other text editor
      ```bash
   mcedit config.json
   ```

1) Configure you DNS name, need domain (example.com) and external IP (white) 
   ######(if you skip this step it may work as hls without LL as it requires http 2.0)
2) Configure you dns name
   ```json
   {"server": {
      "http_server_name": "example.com",
      "http_port":        ":80",
      "https_port":       ":443"
   }}
   ```
3) If you know exactly the FPS of your stream, it is better to specify it in the config.

#### fps_mode
```bash
   fixed  - read config value fps 
   sdp    - read fps send by camera sdp
   sps    - read fps over sps vui 
   probe  - cal fps over interval (default)
   pts    - use pts unstable ;(  
```
   ####example
```json
   {"streams": {
      "H264_AAC": {
          "on_demand": false,
          "url": "rtsp://171.25.232.20/d7b92541b4914c8e98104cba907864f8",
          "fps_mode": "probe",
          "fps_probe_time": 2,
          "fps": 25
      }
   }}
   ```

## Run

1. Go to source code directory
```bash
$ cd $GOPATH/src/github.com/deepch/RTSPtoHLSLL
```
2. Run source code
```bash
$ go run .
```

## Team

Deepch - https://github.com/deepch streaming developer

Dmitry - https://github.com/vdalex25 web developer

## Worked Browser

1) Safari - Mac OS
2) Chrome - Mac OS
3) Safari - IOS

## Other Example

Examples of working with video on golang

- [RTSPtoWeb](https://github.com/deepch/RTSPtoWeb)
- [RTSPtoWebRTC](https://github.com/deepch/RTSPtoWebRTC)
- [RTSPtoWSMP4f](https://github.com/deepch/RTSPtoWSMP4f)
- [RTSPtoImage](https://github.com/deepch/RTSPtoImage)
- [RTSPtoHLS](https://github.com/deepch/RTSPtoHLS)
- [RTSPtoHLSLL](https://github.com/deepch/RTSPtoHLSLL)

[![paypal.me/AndreySemochkin](https://ionicabizau.github.io/badges/paypal.svg)](https://www.paypal.me/AndreySemochkin) - You can make one-time donations via PayPal. I'll probably buy a ~~coffee~~ tea. :tea: