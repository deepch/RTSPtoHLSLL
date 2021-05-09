package main

import (
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/deepch/vdk/format/mp4f"
	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
)

func serveHTTP() {
	router := gin.New()
	router.Use(CORSMiddleware())
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
	router.GET("/play/hls/:uuid/index.m3u8", HttpHlsIndex)
	router.GET("/play/hls/:uuid/init.mp4", HttpHlsInit)
	router.GET("/play/hls/:uuid/segment/:segment/:any", HttpHlsSegment)
	router.GET("/play/hls/:uuid/fragment/:segment/:fragment/:any", HttpHlsFragment)
	router.StaticFS("/static", http.Dir("web/static"))
	go func() {
		err := autotls.Run(router, Config.HttpName()+Config.HttpsPort())
		if err != nil {
			log.Println("Start HTTPS Server Error", err)
		}
	}()
	err := router.Run(Config.HttpPort())
	if err != nil {
		log.Println("Start HTTP Server Error", err)
	}
}
func HttpHlsInit(c *gin.Context) {
	if !Config.ext(c.Param("uuid")) {
		log.Println("HttpHlsInit", c.Param("uuid"), ErrorStreamNotFound)
		return
	}
	Config.RunIFNotRun(c.Param("uuid"))
	c.Header("Content-Type", "video/mp4")
	codecs := Config.coGe(c.Param("uuid"))
	if codecs == nil {
		log.Println("HttpHlsInit Codec Error")
		return
	}
	Muxer := mp4f.NewMuxer(nil)
	err := Muxer.WriteHeader(codecs)
	if err != nil {
		log.Println("HttpHlsInit WriteHeader Error", err)
		return
	}
	_, buf := Muxer.GetInit(codecs)
	_, err = c.Writer.Write(buf)
	if err != nil {
		log.Println("HttpHlsInit Write Error", err)
	}
}

func HttpHlsIndex(c *gin.Context) {
	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	if !Config.ext(c.Param("uuid")) {
		log.Println("HttpHlsIndex", c.Param("uuid"), ErrorStreamNotFound)
		return
	}
	index, err := Config.HLSMuxerM3U8(c.Param("uuid"), stringToInt(c.DefaultQuery("_HLS_msn", "-1")), stringToInt(c.DefaultQuery("_HLS_part", "-1")))
	if err != nil {
		log.Println("HttpHlsIndex HLSMuxerM3U8 Error", err)
		return
	}
	_, err = c.Writer.Write([]byte(index))
	if err != nil {
		log.Println("HttpHlsIndex Write Error", err)
		return
	}
}

func HttpHlsSegment(c *gin.Context) {
	c.Header("Content-Type", "video/mp4")
	if !Config.ext(c.Param("uuid")) {
		log.Println("HttpHlsSegment", c.Param("uuid"), ErrorStreamNotFound)
		return
	}
	codecs := Config.coGe(c.Param("uuid"))
	if codecs == nil {
		log.Println("HttpHlsSegment Codec Error")
		return
	}
	Muxer := mp4f.NewMuxer(nil)
	err := Muxer.WriteHeader(codecs)
	if err != nil {
		log.Println("HttpHlsSegment WriteHeader Error", err)
		return
	}
	seqData, err := Config.HLSMuxerSegment(c.Param("uuid"), stringToInt(c.Param("segment")))
	if err != nil {
		log.Println("HttpHlsSegment HLSMuxerSegment Error", err)
		return
	}
	for _, v := range seqData {
		err = Muxer.WritePacket4(*v)
		if err != nil {
			log.Println("HttpHlsSegment WritePacket4 Error", err)
			return
		}
	}
	buf := Muxer.Finalize()
	_, err = c.Writer.Write(buf)
	if err != nil {
		log.Println("HttpHlsSegment Writer Error", err)
		return
	}
}
func HttpHlsFragment(c *gin.Context) {
	c.Header("Content-Type", "video/mp4")
	if !Config.ext(c.Param("uuid")) {
		log.Println("HttpHlsFragment", c.Param("uuid"), ErrorStreamNotFound)
		return
	}
	codecs := Config.coGe(c.Param("uuid"))
	if codecs == nil {
		log.Println("HttpHlsFragment Codec Error")
		return
	}
	Muxer := mp4f.NewMuxer(nil)
	err := Muxer.WriteHeader(codecs)
	if err != nil {
		log.Println("HttpHlsFragment WriteHeader Error", err)
		return
	}
	seqData, err := Config.HLSMuxerFragment(c.Param("uuid"), stringToInt(c.Param("segment")), stringToInt(c.Param("fragment")))
	if err != nil {
		log.Println("HttpHlsFragment HLSMuxerFragment Error", err)
		return
	}
	for _, v := range seqData {
		err = Muxer.WritePacket4(*v)
		if err != nil {
			log.Println("HttpHlsFragment WritePacket4 Error", err)
			return
		}
	}
	buf := Muxer.Finalize()
	_, err = c.Writer.Write(buf)
	if err != nil {
		log.Println("HttpHlsFragment Write Error", err)
		return
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, x-access-token")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
