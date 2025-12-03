package http

import (
	"net/http"
	"net/http/httptrace"
)

func ClientIP(remoteAddr string, header http.Header) string {
	addr := header.Get("Client-Ip")
	if addr == "" {
		addr = header.Get("X-Real-IP")
	}
	if addr == "" {
		addr = header.Get("X-Forwarded-For")
	}
	if addr == "" {
		return remoteAddr
	}
	return addr
}

func Scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	if scheme := r.Header.Get("X-Forwarded-Protocol"); scheme != "" {
		return scheme
	}
	if scheme := r.Header.Get("X-Url-Scheme"); scheme != "" {
		return scheme
	}
	if flag := r.Header.Get("X-Forwarded-Ssl"); flag == "on" {
		return "https"
	}
	return "http"
}

func WithTracer(req *http.Request) *http.Request {
	tracer := &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			// log.Debugf("starting to create conn %s", hostPort)
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			// log.Debugf("starting to look up dns %v", info)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			// log.Debugf("done looking up dns %v", info)
		},
		ConnectStart: func(network, addr string) {
			//  log.Debugf("starting tcp connection %v, addr %s", network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			// log.Debugf("tcp connection created network %s addr %s, err %v", network, addr, err)
		},
		GotFirstResponseByte: func() {
			// now := time.Now()
		},
		GotConn: func(info httptrace.GotConnInfo) {
			// log.Debugf("connection established. info %v", info)
		},
	}

	return req.WithContext(httptrace.WithClientTrace(req.Context(), tracer))
}
