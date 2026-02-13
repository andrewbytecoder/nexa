package httpbin

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// base64Helper encapsulates a base64 operation (encode or decode) and its input
// data.
type base64Helper struct {
	maxLen    int64
	operation string
	data      string
}

// newBase64Helper creates a new base64Helper from a URL path, which should be
// in one of two forms:
// - /base64/<base64_encoded_data>
// - /base64/<operation>/<base64_encoded_data>
func newBase64Helper(c *gin.Context, maxLen int64) *base64Helper {
	b := &base64Helper{
		operation: c.Param("operation"),
		data:      c.Param("data"),
		maxLen:    maxLen,
	}

	if b.operation == "" {
		b.operation = "encode"
	}
	return b
}

func (b *base64Helper) transform() ([]byte, error) {
	if dataLen := int64(len(b.data)); dataLen == 0 {
		return nil, errors.New("no data provided")
	} else if dataLen > b.maxLen {
		return nil, errors.New("data too large")
	}

	switch b.operation {
	case "encode":
		return b.encode(), nil
	case "decode":
		return b.decode()
	default:
		return nil, errors.New("invalid operation")
	}
}

func (b *base64Helper) encode() []byte {
	// always encode using the URL-safe character set
	buff := make([]byte, base64.URLEncoding.EncodedLen(len(b.data)))
	base64.URLEncoding.Encode(buff, []byte(b.data))
	return buff
}

func (b *base64Helper) decode() ([]byte, error) {
	// first, try URL-safe encoding, then std encoding
	if result, err := base64.URLEncoding.DecodeString(b.data); err == nil {
		return result, nil
	}
	return base64.StdEncoding.DecodeString(b.data)
}

// The following content types are considered safe enough to skip HTML-escaping
// response bodies.
//
// See [1] for an example of the wide variety of unsafe content types, which
// varies by browser vendor and could change in the future.
//
// [1]: https://github.com/BlackFan/content-type-research/blob/4e4347254/XSS.md
var safeContentTypes = map[string]bool{
	"text/plain":               true,
	"application/json":         true,
	"application/octet-string": true,
}

// isDangerousContentType determines whether the given Content-Type header
// value could be unsafe (e.g. at risk of XSS) when rendered by a web browser.
func isDangerousContentType(ct string) bool {
	mediatype, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return true
	}
	return !safeContentTypes[mediatype]
}

// Returns a new rand.Rand from the given seed string.
func parseSeed(rawSeed string) (*rand.Rand, error) {
	var seed int64
	if rawSeed != "" {
		var err error
		seed, err = strconv.ParseInt(rawSeed, 10, 64)
		if err != nil {
			return nil, err
		}
	} else {
		seed = time.Now().UnixNano()
	}

	src := rand.NewSource(seed)
	rng := rand.New(src)
	return rng, nil
}

func sha1hash(input string) string {
	h := sha1.New()
	return fmt.Sprintf("%x", h.Sum([]byte(input)))
}

func mustMarshalJSON(w io.Writer, val any) {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(val); err != nil {
		panic(err.Error())
	}
}

// requestHeaders takes in incoming request and returns an http.Header map
// suitable for inclusion in our response data structures.
//
// This is necessary to ensure that the incoming Host and Transfer-Encoding
// headers are included, because golang only exposes those values on the
// http.Request struct itself.
func getRequestHeaders(c *gin.Context, fn headersProcessorFunc) http.Header {
	c.Header("Host", c.Request.Host)
	if r := c.Request; r != nil {
		if len(r.TransferEncoding) > 0 {
			c.Header("Transfer-Encoding", strings.Join(r.TransferEncoding, ","))
		}
	}

	if fn != nil {
		return fn(c.Request.Header)
	}
	return c.Request.Header
}

// parseDuration takes a user's input as a string and attempts to convert it
// into a time.Duration. If not given as a go-style duration string, the input
// is assumed to be seconds as a float.
func parseDuration(input string) (time.Duration, error) {
	d, err := time.ParseDuration(input)
	if err != nil {
		n, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return 0, err
		}
		d = time.Duration(n*1000) * time.Millisecond
	}
	return d, nil
}

// parseBoundedDuration parses a time.Duration from user input and ensures that
// it is within a given maximum and minimum time
func parseBoundedDuration(input string, minVal, maxVal time.Duration) (time.Duration, error) {
	d, err := parseDuration(input)
	if err != nil {
		return 0, err
	}

	if d > maxVal {
		err = fmt.Errorf("duration %s longer than %s", d, maxVal)
	} else if d < minVal {
		err = fmt.Errorf("duration %s shorter than %s", d, minVal)
	}
	return d, err
}

// Server-Timing header/trailer helpers. See MDN docs for reference:
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Server-Timing
type serverTiming struct {
	name string
	dur  time.Duration
	desc string
}

func encodeServerTimings(timings []serverTiming) string {
	entries := make([]string, len(timings))
	for i, t := range timings {
		ms := t.dur.Seconds() * 1e3
		entries[i] = fmt.Sprintf("%s;dur=%0.2f;desc=\"%s\"", t.name, ms, t.desc)
	}
	return strings.Join(entries, ", ")
}
