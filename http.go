package main

import (
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/deepch/vdk/format/mp4f"
	"github.com/gin-gonic/gin"
)

func serveHTTP() {

	router := gin.New()
	//HLS index.m3u8 need gzip
	//router.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{".mp4", ".m4s"})))
	gin.SetMode(gin.ReleaseMode)
	router.LoadHTMLGlob("web/templates/*")
	router.GET("/", func(c *gin.Context) {
		fi, all := Config.list()
		sort.Strings(all)
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"port":     Config.Server.HTTPPort,
			"suuid":    fi,
			"suuidMap": all,
			"version":  time.Now().String(),
		})
	})
	router.GET("/player/:suuid", func(c *gin.Context) {
		_, all := Config.list()
		sort.Strings(all)
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"port":     Config.Server.HTTPPort,
			"suuid":    c.Param("suuid"),
			"suuidMap": all,
			"version":  time.Now().String(),
		})
	})
	router.GET("/play/hls/:suuid/index.m3u8", PlayHLS)
	router.GET("/play/hls/:suuid/init.mp4", AppStreamNvrHLSMP4Init)
	router.GET("/play/hls/:suuid/segment/:seq/:any", PlayHLSM4S)
	router.GET("/play/hls/:suuid/segmentpart/:seq/:part/:any", PlayHLSM4SSeq)
	router.StaticFS("/static", http.Dir("web/static"))
	//err := router.Run(Config.Server.HTTPPort)
	err := router.RunTLS(Config.Server.HTTPPort, "./testdata/server.pem", "./testdata/server.key")
	if err != nil {
		log.Fatalln(err)
	}
}
func AppStreamNvrHLSMP4Init(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
	name := c.Param("suuid")
	c.Header("Access-Control-Allow-Origin", "*")
	if !Config.ext(name) {
		return
	}
	Config.RunIFNotRun(name)
	c.Header("Content-Type", "video/mp4")
	codecs := Config.coGe(name)
	Muxer := mp4f.NewMuxer(nil)
	Muxer.WriteHeader(codecs)
	_, buf := Muxer.GetInit(codecs)
	log.Println("Write Init")
	c.Writer.Write(buf)
}

func PlayHLS(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	suuid := c.Param("suuid")
	_HLS_msn := stringToInt(c.DefaultQuery("_HLS_msn", "-1"))
	_HLS_part := stringToInt(c.DefaultQuery("_HLS_part", "-1"))
	index, _, _, _ := Config.StreamHLSm3u8(suuid, true)
	if _HLS_msn == -1 {
		c.Writer.Write([]byte(index))
		return
	}
	ch := Config.WaitPart(suuid, _HLS_msn, _HLS_part)
	clientTest := time.NewTimer(20 * time.Second)
	select {
	case <-clientTest.C:
		return
	case playlist := <-ch:
		c.Writer.Write([]byte(playlist))
	}
}

//PlayHLSTS send client ts segment
func PlayHLSM4S(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
	c.Header("Content-Type", "video/mp4")
	suuid := c.Param("suuid")
	if !Config.ext(suuid) {
		return
	}
	codecs := Config.coGe(c.Param("suuid"))
	if codecs == nil {
		return
	}
	Muxer := mp4f.NewMuxer(nil)
	//Muxer.SetIndex(stringToInt(c.Param("seq")))
	err := Muxer.WriteHeader(codecs)
	if err != nil {
		log.Println(err)
		return
	}
	seqData, err := Config.StreamHLSTS(c.Param("suuid"), stringToInt(c.Param("seq")))
	if err != nil {
		log.Println(err)
		return
	}
	if len(seqData) == 0 {
		log.Println(err)
		return
	}
	for _, v := range seqData {
		err = Muxer.WritePacket4(*v)
		if err != nil {
			log.Fatalln(err)
			return
		}
	}
	buf := Muxer.Finalize()
	_, err = c.Writer.Write(buf)
	if err != nil {
		log.Println(err)
		return
	}
}

//PlayHLSTS send client ts segment
func PlayHLSM4SSeq(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
	c.Header("Content-Type", "video/mp4")
	suuid := c.Param("suuid")
	if !Config.ext(suuid) {
		return
	}
	codecs := Config.coGe(c.Param("suuid"))
	if codecs == nil {
		return
	}
	Muxer := mp4f.NewMuxer(nil)
	//Muxer.SetIndex(stringToInt(c.Param("seq") + c.Param("part")))
	err := Muxer.WriteHeader(codecs)
	if err != nil {
		log.Println(err)
		return
	}
	seqData, err, got := Config.StreamHLSTSPart(c.Param("suuid"), stringToInt(c.Param("seq")), stringToInt(c.Param("part")))
	var found bool
	if !got && err == nil {
		//TODO: need wait create wait chan brodcast
		for i := 0; i < 100; i++ {
			seqData, err, got = Config.StreamHLSTSPart(c.Param("suuid"), stringToInt(c.Param("seq")), stringToInt(c.Param("part")))
			if err == nil && got {
				found = true
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if !found {
			log.Println("not found", c.Param("suuid"), stringToInt(c.Param("seq")), stringToInt(c.Param("part")))
			return
		}
	}
	if err != nil {
		c.Status(400)
		log.Println(err)
		return
	}
	if len(seqData) == 0 {
		log.Println(err)
		return
	}
	for _, v := range seqData {
		err = Muxer.WritePacket4(*v)
		if err != nil {
			log.Println(err)
			return
		}
	}
	buf := Muxer.Finalize()
	if err != nil {
		log.Fatalln(err)
	}
	_, err = c.Writer.Write(buf)
	if err != nil {
		log.Println(err)
		return
	}
}
