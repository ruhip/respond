package respond

import (
	"bytes"
	"log"
	"net/http"
)

// With specifies the status code and data to repsond with.
func With(status int, data interface{}) *W {
	return &W{Code: status, Data: data}
}

// W holds details about the response that will be made
// when To is called.
type W struct {
	Code   int
	Data   interface{}
	header http.Header
}

// To writes the repsonse.
func (with *W) To(w http.ResponseWriter, r *http.Request) {
	// copy headers to ResponseWriter
	copyheaders(with.header, w.Header())
	// find the encoder
	var encoder Encoder
	var ok bool
	if encoder, ok = Encoders().Match(r.Header.Get("Accept")); !ok {
		encoder = DefaultEncoder
	}
	// get the public view (if any)
	data := public(with.Data)
	// transform the data
	transformLock.RLock()
	data = transform(w, r, data)
	transformLock.RUnlock()

	afterLock.RLock()
	res := &Response{
		w:        w,
		keepbody: keepbody,
		status:   with.Code,
		body:     new(bytes.Buffer),
		encoder:  encoder,
	}
	afterLock.RUnlock()

	// write response
	if err := Write(res, r, with.Code, data, encoder); err != nil {
		Err(w, r, with, err)
	}

	// call after (if there is one)
	if after != nil {
		afterLock.RLock()
		after(res, r, with.Code, data)
		afterLock.RUnlock()
	}
}

// Write is the function that sets the Content-Type, writes the header
// and encodes the body using the specified Encoder.
var Write = func(w http.ResponseWriter, r *http.Request, status int, data interface{}, encoder Encoder) error {
	w.Header().Set("Content-Type", encoder.ContentType(w, r))
	w.WriteHeader(status)
	return encoder.Encode(w, r, data)
}

// Err is called when an internal error occurs while responding.
var Err = func(w http.ResponseWriter, r *http.Request, with *W, err error) {
	log.Println("Err:", r.URL.String(), err)
}
