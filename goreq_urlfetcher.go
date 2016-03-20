package squeegee

import (
	"github.com/franela/goreq"
	"io/ioutil"
	"time"
)

// GoReqURLFetcher -
type GoReqURLFetcher struct {
}

// Fetch -
func (gr GoReqURLFetcher) Fetch(sf *Fetch) (*[]byte, error) {
	req := goreq.Request{
		Method:      "GET",
		Compression: goreq.Gzip(),
		Uri:         sf.URL,
		Timeout:     5 * time.Second,
	}
	if sf.Proxy != "" {
		req.Proxy = "http://" + sf.Proxy
	}
	if sf.Useragent != "" {
		req.UserAgent = sf.Useragent
	}
	res, err := req.Do()
	if err != nil {
		return nil, err
	}
	d, err := ioutil.ReadAll(res.Body)
	if err != nil {
		res.Body.Close()
		return nil, err
	}
	res.Body.Close()
	return &d, nil
}
