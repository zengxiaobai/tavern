package caching

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/kelindar/bitmap"

	configv1 "github.com/omalloc/tavern/api/defined/v1/middleware"
	"github.com/omalloc/tavern/api/defined/v1/storage"
	"github.com/omalloc/tavern/api/defined/v1/storage/object"
	"github.com/omalloc/tavern/contrib/log"
	"github.com/omalloc/tavern/pkg/iobuf"
	xhttp "github.com/omalloc/tavern/pkg/x/http"
	"github.com/omalloc/tavern/proxy"
	"github.com/omalloc/tavern/server/middleware"
)

const BitBlock = 1 << 15

var fileMode = os.O_RDONLY | 0o1000000 // O_NOATIME

type cachingOption struct {
	IncludeQueryInCacheKey  bool     `json:"include_query_in_cache_key" yaml:"include_query_in_cache_key"`
	FuzzyRefresh            bool     `json:"fuzzy_refresh" yaml:"fuzzy_refresh"`
	FuzzyRefreshRate        float64  `json:"fuzzy_refresh_rate" yaml:"fuzzy_refresh_rate"`
	CollapsedRequest        bool     `json:"collapsed_request" yaml:"collapsed_request"`
	CollapsedRequestTimeout string   `json:"collapsed_request_timeout" yaml:"collapsed_request_timeout"`
	VaryLimit               int      `json:"vary_limit" yaml:"vary_limit"`
	VaryIgnoreKey           []string `json:"vary_ignore_key" yaml:"vary_ignore_key"`
}

func init() {
	middleware.Register("caching", Middleware)
}

func Middleware(c *configv1.Middleware) (middleware.Middleware, func(), error) {
	opts := new(cachingOption)
	if err := c.Unmarshal(opts); err != nil {
		return nil, middleware.EmptyCleanup, err
	}

	processor := NewProcessorChain(
		NewStateProcessor(),
	).fill()

	return func(origin http.RoundTripper) http.RoundTripper {

		proxyClient := proxy.GetProxy()

		return middleware.RoundTripperFunc(func(req *http.Request) (resp *http.Response, err error) {
			// find indexdb cache-key has hit/miss.
			caching, err := processor.preCacheProcessor(proxyClient, opts, req)
			// err to BYPASS caching
			if err != nil {
				caching.log.Warnf("caching find failed: %v BYPASS", err)
				return caching.doProxy(req, false)
			}

			// cache HIT
			if caching.hit {
				rng, err1 := xhttp.SingleRange(req.Header.Get("Range"), caching.md.Size)
				if err1 != nil {
					// 无效 Range 处理
					headers := make(http.Header)
					xhttp.CopyHeader(caching.md.Headers, headers)
					headers.Set("Content-Range", fmt.Sprintf("bytes */%d", caching.md.Size))
					return nil, xhttp.NewBizError(http.StatusRequestedRangeNotSatisfiable, headers)
				}
				// TODO: mark x-cache status

				// find file seek(start, end)
				resp, err = caching.responseCache(req, rng.Start, rng.End)
				if err != nil {
					// fd leak
					if resp != nil && resp.Body != nil {
						_ = resp.Body.Close()
					}
					return nil, err
				}

				// response now
				resp, err = caching.processor.postCacheProcessor(caching, req, resp)
				return
			}

			// full MISS
			resp, err = caching.doProxy(req, false)

			resp, err = processor.postCacheProcessor(caching, req, resp)
			return
		})

	}, middleware.EmptyCleanup, nil
}

type Caching struct {
	log         *log.Helper
	processor   *ProcessorChain
	opt         *cachingOption
	req         *http.Request
	id          *object.ID
	md          *object.Metadata
	bucket      storage.Bucket
	proxyClient proxy.Proxy

	cacheable    bool
	hit          bool
	refresh      bool
	fileChanged  bool
	noContentLen bool
}

func (c *Caching) responseCache(req *http.Request, start, end int64) (*http.Response, error) {
	// find disk cache file and return Body

	// check

	return nil, nil
}

func (c *Caching) doProxy(req *http.Request, subRequest bool) (*http.Response, error) {
	proxyReq, err := c.processor.PreRequst(c, cloneRequest(req))
	if err != nil {
		return nil, fmt.Errorf("pre-request failed: %w", err)
	}

	c.log.Infof("doPorxy with %s", proxyReq.URL.String())

	resp, err := c.proxyClient.Do(proxyReq, false)
	if err != nil {
		return resp, err
	}

	c.log.Debugf("doProxy upstream resp content-length %d content-range %s etag %q lm %q",
		resp.ContentLength, resp.Header.Get("Content-Range"),
		resp.Header.Get("ETag"), resp.Header.Get("Last-Modified"))

	if log.Enabled(log.LevelDebug) {
		buf, _ := httputil.DumpResponse(resp, false)
		c.log.Debugf("doProxy resp dump: \n%s\n", string(buf))
	}

	var proxyErr error

	// handle redirect caching
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
		// origin response
		c.log.Debugf("doProxy upstream returns 301/302 url: %s location: %s",
			proxyReq.URL.String(), resp.Header.Get("Location"))
		return resp, nil
	}

	// handle Range Not Satisfiable
	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		// errors.New("upstream returns 416 Range Not Satisfiable")
		return resp, xhttp.NewBizError(resp.StatusCode, resp.Header)
	}

	// handle error response
	if resp.StatusCode >= http.StatusBadRequest {
		if c.md != nil && !c.refresh {
			proxyErr = fmt.Errorf("upstream returns error status: %d", resp.StatusCode)
		}
	}

	// code check
	notModified := resp.StatusCode == http.StatusNotModified
	statusOK := resp.StatusCode == http.StatusOK

	respRange, err := xhttp.ParseContentRange(resp.Header)
	if !notModified && !statusOK && err != nil && !errors.Is(err, xhttp.ErrContentRangeHeaderNotFound) {
		c.log.Errorf("doProxy parse upstream Content-Range header failed: %v", err)
		return resp, err
	}

	if err != nil {
		c.noContentLen = true
	}

	// parsed cache-control header
	expiredAt, cacheable := xhttp.ParseCacheTime("", resp.Header)

	now := time.Now()
	if c.md == nil {
		c.md = &object.Metadata{
			ID:          c.id,
			Headers:     make(http.Header),
			Parts:       bitmap.Bitmap{},
			Size:        respRange.ObjSize,
			Code:        http.StatusOK,
			RespUnix:    now.Unix(),
			LastRefUnix: now.Unix(),
		}
	}

	c.cacheable = cacheable
	// expire time
	c.md.ExpiresAt = now.Add(expiredAt).Unix()
	c.md.RespUnix = now.Unix()
	c.md.LastRefUnix = now.Unix()

	// file changed.
	if !notModified {

		xhttp.RemoveHopByHopHeaders(resp.Header)

		statusCode := resp.StatusCode
		if statusCode == http.StatusPartialContent {
			statusCode = http.StatusOK
		}
		c.md.Code = statusCode
		c.md.Size = respRange.ObjSize

		// error code cache feature.
		if statusCode >= http.StatusBadRequest {
			copiedHeaders := make(http.Header)
			xhttp.CopyHeader(copiedHeaders, resp.Header)
			c.md.Headers = copiedHeaders
		}

		// flushbuffer 文件从这里写出到 bucket / disk
		flushBuffer, cleanup := c.flushbuffer(respRange)

		// save body stream to bucket(disk).
		resp.Body = iobuf.SavepartReader(resp.Body, BitBlock, 0, flushBuffer, c.flushFailed, cleanup)
	}

	resp, err = c.processor.PostRequst(c, proxyReq, resp)
	if err != nil {
		return resp, err
	}

	// upgrade to chunked type
	if c.noContentLen && statusOK {
		c.md.Flags |= object.FlagChunkedCache
	}

	// update indexdb headers
	if c.fileChanged || !subRequest {
		xhttp.CopyHeader(c.md.Headers, resp.Header)
	}

	c.log.Debugf("doProxy end %s %q code: %d %s", proxyReq.Method, proxyReq.URL.String(), resp.StatusCode, respRange.String())
	return resp, proxyErr
}

// flushbuffer returns flush cache file to bucket callback
func (c *Caching) flushbuffer(respRange xhttp.ContentRange) (iobuf.EventSuccess, iobuf.EventClose) {
	// is chunked encoding
	// chunked encoding when object size unknown, waiting for Read io.EOF
	chunked := respRange.ObjSize <= 0

	// MAX_FILE_SIZE / PART_SIZE
	// PART_SIZE -> bitmap block_size

	endPart := func() uint32 {
		epart := uint32(respRange.ObjSize / BitBlock)
		if respRange.ObjSize%BitBlock > 0 {
			epart++
		}
		return epart
	}()

	getOffset := func(partIdx uint32) int64 {
		point := partIdx * BitBlock
		if partIdx == 0 {
			point = 0
		}
		return int64(point)
	}

	c.log.Debugf("flushbuffer now. chunked %t", chunked)

	wpath := c.id.WPath(c.bucket.Path())

	if err := os.MkdirAll(filepath.Dir(wpath), 0o755); err != nil {
		c.log.Debugf("mkdir fail %s", err)
	}

	w := bufio.NewWriter(io.Discard)
	f, err := os.OpenFile(wpath, os.O_CREATE|os.O_RDWR, 0o755)
	if err == nil {
		w = bufio.NewWriter(f)
	} else {
		log.Warnf("open-file failed err %s", err)
	}

	cleanup := func(eof bool) {
		_ = c.bucket.Store(c.req.Context(), c.md)
	}

	// TODO: global resource lock.
	// c.Lock()
	// defer c.Unlock()

	// write file.
	writeBuffer := func(buf []byte, bitIdx uint32, current uint64, eof bool) error {
		if chunked {
			c.md.Size = current
			c.md.Headers.Set("Content-Length", fmt.Sprintf("%d", current))
		} else if uint64(len(buf)) != BitBlock && current != respRange.ObjSize {
			c.log.Debugf("part[%d] is not complete, want end-part [%d] ", bitIdx+1, endPart)
			return nil
		}

		offset := getOffset(bitIdx)
		if offset > 0 {
			// write buf to `wpath` file at offset
			if _, err = f.Seek(offset, io.SeekStart); err != nil {
				return err
			}
		}

		if nn, err1 := w.Write(buf); err1 != nil || nn != len(buf) {
			return fmt.Errorf("writeBuffer part[%d] failed err %w", bitIdx, err)
		}
		c.md.Parts.Set(bitIdx)

		if eof {
			if endPart == uint32(c.md.Parts.Count()) {
				c.log.Debugf("file all part complete at %s", time.Now().Format(time.DateTime))
				_ = w.Flush()

				// TODO: trigger file crc check
				// ...
			}
		}

		//c.log.Debugf("flushBuffer %s, curPart: %d endPart: %d, offset %d, write bufsize %d", wpath, bitIdx, endPart, offset, nn)
		return nil
	}

	return writeBuffer, cleanup
}

// flushFailed flush cache file to bucket failed callback
func (c *Caching) flushFailed(err error) {
	c.log.Errorf("flush body to disk failed: %v", err)
	_ = c.bucket.DiscardWithMetadata(c.req.Context(), c.md)
}

func cloneRequest(req *http.Request) *http.Request {
	proxyURL, _ := url.Parse(req.URL.String())
	if proxyURL.Host == "" {
		proxyURL.Host = req.Host
	}
	if proxyURL.Scheme == "" {
		proxyURL.Scheme = xhttp.Scheme(req)
	}
	proxyReq := &http.Request{
		ProtoMajor: 1,
		ProtoMinor: 1,
		Host:       req.Host,
		Proto:      req.Proto,
		Method:     req.Method,
		URL:        proxyURL,
		Header:     make(http.Header),
	}

	return proxyReq
}

func newObjectIDFromRequest(req *http.Request, vd string, includeQuery bool) (*object.ID, error) {
	// option: cache-key include querystring
	//
	// TODO: get cache-key from frontend protocol rule.

	// or later default rule.
	if includeQuery {
		return object.NewVirtualID(req.URL.String(), vd), nil
	}

	return object.NewVirtualID(fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.Host, req.URL.Path), vd), nil
}

func ropen(wpath string) (*os.File, error) {
	return os.OpenFile(wpath, fileMode, 0o755)
}
