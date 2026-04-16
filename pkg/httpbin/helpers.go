package httpbin

import (
	crypto_rand "crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
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
	// Registered as /base64/*path, where path is:
	//   - "/<data>" OR
	//   - "/<operation>/<data>"
	raw := strings.TrimPrefix(c.Param("path"), "/")
	operation := "encode"
	data := raw
	if op, rest, found := strings.Cut(raw, "/"); found {
		operation = op
		data = rest
	}

	return &base64Helper{
		operation: operation,
		data:      data,
		maxLen:    maxLen,
	}
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
	"application/octet-stream": true,
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
	_, _ = h.Write([]byte(input))
	return fmt.Sprintf("%x", h.Sum(nil))
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
	var headers http.Header
	if c.Request != nil {
		headers = c.Request.Header.Clone()
	} else {
		headers = make(http.Header)
	}

	// Include Host and Transfer-Encoding, which are not guaranteed to exist in
	// Request.Header.
	if c.Request != nil {
		if c.Request.Host != "" {
			headers.Set("Host", c.Request.Host)
		}
		if len(c.Request.TransferEncoding) > 0 {
			headers.Set("Transfer-Encoding", strings.Join(c.Request.TransferEncoding, ","))
		}
	}

	if fn != nil {
		return fn(headers)
	}
	return headers
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

func parseStatusCode(input string) (int, error) {
	return parseBoundedStatusCode(input, 100, 599)
}
func parseBoundedStatusCode(input string, minVal, maxVal int) (int, error) {
	code, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid status code: %q: %w", input, err)
	}
	if code < minVal || code > maxVal {
		return 0, fmt.Errorf("invalid status code: %d not in range [%d, %d]", code, minVal, maxVal)
	}
	return code, nil
}

// Server-Timing header/trailer helpers. See MDN docs for reference:
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Server-Timing
type serverTiming struct {
	name string
	dur  time.Duration
	desc string
}

func getURL(r *http.Request) *url.URL {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = r.Header.Get("X-Forwarded-Protocol")
	}
	if scheme == "" && r.Header.Get("X-Forwarded-Ssl") == "on" {
		scheme = "https"
	}
	if scheme == "" && r.TLS != nil {
		scheme = "https"
	}
	if scheme == "" {
		scheme = "http"
	}

	host := r.URL.Host
	if host == "" {
		host = r.Host
	}

	return &url.URL{
		Scheme:     scheme,
		Opaque:     r.URL.Opaque,
		User:       r.URL.User,
		Host:       host,
		Path:       r.URL.Path,
		RawPath:    r.URL.RawPath,
		ForceQuery: r.URL.ForceQuery,
		RawQuery:   r.URL.RawQuery,
		Fragment:   r.URL.Fragment,
	}
}
func encodeServerTimings(timings []serverTiming) string {
	entries := make([]string, len(timings))
	for i, t := range timings {
		ms := t.dur.Seconds() * 1e3
		entries[i] = fmt.Sprintf("%s;dur=%0.2f;desc=\"%s\"", t.name, ms, t.desc)
	}
	return strings.Join(entries, ", ")
}

// syntheticByteStream implements the ReadSeeker interface to allow reading
// arbitrary subsets of bytes up to a maximum size given a function for
// generating the byte at a given offset.
type syntheticByteStream struct {
	mu sync.Mutex

	size         int64
	factory      func(int64) byte
	pausePerByte time.Duration

	// internal offset for tracking the current position in the stream
	offset int64
}

// newSyntheticByteStream returns a new stream of bytes of a specific size,
// given a factory function for generating the byte at a given offset.
func newSyntheticByteStream(size int64, duration time.Duration, factory func(int64) byte) io.ReadSeeker {
	return &syntheticByteStream{
		size:         size,
		pausePerByte: duration / time.Duration(size),
		factory:      factory,
	}
}

// Read implements the Reader interface for syntheticByteStream
func (s *syntheticByteStream) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	start := s.offset
	end := start + int64(len(p))
	var err error
	if end >= s.size {
		err = io.EOF
		end = s.size
	}

	for idx := start; idx < end; idx++ {
		p[idx-start] = s.factory(idx)
	}
	s.offset = end

	if s.pausePerByte > 0 {
		time.Sleep(s.pausePerByte * time.Duration(end-start))
	}

	return int(end - start), err
}

// Seek implements the Seeker interface for syntheticByteStream
func (s *syntheticByteStream) Seek(offset int64, whence int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch whence {
	case io.SeekStart:
		s.offset = offset
	case io.SeekCurrent:
		s.offset += offset
	case io.SeekEnd:
		s.offset = s.size - offset
	default:
		return 0, errors.New("Seek: invalid whence")
	}

	if s.offset < 0 {
		return 0, errors.New("Seek: invalid offset")
	}

	return s.offset, nil
}

// writeServerSentEvent writes the bytes that constitute a single server-sent
// event message, including both the event type and data.
func writeServerSentEvent(dst io.Writer, id int, ts time.Time) {
	dst.Write([]byte("event: ping\n"))
	dst.Write([]byte("data: "))
	json.NewEncoder(dst).Encode(serverSentEvent{
		ID:        id,
		Timestamp: ts.UnixMilli(),
	})
	// each SSE ends with two newlines (\n\n), the first of which is written
	// automatically by json.NewEncoder().Encode()
	dst.Write([]byte("\n"))
}

// weightedChoice represents a choice with its associated weight.
type weightedChoice[T any] struct {
	Choice T
	Weight float64
}

// parseWeighteChoices parses a comma-separated list of choices in
// choice:weight format, where weight is an optional floating point number.
func parseWeightedChoices[T any](rawChoices string, parser func(string) (T, error)) ([]weightedChoice[T], error) {
	if rawChoices == "" {
		return nil, nil
	}

	var (
		choicePairs = strings.Split(rawChoices, ",")
		choices     = make([]weightedChoice[T], 0, len(choicePairs))
		err         error
	)
	for _, choicePair := range choicePairs {
		weight := 1.0
		rawChoice, rawWeight, found := strings.Cut(choicePair, ":")
		if found {
			weight, err = strconv.ParseFloat(rawWeight, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid weight value: %q", rawWeight)
			}
		}
		choice, err := parser(rawChoice)
		if err != nil {
			return nil, fmt.Errorf("invalid choice value: %q", rawChoice)
		}
		choices = append(choices, weightedChoice[T]{Choice: choice, Weight: weight})
	}
	return choices, nil
}

// weightedRandomChoice returns a randomly chosen element from the weighted
// choices, given as a slice of "choice:weight" strings where weight is a
// floating point number. Weights do not need to sum to 1.
func weightedRandomChoice[T any](choices []weightedChoice[T]) T {
	// Calculate total weight
	var totalWeight float64
	for _, wc := range choices {
		totalWeight += wc.Weight
	}
	randomNumber := rand.Float64() * totalWeight
	currentWeight := 0.0
	for _, wc := range choices {
		currentWeight += wc.Weight
		if randomNumber < currentWeight {
			return wc.Choice
		}
	}
	panic("failed to select a weighted random choice")
}

// set of keys that may not be specified in trailers, per
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Trailer#directives
var forbiddenTrailers = map[string]struct{}{
	http.CanonicalHeaderKey("Authorization"):     {},
	http.CanonicalHeaderKey("Cache-Control"):     {},
	http.CanonicalHeaderKey("Content-Encoding"):  {},
	http.CanonicalHeaderKey("Content-Length"):    {},
	http.CanonicalHeaderKey("Content-Range"):     {},
	http.CanonicalHeaderKey("Content-Type"):      {},
	http.CanonicalHeaderKey("Host"):              {},
	http.CanonicalHeaderKey("Max-Forwards"):      {},
	http.CanonicalHeaderKey("Set-Cookie"):        {},
	http.CanonicalHeaderKey("TE"):                {},
	http.CanonicalHeaderKey("Trailer"):           {},
	http.CanonicalHeaderKey("Transfer-Encoding"): {},
}

func uuidv4() string {
	buff := make([]byte, 16)
	if _, err := crypto_rand.Read(buff[:]); err != nil {
		panic(err)
	}
	buff[6] = (buff[6] & 0x0f) | 0x40 // Version 4
	buff[8] = (buff[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", buff[0:4], buff[4:6], buff[6:8], buff[8:10], buff[10:])
}
