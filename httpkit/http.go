package httpkit

import (
	"github.com/rz1226/simplegokit/blackboardkit"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

type HttpClient struct {
	client *http.Client
	bb     *blackboardkit.BlackBoradKit
}

func NewHttpClient(timeout uint, maxIdle int) *HttpClient {
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: maxIdle,
			Dial: (&net.Dialer{
				Timeout:   3 * time.Second, //建立连接的等待时间
				KeepAlive: 3000 * time.Second,
			}).Dial,
		},
		Timeout: time.Duration(timeout) * time.Second,
	}
	hc := &HttpClient{}
	hc.client = client
	hc.bb = blackboardkit.NewBlockBorad()
	hc.bb.InitLogKit("post_info", "post_error", "get_info", "get_error")
	hc.bb.SetLogReadme("post_info", "POST执行中的日志信息")
	hc.bb.SetLogReadme("post_error", "POST执行中的错误信息")
	hc.bb.SetLogReadme("get_info", "GET执行中的日志信息")
	hc.bb.SetLogReadme("get_error", "GET执行中的错误信息")

	hc.bb.InitTimerKit("default")
	hc.bb.SetTimerReadme("default", "http客户端耗时记录")
	hc.bb.SetName("http客户端httpkit")
	hc.bb.Ready()
	return hc
}

func (hc *HttpClient) Post(url string, bodyType string, body io.Reader) (string, error) {
	t := hc.bb.Start("default", "http post: "+url)
	res, err := hc.client.Post(url, bodyType, body)
	hc.bb.End(t)
	if err != nil {
		hc.bb.Log("post_error", "http post error: url="+url, "err=", err)
		return "", err
	}
	defer res.Body.Close()
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		hc.bb.Log("post_error", "http post read error: url="+url, "err=", err)
		return "", err
	}
	hc.bb.Log("post_info", "http post result:"+url, " resp: "+string(content))
	return string(content), nil
}

func (hc *HttpClient) PostForm(url string, data url.Values) (string, error) {
	t := hc.bb.Start("default", "http post: "+url)
	res, err := hc.client.PostForm(url, data)
	hc.bb.End(t)
	if err != nil {
		hc.bb.Log("post_error", "http post error: url="+url, "err=", err)
		return "", err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		hc.bb.Log("post_error", "http post read error: url="+url, "err=", err)
		return "", err
	}
	hc.bb.Log("post_info", "http post result:"+url, " resp: "+string(body))
	return string(body), nil
}

func (hc *HttpClient) Get(url string) (string, error) {
	t := hc.bb.Start("default", "http get: "+url)
	res, err := hc.client.Get(url)
	hc.bb.End(t)
	if err != nil {
		hc.bb.Log("get_error", "http get error: url="+url, "err=", err)
		return "", err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		hc.bb.Log("get_error", "http get read error: url="+url, "err=", err)
		return "", err
	}
	hc.bb.Log("get_info", "http get result:"+url, " resp: "+string(body))
	return string(body), nil
}

func (hc *HttpClient) Show() string {
	return hc.bb.Show()
}
