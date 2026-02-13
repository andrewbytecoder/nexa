package httpbin

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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

func (h *HttpBin) doRedirect(c *gin.Context, path string, code int) {
	var sb strings.Builder
	if strings.HasPrefix(path, "/") && strings.HasPrefix(path, "//") {
		sb.WriteString(h.)
	}
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

}

func (h *HttpBin) AbsoluteRedirect(c *gin.Context) {
}
