# RTSPtoHLSLL

not Ready! only demo work


NOW: work only demo stream 25 fps, next version full redesign and ready soon


Work in progress m4s ready next ll

```bash
mediastreamvalidator http://127.0.0.1:8083/play/hls/H264_AAC/index.m3u8
mediastreamvalidator: mediastreamvalidator: Version 1.7.4 (496.12-201030)
[/play/hls/H264_AAC/index.m3u8] Started root playlist download
[/play/hls/H264_AAC/index.m3u8] Started media playlist download
[/play/hls/H264_AAC/index.m3u8] All media files delivered, waiting until next playlist fetch
[/play/hls/H264_AAC/index.m3u8] All media files delivered, waiting until next playlist fetch
[/play/hls/H264_AAC/index.m3u8] All media files delivered, waiting until next playlist fetch
[/play/hls/H264_AAC/index.m3u8] All media files delivered, waiting until next playlist fetch

--------------------------------------------------------------------------------
http://127.0.0.1:8083/play/hls/H264_AAC/index.m3u8
--------------------------------------------------------------------------------
HTTP Content-Type: application/vnd.apple.mpegurl

Processed 10 out of 10 segments
Average segment duration: 2.000000
Total segment bitrates (all discontinuities): average: 2747.85 kb/s, max: 2761.17 kb/s


Discontinuity: sequence: 0, parsed segment count: 10 of 10, duration: 20.000 sec, average: 2747.85 kb/s, max: 2761.17 kb/s
Track ID: 1
Video Codec: avc1
Video profile: High
Video level: 5,1
Video resolution: 3840x2160
Video average IDR interval: 1.999667, Standard deviation: 0.009274
Video frame rate: 25.497

--------------------------------------------------------------------------------
CAUTION
--------------------------------------------------------------------------------
MediaStreamValidator only checks for violations of the HLS specification. For a more
comprehensive check against the HLS Authoring Specification, please run hlsreport
on the JSON output.
```