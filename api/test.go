package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type OpRequest struct {
	Calls []CallSpec `json:"calls"`
}

type OpResult struct {
	Calls []CallResult `json:"calls"`
}

type CallSpec struct {
	URL        string            `json:"url"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers"`
	BodyString string            `json:"bodyString"`
	Body       []byte            `json:"body"`
}

type CallResult struct {
	Status   int               `json:"status"`
	Headers  map[string]string `json:"headers"`
	Response []byte            `json:"response"`
}

func Handler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.Header().Add("msg", "hello")
		w.WriteHeader(200)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Add("msg", "hello")
		w.WriteHeader(200)
		return
	}

	var req OpRequest

	err = json.Unmarshal(body, &req)
	if err != nil {
		w.Header().Add("err-msg", err.Error())
		w.WriteHeader(500)
		return
	}

	res := []CallResult{}

	for _, opr := range req.Calls {

		var reqb []byte
		if opr.BodyString != "" {
			reqb = []byte(opr.BodyString)
		} else if len(opr.Body) != 0 {
			reqb = opr.Body
		}

		hreq, err := http.NewRequest(opr.Method, opr.URL, bytes.NewReader(reqb))
		if err != nil {
			res = append(res, CallResult{
				Status:   -1,
				Response: []byte(err.Error()),
			})
			continue
		}

		for hn, hv := range opr.Headers {
			hreq.Header.Set(hn, hv)
		}

		hres, err := http.DefaultClient.Do(hreq)
		if err != nil {
			res = append(res, CallResult{
				Status:   -1,
				Response: []byte(err.Error()),
			})
			continue
		}
		hresbody, err := io.ReadAll(hres.Body)
		if err != nil {
			res = append(res, CallResult{
				Status:   -1,
				Response: []byte(err.Error()),
			})
			continue
		}

		respHeaders := map[string]string{}

		for hn, hvs := range hres.Header {
			respHeaders[hn] = strings.Join(hvs, ", ")
		}

		res = append(res, CallResult{
			Status:   hres.StatusCode,
			Headers:  respHeaders,
			Response: hresbody,
		})

	}

	objRes, err := json.Marshal(OpResult{
		Calls: res,
	})
	if err != nil {
		w.Header().Add("err-msg", err.Error())
		w.WriteHeader(500)
		return
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(200)
	_, _ = w.Write(objRes)
}
