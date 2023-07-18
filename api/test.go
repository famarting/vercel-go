package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type CloudEventRequest struct {
	Data OpRequest `json:"data"`
}

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
		fmt.Println("error reading request body " + err.Error())
		w.Header().Add("msg", "hello")
		w.WriteHeader(200)
		return
	}
	fmt.Println("received request " + string(body))

	var req OpRequest

	err = JSONStrictUnmarshal(body, &req)
	if err != nil {
		var ceReq CloudEventRequest
		ceerr := json.Unmarshal(body, &ceReq)
		if ceerr != nil {
			w.Header().Add("err-msg", err.Error())
			w.WriteHeader(500)
			return
		}
		req = ceReq.Data
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

func JSONStrictUnmarshal(b []byte, t interface{}) error {
	reader := bytes.NewReader(b)
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	return decoder.Decode(t)
}
