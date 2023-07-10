package entity

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

type Entity struct {
	StartTime, EndTime time.Time
	Request            *http.Request
	Response           *http.Response
	// http.Body can only be read once, a new body needs to be copied
	ReqBody, RespBody io.ReadCloser
	Value             interface{}
}

func NewEntity(conn net.Conn) (*Entity, error) {
	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		return nil, errors.Wrap(err, "请求对象生成失败")
	}

	bodyBytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}

	request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	return &Entity{
		StartTime: time.Now(),
		Request:   request,
		ReqBody:   ioutil.NopCloser(bytes.NewBuffer(bodyBytes)),
	}, nil
}

func NewEntityWithRequest(request *http.Request) (*Entity, error) {
	bodyBytes, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}

	request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	return &Entity{
		StartTime: time.Now(),
		Request:   request,
		ReqBody:   ioutil.NopCloser(bytes.NewBuffer(bodyBytes)),
	}, nil
}

func (entity *Entity) SetResponse(response *http.Response) error {
	entity.EndTime = time.Now()
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	response.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	entity.Response = response
	entity.RespBody = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	return nil
}

func (entity *Entity) SetScheme(scheme string) {
	entity.Request.URL.Scheme = scheme
}

func (entity *Entity) SetHost(host string) {
	entity.Request.URL.Host = host
}

func (entity *Entity) SetRemoteAddr(remoteAddr string) {
	entity.Request.RemoteAddr = remoteAddr
}

func (entity *Entity) GetRequestBody() io.ReadCloser {
	return entity.ReqBody
}

func (entity *Entity) GetResponseBody() io.ReadCloser {
	return entity.RespBody
}
