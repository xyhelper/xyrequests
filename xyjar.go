package xyrequests

import (
	"net/url"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
)

type XyJar struct {
	tls_client.CookieJar
}

// NewJar 创建一个新的XyJar
func NewJar() *XyJar {
	jar := tls_client.NewCookieJar()

	return &XyJar{
		CookieJar: jar,
	}
}

// Clear 清除所有Cookie
func (j *XyJar) Clear() {
	j.CookieJar = tls_client.NewCookieJar()
}

// SetCookiesByMap 将map格式的Cookie设置到Jar中
func (j *XyJar) SetCookiesByMap(urlStr string, cookies map[string]string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	var cookieList []*fhttp.Cookie
	for name, value := range cookies {
		cookieList = append(cookieList, &fhttp.Cookie{
			Name:  name,
			Value: value,
		})
	}
	j.SetCookies(u, cookieList)
	return nil
}
