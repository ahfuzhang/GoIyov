package main

import (
	"compress/gzip"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/nicecp/GoIyov"
	"github.com/nicecp/GoIyov/entity"
	"github.com/valyala/bytebufferpool"
)

var addr = flag.String("addr", ":8888", "http listen addr")
var logFile = flag.String("log_file", "/dev/stdout", "log to file")

type Handler struct {
	GoIyov.Delegate
}

func (handler *Handler) BeforeRequest(entity *entity.Entity) {
	buf := bytebufferpool.Get()
	buf.Reset()
	buf.WriteString(time.Now().Format("2006-01-02 15:04:05.999"))
	buf.WriteByte('\n')
	if entity.Request != nil {
		r := entity.Request
		buf.WriteString(r.Method)
		buf.WriteByte(' ')
		buf.WriteString(r.RequestURI)
		buf.WriteByte(' ')
		buf.WriteString(r.Proto)
		buf.WriteByte('\n')
		if h := r.Header.Get("Host"); len(h) == 0 {
			buf.WriteString("Host: ")
			buf.WriteString(r.Host)
			buf.WriteByte('\n')
		}
		formatHeader(r.Header, buf)
	}
	io.Copy(buf, entity.GetRequestBody())
	buf.WriteString("\n\n")
	entity.Value = buf
}

func formatHeader(headers http.Header, buf *bytebufferpool.ByteBuffer) {
	for k, v := range headers {
		for _, item := range v {
			buf.WriteString(k)
			buf.WriteString(": ")
			buf.WriteString(item)
			buf.WriteByte('\n')
		}
	}
	buf.WriteByte('\n')
}

func (handler *Handler) BeforeResponse(entity *entity.Entity, err error) {
	buf, ok := entity.Value.(*bytebufferpool.ByteBuffer)
	if !ok {
		panic("impossible error")
	}
	buf.WriteString(time.Now().Format("2006-01-02 15:04:05.999"))
	buf.WriteString(" latency: ")
	buf.WriteString(strconv.Itoa(int(entity.EndTime.Sub(entity.StartTime).Milliseconds())))
	buf.WriteByte('\n')
	if entity.Response != nil {
		rsp := entity.Response
		//buf.WriteString(strconv.Itoa(rsp.StatusCode))
		//buf.WriteByte(' ')
		buf.WriteString(rsp.Status)
		buf.WriteByte('\n')
		formatHeader(rsp.Header, buf)
		en, ok := rsp.Header["Content-Encoding"]
		if ok && len(en) > 0 && en[0] == "gzip" {
			gzip, _ := gzip.NewReader(entity.GetResponseBody())
			defer gzip.Close()
			io.Copy(buf, gzip)
		} else {
			io.Copy(buf, entity.GetResponseBody())
		}
	} else {
		io.Copy(buf, entity.GetResponseBody())
	}
	buf.WriteString("\n\n----------\n\n")
	handler.WriteLog(buf)
	bytebufferpool.Put(buf)
}

func (handler *Handler) ErrorLog(err error) {
	buf := bytebufferpool.Get()
	buf.Reset()
	defer bytebufferpool.Put(buf)
	buf.WriteString(time.Now().Format("2006-01-02 15:04:05.999"))
	buf.WriteByte('\n')
	buf.WriteString(err.Error())
	buf.WriteString("\n\n----------\n\n")
	handler.WriteLog(buf)
}

var logOnce sync.Once
var logFileHande *os.File

func (handler *Handler) WriteLog(buf *bytebufferpool.ByteBuffer) {
	if *logFile == "/dev/stdout" {
		os.Stdout.Write(buf.B)
		return
	} else if *logFile == "/dev/stderr" {
		os.Stderr.Write(buf.B)
		return
	}
	logFileHande.Write(buf.B)
}

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	//
	var err error
	if !(*logFile == "/dev/stdout" || *logFile == "/dev/stderr") {
		logOnce.Do(func() {
			logFileHande, err = os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
			if err != nil {
				panic("open log file error:" + err.Error())
			}
		})
		_, err = logFileHande.WriteString("start:\n")
		if err != nil {
			log.Fatalln(err)
		}
	}
	//
	GoIyov.Init()
	proxy := GoIyov.NewWithDelegate(&Handler{})
	server := &http.Server{
		Addr: *addr,
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			proxy.ServerHandler(rw, req)
		}),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
