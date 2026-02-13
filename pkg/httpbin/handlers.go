package httpbin

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
