package httpbin

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nexa/pkg/httpbin/digest"
	"github.com/nexa/pkg/httpbin/websocket"
)

var nilValues = url.Values{}

func parseFiles(fileHeaders map[string][]*multipart.FileHeader) (map[string][]string, error) {
	files := map[string][]string{}
	for k, fs := range fileHeaders {
		files[k] = []string{}
		for _, f := range fs {
			fh, err := f.Open()
			if err != nil {
				return nil, err
			}
			contents, err := io.ReadAll(fh)
			if err != nil {
				return nil, err
			}
			files[k] = append(files[k], string(contents))
		}
	}
	return files, nil
}

func encodeData(body []byte, contentType string) string {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	data := base64.URLEncoding.EncodeToString(body)
	return fmt.Sprintf("data:%s;base64,%s", contentType, data)
}

// parseBody 是 parseBody
func parseBody(c *gin.Context, resp *bodyResponse) error {
	// 读取原始请求体
	body, err := c.GetRawData()
	if err != nil {
		return err
	}

	// 如果请求体为空，直接返回
	if len(body) == 0 {
		return nil
	}

	// 始终存储原始请求体
	resp.Data = string(body)

	// 获取 Content-Type 并去除参数部分（如 charset）
	contentType := c.ContentType()
	contentType, _, _ = strings.Cut(contentType, ";")

	switch contentType {
	case "text/html", "text/plain":
		// 不需要额外解析，字符串已设置
		return nil

	case "application/x-www-form-urlencoded":
		// 兼容 DELETE/GET 请求，Gin 默认不支持这些方法的 PostForm 解析
		if c.Request.Method == http.MethodDelete || c.Request.Method == http.MethodGet {
			originalMethod := c.Request.Method
			c.Request.Method = http.MethodPost
			defer func() { c.Request.Method = originalMethod }()
		}
		if err := c.Request.ParseForm(); err != nil {
			return err
		}
		resp.Form = c.Request.PostForm

	case "multipart/form-data":
		// 解析 multipart 表单数据
		if err := c.Request.ParseMultipartForm(1024); err != nil {
			return err
		}
		resp.Form = c.Request.PostForm
		files, err := parseFiles(c.Request.MultipartForm.File)
		if err != nil {
			return err
		}
		resp.Files = files

	case "application/json":
		// 绑定 JSON 到结构体
		if err := c.ShouldBindJSON(&resp.JSON); err != nil {
			return err
		}

	default:
		// 未知 Content-Type，编码为 base64 数据 URL
		resp.Data = encodeData(body, contentType)
	}

	return nil
}

func (h *HttpBin) RequestWithBody(c *gin.Context) {
	resp := &bodyResponse{
		Args:    c.Request.URL.Query(),
		Files:   nilValues,
		Form:    nilValues,
		Headers: c.Request.Header,
		Method:  c.Request.Method,
		Origin:  c.ClientIP(),
		URL:     c.Request.URL.String(),
	}

	if err := parseBody(c, resp); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	writeJSON(c, http.StatusOK, resp)
}

func (h *HttpBin) Index(c *gin.Context) {
	c.Header("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' camo.githubusercontent.com")
	writeJSON(c, http.StatusOK, map[string]string{"message": "Welcome to httpbin"})
}

func (h *HttpBin) Get(c *gin.Context) {
	resp := noBodyResponse{
		Args:    c.Request.URL.Query(),
		Headers: c.Request.Header,
		Method:  c.Request.Method,
		Origin:  c.ClientIP(),
		URL:     c.Request.URL.String(),
	}

	writeJSON(c, http.StatusOK, resp)
}

func (h *HttpBin) WebsocketEcho(c *gin.Context) {
	var (
		maxFragmentSize = h.MaxBodySize / 2
		maxMessageSize  = h.MaxBodySize
		q               = c.Request.URL.Query()
		err             error
	)

	if userMaxFragmentSize := q.Get("max_fragment_size"); userMaxFragmentSize != "" {
		maxFragmentSize, err = strconv.ParseInt(userMaxFragmentSize, 10, 32)
		if err != nil {
			writeError(c, http.StatusBadRequest, fmt.Errorf("invalid max_fragment_size: %w", err))
			return
		} else if maxFragmentSize < 1 || maxFragmentSize > h.MaxBodySize {
			writeError(c, http.StatusBadRequest, fmt.Errorf("invalid max_fragment_size: %d not in range [1, %d]", maxFragmentSize, h.MaxBodySize))
			return
		}
	}

	if userMaxMessageSize := q.Get("max_message_size"); userMaxMessageSize != "" {
		maxMessageSize, err = strconv.ParseInt(userMaxMessageSize, 10, 32)
		if err != nil {
			writeError(c, http.StatusBadRequest, fmt.Errorf("invalid max_message_size: %w", err))
			return
		} else if maxMessageSize < 1 || maxMessageSize > h.MaxBodySize {
			writeError(c, http.StatusBadRequest, fmt.Errorf("invalid max_message_size: %d not in range [1, %d]", maxMessageSize, h.MaxBodySize))
			return
		}
	}

	if maxFragmentSize > maxMessageSize {
		writeError(c, http.StatusBadRequest, fmt.Errorf("max_fragment_size %d must be less than or equal to max_message_size %d", maxFragmentSize, maxMessageSize))
		return
	}

	ws := websocket.New(c.Writer, c.Request, websocket.Limits{
		MaxDuration:     h.MaxDuration,
		MaxFragmentSize: int(maxFragmentSize),
		MaxMessageSize:  int(maxMessageSize),
	})
	if err := ws.Handshake(); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	ws.Serve(websocket.EchoHandler)
}

func (h *HttpBin) redirectLocation(c *gin.Context, relative bool, n int) string {
	var location string
	var path string

	if n < 1 {
		path = "/get"
	} else if relative {
		path = fmt.Sprintf("/relative-redirect/%d", n)
	} else {
		path = fmt.Sprintf("/absolute-redirect/%d", n)
	}

	if relative {
		location = path
	} else {
		u := *c.Request.URL
		u.Path = path
		u.RawQuery = ""
		location = u.String()
	}

	return location
}

func (h *HttpBin) handleRedirect(c *gin.Context, relative bool) {
	n, err := strconv.Atoi(c.Param("numRedirects"))
	if err != nil {
		writeError(c, http.StatusBadRequest, fmt.Errorf("invalid number of redirects: %w", err))
		return
	} else if n < 1 {
		writeError(c, http.StatusBadRequest, fmt.Errorf("number of redirects must be greater than 0"))
		return
	}
	// 重定向到 /cookies 路径
	c.Redirect(http.StatusFound, h.redirectLocation(c, relative, n-1))
}

func (h *HttpBin) AbsoluteRedirect(c *gin.Context) {
	h.handleRedirect(c, false)
}

func (h *HttpBin) AnyThing(c *gin.Context) {
	// Short-circuit for HEAD requests, which should be handled like regular
	// GET requests (where the autohead middleware will take care of discarding
	// the body)
	if http.MethodHead == c.Request.Method {
		h.Get(c)
		return
	}

	h.RequestWithBody(c)
}

func (h *HttpBin) Base64(c *gin.Context) {
	result, err := newBase64Helper(c, h.MaxBodySize).transform()
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	ct := c.Query("Content-type")
	if ct == "" {
		ct = textContentType
	}

	// prevent XSS and other client side vulns if the content type is dangerous
	if h.mustEscapeResponse(ct) {
		result = []byte(html.EscapeString(string(result)))
	}
	writeResponse(c, http.StatusOK, ct, result)
}

// mustEscapeResponse returns true if the response body should be HTML-escaped
// to prevent XSS and similar attacks when rendered by a web browser.
func (h *HttpBin) mustEscapeResponse(contentType string) bool {
	if h.unsafeAllowDangerousResponses {
		return false
	}
	return isDangerousContentType(contentType)
}

func (h *HttpBin) BasicAuth(c *gin.Context) {
	expectedUser := c.Param("user")
	expectedPass := c.Param("password")

	givenUser, givenPass, _ := c.Request.BasicAuth()

	status := http.StatusOK
	authorized := givenUser == expectedUser && givenPass == expectedPass
	if !authorized {
		status = http.StatusUnauthorized
		c.Header("WWW-Authenticate", `Basic realm="Fake Realm"`)
	}

	writeJSON(c, status, authResponse{
		Authenticated: authorized,
		Authorized:    authorized,
		User:          givenUser,
	})
}

func (h *HttpBin) Bearer(c *gin.Context) {
	reqToken := c.GetHeader("Authorization")
	tokenFileds := strings.Fields(reqToken)

	if len(tokenFileds) != 2 || tokenFileds[0] != "Bearer" {
		c.Header("WWW-Authenticate", "Bearer realm=\"Fake Realm\"")
		writeJSON(c, http.StatusUnauthorized, bearerResponse{
			Authenticated: false,
			Token:         "",
		})
		return
	}
	writeJSON(c, http.StatusOK, bearerResponse{
		Authenticated: true,
		Token:         tokenFileds[1],
	})
}

func (h *HttpBin) HandleBytes(c *gin.Context) {
	// 获取路径参数 numBytes
	numBytesStr := c.Param("numBytes")
	numBytes, err := strconv.Atoi(numBytesStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid byte count: %v", err)})
		return
	}

	// 解析种子参数 seed
	seedStr := c.Query("seed")
	rng, err := parseSeed(seedStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid seed: %v", err)})
		return
	}

	// 验证字节长度是否合法
	if numBytes < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid byte count: %d must be greater than 0", numBytes)})
		return
	}

	// 特殊情况：0 字节直接返回
	if numBytes == 0 {
		c.Header("Content-Length", "0")
		c.Status(http.StatusOK)
		return
	}

	// 检查字节长度是否超出最大限制
	if numBytes > int(h.MaxBodySize) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid byte count: %d not in range [1, %d]", numBytes, h.MaxBodySize)})
		return
	}

	// 初始化 chunkSize 和写入函数
	var chunkSize int
	var write func([]byte)

	streaming := c.Query("streaming") == "true"
	if streaming {
		// 如果是流式传输，则解析 chunk_size 参数
		chunkSizeStr := c.Query("chunk_size")
		if chunkSizeStr != "" {
			chunkSize, err = strconv.Atoi(chunkSizeStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid chunk_size: %v", err)})
				return
			}
		} else {
			chunkSize = 10 * 1024 // 默认块大小为 10KB
		}

		// 定义流式写入函数
		write = func(chunk []byte) {
			c.Data(http.StatusOK, binaryContentType, chunk)
			c.Writer.Flush() // 强制刷新缓冲区
		}
	} else {
		// 非流式传输时设置 Content-Length 并一次性写入
		chunkSize = numBytes
		c.Header("Content-Length", strconv.Itoa(numBytes))
		write = func(chunk []byte) {
			c.Data(http.StatusOK, binaryContentType, chunk)
		}
	}

	// 设置响应头并开始写入数据
	c.Header("Content-Type", binaryContentType)
	c.Status(http.StatusOK)

	var chunk []byte
	for i := 0; i < numBytes; i++ {
		// 生成随机字节
		chunk = append(chunk, byte(rng.Intn(256)))
		if len(chunk) == chunkSize {
			write(chunk)
			chunk = nil
		}
	}
	// 写入剩余的数据
	if len(chunk) > 0 {
		write(chunk)
	}
}

// Bytes returns N random bytes generated with an optional seed
func (h *HttpBin) Bytes(c *gin.Context) {
	h.HandleBytes(c)
}

// Cache returns a 304 if an If-Modified-Since or an If-None-Match header is
// present, otherwise returns the same response as Get.
func (h *HttpBin) Cache(c *gin.Context) {
	if c.GetHeader("If-Modified-Since") != "" || c.GetHeader("If-None-Match") != "" {
		c.Status(http.StatusNotModified)
		return
	}

	lastModified := time.Now().UTC().Format(http.TimeFormat)
	c.Header("Last-Modified", lastModified)
	c.Header("ETag", sha1hash(lastModified))

	h.Get(c)
}

func (h *HttpBin) CacheControl(c *gin.Context) {
	seconds, err := strconv.ParseInt(c.Param("numSeconds"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	c.Header("Cache-Control", fmt.Sprintf("max-age=%d", seconds))

	h.Get(c)
}

func (h *HttpBin) Cookies(c *gin.Context) {
	resp := cookiesResponse{Cookies: make(map[string]string)}
	for _, cookie := range c.Request.Cookies() {
		resp.Cookies[cookie.Name] = cookie.Value
	}
	writeJSON(c, http.StatusOK, resp)
}

func (h *HttpBin) DeleteCookies(c *gin.Context) {
	params := c.Request.URL.Query()
	for k := range params {
		c.SetCookie(k, params.Get(k), -1, "/", "", false, true)
	}
	c.Redirect(http.StatusFound, "/cookies")
}

// SetCookies sets cookies as specified in query params and redirects to
// Cookies endpoint using Gin framework
func (h *HttpBin) SetCookies(c *gin.Context) {
	// 获取查询参数
	params := c.Request.URL.Query()

	// 遍历查询参数并设置 Cookie
	for k := range params {
		c.SetCookie(k, params.Get(k), 0, "/", "", false, true)
	}

	// 重定向到 /cookies 路径
	c.Redirect(http.StatusFound, "/cookies")
}

// Deflate returns a deflated response using Gin framework
func (h *HttpBin) Deflate(c *gin.Context) {
	var (
		buf bytes.Buffer
		zw  = zlib.NewWriter(&buf)
	)

	// 构造响应数据并进行 deflate 压缩
	mustMarshalJSON(zw, &noBodyResponse{
		Args:     c.Request.URL.Query(),
		Headers:  getRequestHeaders(c, h.excludeHeadersProcessor),
		Method:   c.Request.Method,
		Origin:   c.ClientIP(),
		Deflated: true,
	})
	zw.Close()

	body := buf.Bytes()

	// 设置响应头
	c.Header("Content-Encoding", "deflate")
	c.Header("Content-Type", jsonContentType)
	c.Header("Content-Length", strconv.Itoa(len(body)))

	// 返回压缩后的数据
	c.Data(http.StatusOK, jsonContentType, body)
}

// Delay waits for a given amount of time before responding, where the time may
// be specified as a golang-style duration or seconds in floating point.
func (h *HttpBin) Delay(c *gin.Context) {
	// 获取路径参数 duration
	durationStr := c.Param("duration")
	delay, err := parseBoundedDuration(durationStr, 0, h.MaxDuration)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid duration: %v", err),
		})
		return
	}

	// 等待指定时间或监听客户端断开
	select {
	case <-c.Request.Context().Done():
		c.Status(499) // "Client Closed Request" https://httpstatuses.com/499
		return
	case <-time.After(delay):
	}

	// 设置 Server-Timing 响应头
	c.Header("Server-Timing", encodeServerTimings([]serverTiming{
		{"initial_delay", delay, "initial delay"},
	}))

	// 调用 RequestWithBody 处理后续逻辑
	h.RequestWithBody(c)
}

func (h *HttpBin) Deny(c *gin.Context) {
	writeResponse(c, http.StatusOK, textContentType, []byte(`YOU SHOULDN'T BE HERE`))
}

// DigestAuth handles a simple implementation of HTTP Digest Authentication,
// which supports the "auth" QOP and the MD5 and SHA-256 crypto algorithms.
//
// /digest-auth/<qop>/<user>/<password>
// /digest-auth/<qop>/<user>/<password>/<algorithm>
func (h *HttpBin) DigestAuth(c *gin.Context) {
	var (
		qop      = strings.ToLower(c.Param("qop"))
		user     = c.Param("user")
		password = c.Param("password")
		algoName = strings.ToUpper(c.Param("algorithm"))
	)
	if algoName == "" {
		algoName = "MD5"
	}

	if qop != "" && qop != "auth" {
		writeError(c, http.StatusBadRequest, fmt.Errorf("invalid qop: %s", qop))
		return
	}

	if algoName != "MD5" && algoName != "SHA-256" {
		writeError(c, http.StatusBadRequest, fmt.Errorf("invalid algorithm: %s", algoName))
		return
	}

	algorithm := digest.MD5
	if algoName == "SHA-256" {
		algorithm = digest.SHA256
	}

	if !digest.Check(c.Request, user, password) {
		c.Header("WWW-Authenticate", digest.Challenge("go-httpbin", algorithm))
		writeError(c, http.StatusUnauthorized, fmt.Errorf("invalid credentials"))
		return
	}

	writeJSON(c, http.StatusOK, authResponse{
		Authenticated: true,
		Authorized:    true,
		User:          user,
	})
}

// Drip simulates a slow HTTP server by writing data over a given duration
// after an optional initial delay.
//
// Because this endpoint is intended to simulate a slow HTTP connection, it
// intentionally does NOT use chunked transfer encoding even though its
// implementation writes the response incrementally.
//
// See Stream (/stream) or StreamBytes (/stream-bytes) for endpoints that
// respond using chunked transfer encoding.
func (h *HttpBin) Drip(c *gin.Context) {
	var (
		duration = h.DefaultParams.DripDuration
		delay    = h.DefaultParams.DripDelay
		numBytes = h.DefaultParams.DripNumBytes
		code     = http.StatusOK

		err error
	)

	if userDuration := c.Query("duration"); userDuration != "" {
		duration, err = parseBoundedDuration(userDuration, 0, h.MaxDuration)
		if err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}
	}

	if userDelay := c.Query("delay"); userDelay != "" {
		delay, err = parseBoundedDuration(userDelay, 0, h.MaxDuration)
		if err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}
	}

	if userNumBytes := c.Query("numbytes"); userNumBytes != "" {
		numBytes, err = strconv.ParseInt(userNumBytes, 10, 64)
		if err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		} else if numBytes < 1 || numBytes > h.MaxBodySize {
			writeError(c, http.StatusBadRequest, fmt.Errorf("numbytes must be between 1 and %d", h.MaxBodySize))
			return
		}
	}

	if userCode := c.Query("code"); userCode != "" {
		code, err = parseStatusCode(userCode)
		if err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}
	}

	if duration+delay > h.MaxDuration {
		writeError(c, http.StatusBadRequest, fmt.Errorf("duration + delay must be less than %s", h.MaxDuration))
		return
	}

	pause := duration
	if numBytes > 1 {
		pause = duration / time.Duration(numBytes-1)
	}
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-c.Request.Context().Done():
			c.Status(499) // "Client Closed Request" https://httpstatuses.com/499
			return
		}
	}

	c.Header("Content-Type", textContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", numBytes))
	c.Header("Server-Timing", encodeServerTimings([]serverTiming{
		{"total_duration", delay + duration, "total request duration"},
		{"initial_delay", delay, "initial delay"},
		{"write_duration", duration, "duration of writes after initial delay"},
		{"pause_per_write", pause, "computed pause between writes"},
	}))
	c.Status(code)

	b := []byte{'*'}

	if pause == 0 {
		c.Data(http.StatusOK, textContentType, bytes.Repeat(b, int(numBytes)))
		return
	}

	ticker := time.NewTicker(pause)
	defer ticker.Stop()

	for i := int64(0); i < numBytes; i++ {
		c.Writer.Write(b)
		if i == numBytes-1 {
			return
		}

		select {
		case <-ticker.C:
		case <-c.Request.Context().Done():
			return
		}
	}
}

// DumpRequest - returns the given request in its HTTP/1.x wire representation.
// The returned representation is an approximation only;
// some details of the initial request are lost while parsing it into
// an http.Request. In particular, the order and case of header field
// names are lost.
func (h *HttpBin) DumpRequest(c *gin.Context) {
	dump, err := httputil.DumpRequest(c.Request, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":       "failed to dump request",
			"detail":      err.Error(),
			"status_code": http.StatusInternalServerError,
		})
		return
	}
	c.Data(http.StatusOK, "text/plain", dump)
}

// Env - returns environment variables with HTTPBIN_ prefix, if any pre-configured by operator
func (h *HttpBin) Env(c *gin.Context) {
	envroinment := os.Environ()
	for _, env := range envroinment {
		b, a, found := strings.Cut(env, "=")
		if !found {
			continue
		}

		h.env[b] = a
	}

	writeJSON(c, http.StatusOK, &envResponse{
		Env: h.env,
	})
}

// ETag assumes the resource has the given etag and responds to If-None-Match
// and If-Match headers appropriately.
func (h *HttpBin) ETag(c *gin.Context) {
	etag := c.Param("etag")
	c.Header("ETag", etag)
	c.Header("Cache-Control", textContentType)
	var buf bytes.Buffer
	mustMarshalJSON(&buf, noBodyResponse{
		Args:    c.Request.URL.Query(),
		Headers: getRequestHeaders(c, h.excludeHeadersProcessor),
		Method:  c.Request.Method,
		Origin:  c.ClientIP(),
		URL:     getURL(c.Request).String(),
	})

	if noneMatch := c.Request.Header.Get("If-None-Match"); noneMatch != "" {
		if noneMatch == etag || noneMatch == fmt.Sprintf(`"%s"`, etag) {
			c.Status(http.StatusNotModified)
			return
		}
	}

	if match := c.Request.Header.Get("If-Match"); match != "" {
		if match != etag && match != fmt.Sprintf(`"%s"`, etag) {
			c.Status(http.StatusPreconditionFailed)
			return
		}
	}

	c.DataFromReader(http.StatusOK, int64(buf.Len()), "application/json", &buf, nil)
}

// Gzip returns a gzipped response
func (h *HttpBin) Gzip(c *gin.Context) {
	var (
		buf bytes.Buffer
		gzw = gzip.NewWriter(&buf)
	)
	mustMarshalJSON(gzw, &noBodyResponse{
		Args:    c.Request.URL.Query(),
		Headers: getRequestHeaders(c, h.excludeHeadersProcessor),
		Method:  c.Request.Method,
		Origin:  c.ClientIP(),
		Gzipped: true,
	})
	gzw.Close()

	body := buf.Bytes()
	c.Header("Content-Encoding", "gzip")
	c.Header("Content-Length", strconv.Itoa(len(body)))
	c.Data(http.StatusOK, jsonContentType, body)
}

func (h *HttpBin) Headers(c *gin.Context) {
	writeJSON(c, http.StatusOK, &noBodyResponse{
		Args:    c.Request.URL.Query(),
		Headers: getRequestHeaders(c, h.excludeHeadersProcessor),
		Method:  c.Request.Method,
		Origin:  c.ClientIP(),
		URL:     getURL(c.Request).String(),
	})
}

func (h *HttpBin) HiddenBasicAuth(c *gin.Context) {
	expectedUser := c.Param("user")
	expectedPass := c.Param("password")

	givenUser, givenPass, ok := c.Request.BasicAuth()
	if !ok || givenUser != expectedUser || givenPass != expectedPass {
		writeError(c, http.StatusUnauthorized, fmt.Errorf("invalid credentials"))
		return
	}

	authorized := givenUser == expectedUser && givenPass == expectedPass
	if authorized {
		writeJSON(c, http.StatusOK, &authResponse{
			Authenticated: authorized,
			Authorized:    authorized,
			User:          givenUser,
		})
	} else {
		writeError(c, http.StatusUnauthorized, fmt.Errorf("invalid credentials"))
	}
}

// HostName - returns the hostname.
func (h *HttpBin) HostName(c *gin.Context) {
	writeJSON(c, http.StatusOK, &hostnameResponse{
		Hostname: h.hostname,
	})
}

// Html renders a basic HTML page
func (h *HttpBin) Html(c *gin.Context) {
	writeHTML(c, http.StatusOK, mustStaticAsset("moby.html"))
}

func (h *HttpBin) Image(c *gin.Context) {
	doImage(c, c.Param("kind"))
}

// doImage responds with a specific kind of image, if there is an image asset
// of the given kind.
func doImage(c *gin.Context, kind string) {
	img, err := staticAsset(fmt.Sprintf("image.%s", kind))
	if err != nil {
		writeError(c, http.StatusNotFound, err)
		return
	}
	contentType := fmt.Sprintf("image/%s", kind)
	if kind == "svg" {
		contentType = "image/svg+xml"
	}

	writeResponse(c, http.StatusOK, contentType, img)
}

// ImageAccept responds with an appropriate image based on the Accept header
func (h *HttpBin) ImageAccept(c *gin.Context) {
	accept := c.GetHeader("Accept")
	switch accept {

	}
}

// IP echoes the IP address of the incoming request
func (h *HttpBin) IP(c *gin.Context) {
	writeJSON(c, http.StatusOK, &ipResponse{
		Origin: c.ClientIP(),
	})
}

func (h *HttpBin) JSON(c *gin.Context) {
	writeJSON(c, http.StatusOK, &bodyResponse{
		Args:    c.Request.URL.Query(),
		Headers: getRequestHeaders(c, h.excludeHeadersProcessor),
		Method:  c.Request.Method,
		Origin:  c.ClientIP(),
		URL:     getURL(c.Request).String(),
		JSON:    mustStaticAsset("sample.json"),
	})
}

// Links redirects to the first page in a series of N links
func (h *HttpBin) Links(c *gin.Context) {
	n, er := strconv.Atoi(c.Param("numLinks"))
	if er != nil {
		writeError(c, http.StatusBadRequest, er)
		return
	} else if n < 0 || n > 256 {
		writeError(c, http.StatusBadRequest, fmt.Errorf("numLinks must be between 0 and 256"))
		return
	}

	// Are we handling /links/<n>/<offset>? If so, render an HTML page
	if rawOffset := c.Param("offset"); rawOffset != "" {
		offset, er := strconv.Atoi(rawOffset)
		if er != nil {
			writeError(c, http.StatusBadRequest, er)
			return
		}

		h.doLinksPage(c, n, offset)
	}

	// Otherwise, redirect from /links/<n> to /links/<n>/0
	c.Request.URL.Path = c.Request.URL.Path + "/0"
	h.doRedirectGin(c, c.Request.URL.String(), http.StatusFound)
}

// doLinksPage renders a page with a series of N links
func (h *HttpBin) doLinksPage(c *gin.Context, n int, offset int) {
	c.Header("Content-Type", htmlContentType)
	c.Status(http.StatusOK)

	_, err := c.Writer.Write([]byte("<html><head><title>Links</title></head><body>"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	for i := range n {
		if i == offset {
			fmt.Fprintf(c.Writer, "%d ", i)
		} else {
			fmt.Fprintf(c.Writer, `<a href="%s/links/%d/%d">%d</a> `, h.prefix, n, i, i)
		}
	}
	c.Writer.Write([]byte("</body></html>"))
}

// doRedirect set redirect header using Gin
func (h *HttpBin) doRedirectGin(c *gin.Context, path string, code int) {
	var sb strings.Builder
	if strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") {
		sb.WriteString(h.prefix)
	}
	sb.WriteString(path)
	c.Redirect(code, sb.String())
}

// Range returns up to N bytes, with support for HTTP Range requests.
//
// This departs from original httpbin in a few ways:
//
//   - param `chunk_size` IS NOT supported
//
//   - param `duration` IS supported, but functions more as a delay before the
//     whole response is written
//
//   - multiple ranges ARE correctly supported (i.e. `Range: bytes=0-1,2-3`
//     will return a multipart/byteranges response)
//
// Most of the heavy lifting is done by the stdlib's http.ServeContent, which
// handles range requests automatically. Supporting chunk sizes would require
// an extensive reimplementation, especially to support multiple ranges for
// correctness. For now, we choose not to take that work on.
func (h *HttpBin) Range(c *gin.Context) {
	numBytes, err := strconv.ParseInt(c.Param("numBytes"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	var duration time.Duration
	if durationVal := c.Query("duration"); durationVal != "" {
		duration, err = parseBoundedDuration(durationVal, 0, h.MaxDuration)
		if err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}
	}

	c.Header("ETag", fmt.Sprintf("range%d", numBytes))
	c.Header("Accept-Ranges", "bytes")

	if numBytes <= 0 || numBytes > h.MaxBodySize {
		writeError(c, http.StatusBadRequest, fmt.Errorf("numBytes must be between 1 and %d", h.MaxBodySize))
		return
	}

	content := newSyntheticByteStream(numBytes, duration, func(offset int64) byte {
		return byte(97 + (offset % 26))
	})

	var modtime time.Time
	http.ServeContent(c.Writer, c.Request, "", modtime, content)
}

// RedirectTo responds with a redirect to a specific URL with an optional
// status code, which defaults to 302
func (h *HttpBin) RedirectTo(c *gin.Context) {
	inputUrl := c.Query("url")
	if inputUrl == "" {
		writeError(c, http.StatusBadRequest, fmt.Errorf("url parameter is required"))
		return
	}

	u, err := url.Parse(inputUrl)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}

	// If we're given a URL that includes a domain name and we have a list of
	// allowed domains, ensure that the domain is allowed.
	//
	// Note: This checks the hostname directly rather than using the net.URL's
	// IsAbs() method, because IsAbs() will return false for URLs that omit
	// the scheme but include a domain name, like "//evil.com" and it's
	// important that we validate the domain in these cases as well.
	if u.Hostname() != "" && len(h.AllowedRedirectDomains) > 0 {
		if _, ok := h.AllowedRedirectDomains[u.Hostname()]; !ok {
			writeError(c, http.StatusForbidden, fmt.Errorf("redirect URL is not allowed"))
			return
		}
	}

	statusCode := http.StatusFound
	if rawStatusCode := c.Query("status_code"); rawStatusCode != "" {
		statusCode, err = parseBoundedStatusCode(rawStatusCode, http.StatusOK, http.StatusPermanentRedirect)
		if err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}
	}

	h.doRedirectGin(c, u.String(), statusCode)
}

// Redirect responds with 302 redirect a given number of times. Defaults to a
// relative redirect, but an ?absolute=true query param will trigger an
// absolute redirect.
func (h *HttpBin) Redirect(c *gin.Context) {
	relative := c.Query("absolute") != "true"
	h.handleRedirect(c, relative)
}

// RelativeRedirect responds with an HTTP 302 redirect a given number of times
func (h *HttpBin) RelativeRedirect(c *gin.Context) {
	h.handleRedirect(c, true)
}

// ResponseHeaders sets every incoming query parameter as a response header and
// returns the headers serialized as JSON.
//
// If the Content-Type query parameter is given and set to a "dangerous" value
// (i.e. one that might be rendered as HTML in a web browser), the keys and
// values in the JSON response body will be escaped.
func (h *HttpBin) ResponseHeaders(c *gin.Context) {
	// only set our own content type if one was not already set based on
	// incoming request params
	contentType := c.Query("content-type")
	if contentType == "" {
		c.Header("Content-Type", jsonContentType)
	}

	// actual HTTP response headers are not escaped, regardless of content type
	// (unlike the JSON serialized representation of those headers in the
	// response body, which MAY be escaped based on content type)
	for k, vs := range getRequestHeaders(c, h.excludeHeadersProcessor) {
		for _, v := range vs {
			c.Header(k, v)
		}
	}

	// if response content type is dangrous, escape keys and values before
	// serializing response body
	if h.mustEscapeResponse(contentType) {
		tmp := make(url.Values, len(c.Request.URL.Query()))
		for k, vs := range c.Request.URL.Query() {
			for _, v := range vs {
				tmp.Add(html.EscapeString(k), html.EscapeString(v))
			}
		}
		mustMarshalJSON(c.Writer, tmp)
	} else {
		mustMarshalJSON(c.Writer, c.Request.URL.Query())
	}
}

// Robots renders a basic robots.txt file
func (h *HttpBin) Robots(c *gin.Context) {
	robotsTxt := []byte(`User-agent: *Disallow: /deny`)
	writeResponse(c, http.StatusOK, textContentType, robotsTxt)
}

// SSE writes a stream of events over a duration after an optional
// initial delay.
func (h *HttpBin) SSE(c *gin.Context) {
	start := time.Now()
	var (
		count    = h.DefaultParams.SSECount
		delay    = h.DefaultParams.SSEDelay
		duration = h.DefaultParams.SSEDuration
		err      error
	)

	if userCount := c.Query("count"); userCount != "" {
		count, err = strconv.Atoi(userCount)
		if err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}
		if count < 1 || int64(count) > h.maxSSECount {
			writeError(c, http.StatusBadRequest, fmt.Errorf("count must be between 1 and %d", h.maxSSECount))
			return
		}
	}

	if userDuration := c.Query("duration"); userDuration != "" {
		duration, err = parseBoundedDuration(userDuration, 1, h.MaxDuration)
		if err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}
	}

	if userDelay := c.Query("delay"); userDelay != "" {
		delay, err = parseBoundedDuration(userDelay, 0, h.MaxDuration)
		if err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}
	}

	if duration+delay > h.MaxDuration {
		writeError(c, http.StatusBadRequest, fmt.Errorf("delay and duration combined must be less than %s", h.MaxDuration))
		return
	}

	pause := duration
	if count > 1 {
		// compensate for lack of pause after final write (i.e. if we're
		// writing 10 events, we will only pause 9 times)
		pause = duration / time.Duration(count-1)
	}

	// Initial delay before we send any response data
	if delay > 0 {
		select {
		case <-time.After(delay):
		// ok
		case <-c.Request.Context().Done():
			c.Status(499) // "Client Closed Request" https://httpstatuses.com/499
		}
	}

	c.Header("Trailer", "Server-Timing")
	defer func() {
		c.Header("Server-Timing", encodeServerTimings([]serverTiming{
			{"total_duration", time.Since(start), "total request duration"},
			{"initial_delay", delay, "initial delay"},
			{"write_duration", duration, "duration of writes after initial delay"},
			{"pause_per_write", pause, "computed pause between writes"},
		}))
	}()

	c.Header("Content-Type", sseContentType)
	c.Status(http.StatusOK)

	// special case when we only have one event to write
	if count == 1 {
		writeServerSentEvent(c.Writer, 0, time.Now())
		c.Writer.Flush()
		return
	}

	ticker := time.NewTicker(pause)
	defer ticker.Stop()

	for i := 0; i < count; i++ {
		writeServerSentEvent(c.Writer, i, time.Now())
		c.Writer.Flush()

		// don't pause after last byte
		if i == count-1 {
			return
		}

		select {
		case <-ticker.C:
			// ok
		case <-c.Request.Context().Done():
			return
		}
	}
}

// Status responds with the specified status code. TODO: support random choice
// from multiple, optionally weighted status codes.
func (h *HttpBin) Status(c *gin.Context) {
	rawStatus := c.Param("status")

	// simple case, specific status code is requested
	if !strings.Contains(rawStatus, ",") {
		statusCode, err := parseStatusCode(rawStatus)
		if err != nil {
			writeError(c, http.StatusBadRequest, err)
			return
		}
		h.doStatus(c, statusCode)
		return
	}

	// complex case, make a weighted choice from multiple status codes
	choices, err := parseWeightedChoices(rawStatus, strconv.Atoi)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	choice := weightedRandomChoice(choices)
	h.doStatus(c, choice)
}

func (h *HttpBin) doStatus(c *gin.Context, statusCode int) {
	// default to plain text content type, which may be overriden by headers
	// for special cases
	c.Header("Content-Type", textContentType)
	if specialCase, ok := h.statusSpecialCases[statusCode]; ok {
		for key, val := range specialCase.headers {
			c.Header(key, val)
		}
		c.Status(statusCode)
		if specialCase.body != nil {
			c.Writer.Write(specialCase.body)
		}
		return
	}
	c.Status(statusCode)
}

// StreamBytes streams N random bytes generated with an optional seed in chunks
// of a given size.
func (h *HttpBin) StreamBytes(c *gin.Context) {}

// handleBytes consolidates the logic for validating input params of the Bytes
// and StreamBytes endpoints and knows how to write the response in chunks if
// streaming is true.
func (h *HttpBin) handleBytes(c *gin.Context, streaming bool) {
	numBytes, err := strconv.Atoi(c.Param("numBytes"))
	if err != nil {
		writeError(c, http.StatusBadRequest, fmt.Errorf("invalid number of bytes: %q: %w", c.Param("numBytes"), err))
		return
	}

	// rng/seed
	rng, err := parseSeed(c.Query("seed"))
	if err != nil {
		writeError(c, http.StatusBadRequest, fmt.Errorf("invalid seed: %q: %w", c.Query("seed"), err))
		return
	}

	if numBytes < 0 {
		writeError(c, http.StatusBadRequest, fmt.Errorf("number of bytes must be positive"))
		return
	}

	// Special case 0 bytes and exit early, since streaming & chunk size do not
	// matter here.
	if numBytes == 0 {
		c.Header("Content-Type", "0")
		c.Status(http.StatusOK)
		return
	}

	if numBytes > int(h.MaxBodySize) {
		writeError(c, http.StatusBadRequest, fmt.Errorf("number of bytes must be less than %d", h.MaxBodySize))
		return
	}

	var chunkSize int
	var write func([]byte)

	if streaming {
		if c.Query("chunk_size") != "" {
			chunkSize, err = strconv.Atoi(c.Query("chunk_size"))
			if err != nil {
				writeError(c, http.StatusBadRequest, fmt.Errorf("invalid chunk size: %q: %w", c.Query("chunk_size"), err))
				return
			}
		} else {
			chunkSize = 10 * 1024
		}
		write = func() func(chunk []byte) {
			return func(chunk []byte) {
				c.Writer.Write(chunk)
				c.Writer.Flush()
			}
		}()
	} else {
		// if not streaming, we will write the whole response at once
		chunkSize = numBytes
		c.Header("Content-Type", strconv.Itoa(chunkSize))
		write = func(chunk []byte) {
			c.Writer.Write(chunk)
		}
	}

	c.Header("Content-Type", binaryContentType)
	c.Status(http.StatusOK)

	var chunk []byte
	for i := 0; i < numBytes; i++ {
		chunk = append(chunk, byte(rng.Intn(256)))
		if len(chunk) == chunkSize {
			write(chunk)
			chunk = nil
		}
	}
	if len(chunk) > 0 {
		write(chunk)
	}
}

// Stream responds with max(n, 100) lines of JSON-encoded request data.
func (h *HttpBin) Stream(c *gin.Context) {
	n, err := strconv.Atoi(c.Param("numLines"))
	if err != nil {
		writeError(c, http.StatusBadRequest, fmt.Errorf("invalid number of lines: %q: %w", c.Param("numLines"), err))
		return
	}

	if n > 100 {
		n = 100
	} else if n < 1 {
		n = 1
	}

	resp := &streamResponse{
		Args:    c.Request.URL.Query(),
		Headers: getRequestHeaders(c, h.excludeHeadersProcessor),
		Origin:  c.ClientIP(),
		URL:     c.Request.URL.String(),
	}

	for i := 0; i < n; i++ {
		resp.ID = i
		//	 Call json.Marshal directly to avoid pretty printing
		line, _ := json.Marshal(resp)
		c.Writer.Write(append(line, '\n'))
		c.Writer.Flush()
	}
}

// Trailers adds the header keys and values specified in the request's query
// parameters as HTTP trailers in the response.
//
// Trailers are returned in canonical form. Any forbidden trailer will result
// in an error.
func (h *HttpBin) Trailers(c *gin.Context) {
	// ensure all requested trailers are allowed
	for k := range c.Request.URL.Query() {
		if _, found := forbiddenTrailers[http.CanonicalHeaderKey(k)]; found {
			writeError(c, http.StatusBadRequest, fmt.Errorf("forbidden trailer: %q", k))
			return
		}
	}

	for k := range c.Request.URL.Query() {
		c.Header("Tailer", k)
	}

	h.RequestWithBody(c)
	c.Writer.Flush()

	for k, vs := range c.Request.Trailer {
		for _, v := range vs {
			c.Header(k, v)
		}
	}
}

// Unstable - returns 500, sometimes
func (h *HttpBin) Unstable(c *gin.Context) {
	var err error

	// rng/seed
	rng, err := parseSeed(c.Query("seed"))
	if err != nil {
		writeError(c, http.StatusBadRequest, fmt.Errorf("invalid seed: %q: %w", c.Query("seed"), err))
		return
	}

	//	 failure_rate
	failureRate := 0.5
	if rawFailureRate := c.Query("failure_rate"); rawFailureRate != "" {
		failureRate, err = strconv.ParseFloat(rawFailureRate, 64)
		if err != nil {
			writeError(c, http.StatusBadRequest, fmt.Errorf("invalid failure rate: %q: %w", rawFailureRate, err))
			return
		} else if failureRate < 0 || failureRate > 1 {
			writeError(c, http.StatusBadRequest, fmt.Errorf("invalid failure rate: %q: %w", rawFailureRate, err))
			return
		}
	}

	status := http.StatusOK
	if rng.Float64() < failureRate {
		status = http.StatusInternalServerError
	}

	c.Header("Content-Type", textContentType)
	c.Status(status)
}

// RequestWithBodyDiscard handles POST, PUT, and PATCH requests by responding with a
// JSON representation of the incoming request without body data
func (h *HttpBin) RequestWithBodyDiscard(c *gin.Context) {
	resp := &discardedBodyResponse{
		noBodyResponse: noBodyResponse{
			Args:    c.Request.URL.Query(),
			Headers: getRequestHeaders(c, h.excludeHeadersProcessor),
			Method:  c.Request.Method,
			Origin:  c.ClientIP(),
			URL:     c.Request.URL.String(),
		},
	}

	n, err := io.Copy(io.Discard, c.Request.Body)
	if err != nil {
		writeError(c, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err))
		return
	}

	resp.BytesReceived = n
	writeJSON(c, http.StatusOK, resp)
}

// UserAgent echoes the incoming User-Agent header
func (h *HttpBin) UserAgent(c *gin.Context) {
	writeJSON(c, http.StatusOK, &userAgentResponse{
		UserAgent: c.Request.UserAgent(),
	})
}

// UUID - responds with a generated UUID
func (h *HttpBin) UUID(c *gin.Context) {
	writeJSON(c, http.StatusOK, &uuidResponse{
		UUID: uuidv4(),
	})
}

// XML responds with an XML document
func (h *HttpBin) XML(c *gin.Context) {
	writeResponse(c, http.StatusOK, "application/xml", mustStaticAsset("sample.xml"))
}

func notImplementedHandler(c *gin.Context) {
	writeError(c, http.StatusNotImplemented, nil)
}
