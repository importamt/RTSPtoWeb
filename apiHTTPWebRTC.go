package main

import (
	"net/http"
	"time"

	webrtc "github.com/deepch/vdk/format/webrtcv3"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

//HTTPAPIServerStreamWebRTC stream video over WebRTC
func HTTPAPIServerStreamWebRTC(c *gin.Context) {
	if !Storage.StreamChannelExist(c.Param("uuid"), c.Param("channel")) {
		c.JSON(http.StatusInternalServerError, Message{Status: 0, Payload: ErrorStreamNotFound.Error()})
		log.WithFields(logrus.Fields{
			"module":  "http_webrtc",
			"stream":  c.Param("uuid"),
			"channel": c.Param("channel"),
			"func":    "HTTPAPIServerStreamWebRTC",
			"call":    "StreamChannelExist",
		}).Errorln(ErrorStreamNotFound.Error())
		return
	}
	Storage.StreamChannelRun(c.Param("uuid"), c.Param("channel"))
	codecs, err := Storage.StreamChannelCodecs(c.Param("uuid"), c.Param("channel"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, Message{Status: 0, Payload: err.Error()})
		log.WithFields(logrus.Fields{
			"module":  "http_webrtc",
			"stream":  c.Param("uuid"),
			"channel": c.Param("channel"),
			"func":    "HTTPAPIServerStreamWebRTC",
			"call":    "StreamCodecs",
		}).Errorln(err.Error())
		return
	}
	muxerWebRTC := webrtc.NewMuxer(webrtc.Options{ICEServers: Storage.GetICEServers(), PortMin: 45000, PortMax: 60000})
	answer, err := muxerWebRTC.WriteHeader(codecs, c.PostForm("data"))
	if err != nil {
		c.JSON(http.StatusBadRequest, Message{Status: 0, Payload: err.Error()})
		log.WithFields(logrus.Fields{
			"module":  "http_webrtc",
			"stream":  c.Param("uuid"),
			"channel": c.Param("channel"),
			"func":    "HTTPAPIServerStreamWebRTC",
			"call":    "WriteHeader",
		}).Errorln(err.Error())
		return
	}
	_, err = c.Writer.Write([]byte(answer))
	if err != nil {
		c.JSON(http.StatusBadRequest, Message{Status: 0, Payload: err.Error()})
		log.WithFields(logrus.Fields{
			"module":  "http_webrtc",
			"stream":  c.Param("uuid"),
			"channel": c.Param("channel"),
			"func":    "HTTPAPIServerStreamWebRTC",
			"call":    "Write",
		}).Errorln(err.Error())
		return
	}
	go func() {
		defer func() {
			recover()
		}()

		cid, ch, _, err := Storage.ClientAdd(c.Param("uuid"), c.Param("channel"), WEBRTC)
		if err != nil {
			c.JSON(http.StatusBadRequest, Message{Status: 0, Payload: err.Error()})
			log.WithFields(logrus.Fields{
				"module":  "http_webrtc",
				"stream":  c.Param("uuid"),
				"channel": c.Param("channel"),
				"func":    "HTTPAPIServerStreamWebRTC",
				"call":    "ClientAdd",
			}).Errorln(err.Error())
			return
		}
		defer Storage.ClientDelete(c.Param("uuid"), cid, c.Param("channel"))
		var videoStart bool
		noVideo := time.NewTimer(10 * time.Second)
		for {
			select {
			case <-noVideo.C:
				c.JSON(http.StatusInternalServerError, Message{Status: 0, Payload: ErrorStreamNoVideo.Error()})
				log.WithFields(logrus.Fields{
					"module":  "http_webrtc",
					"stream":  c.Param("uuid"),
					"channel": c.Param("channel"),
					"func":    "HTTPAPIServerStreamWebRTC",
					"call":    "ErrorStreamNoVideo",
				}).Errorln(ErrorStreamNoVideo.Error())
				return
			case pck := <-ch:
				if pck.IsKeyFrame {
					noVideo.Reset(10 * time.Second)
					videoStart = true
				}
				if !videoStart {
					continue
				}
				err = muxerWebRTC.WritePacket(*pck)
				if err != nil {
					log.WithFields(logrus.Fields{
						"module":  "http_webrtc",
						"stream":  c.Param("uuid"),
						"channel": c.Param("channel"),
						"func":    "HTTPAPIServerStreamWebRTC",
						"call":    "WritePacket",
					}).Errorln(err.Error())
					return
				}
			}
		}
	}()
}
