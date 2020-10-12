package sumologrus

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type SumoLogicMesssage struct {
	Tags  []string    `json:"tags"`
	Host  string      `json:"host"`
	Level string      `json:"level"`
	Data  interface{} `json:"data"`
}

type SumoLogicHook struct {
	endPointURL           string
	tags                  []string
	host                  string
	levels                []logrus.Level
	logger                *logrus.Logger
	verbose               bool
	interval              time.Duration
	batchSize             int
	gZip                  bool
	maxConcurrentRequests int
	retryAfter            func(int) time.Duration

	msgs     chan SumoLogicMesssage
	quit     chan struct{}
	shutdown chan struct{}
}

var (
	newline = []byte{'\n'}
)

func New(endPointUrl string, host string, level logrus.Level, tags ...string) *SumoLogicHook {
	cfg := Config{
		EndPointURL: endPointUrl,
		Host:        host,
		Level:       level,
		Tags:        tags,
	}
	hook, _ := NewWithConfig(makeConfig(cfg))

	return hook
}

func NewWithConfig(c Config) (*SumoLogicHook, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	levels := []logrus.Level{}
	log := logrus.New()
	log.Out = os.Stdout

	log.WithFields(logrus.Fields{
		"application": "sumologrus",
	})

	for _, l := range []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	} {
		if l <= c.Level {
			levels = append(levels, l)
		}
	}

	var tagList []string
	for _, tag := range c.Tags {
		tagList = append(tagList, tag)
	}

	hook := &SumoLogicHook{
		host:                  c.Host,
		tags:                  tagList,
		endPointURL:           c.EndPointURL,
		levels:                levels,
		logger:                log,
		verbose:               c.Verbose,
		interval:              c.Interval,
		batchSize:             c.BatchSize,
		maxConcurrentRequests: c.maxConcurrentRequests,
		gZip:                  c.GZip,
		retryAfter:            c.RetryAfter,
		msgs:                  make(chan SumoLogicMesssage, 100),
		quit:                  make(chan struct{}),
		shutdown:              make(chan struct{}),
	}

	hook.startLoop()

	return hook, nil
}

func (h *SumoLogicHook) Fire(entry *logrus.Entry) (err error) {
	data := map[string]interface{}{
		"message": entry.Message,
		"fields":  entry.Data,
	}

	msg := SumoLogicMesssage{
		Tags:  h.tags,
		Host:  h.host,
		Level: strings.ToUpper(entry.Level.String()),
		Data:  data,
	}
	err = h.queue(msg)

	return
}

func (h *SumoLogicHook) queue(msg SumoLogicMesssage) (err error) {
	defer func() {
		// When the `msgs` channel is closed writing to it will trigger a panic.
		// To avoid letting the panic propagate to the caller we recover from it
		// and instead report that the client has been closed and shouldn't be
		// used anymore.
		if recover() != nil {
			err = ErrClosed
		}
	}()
	h.msgs <- msg
	return
}

func (h *SumoLogicHook) gZipData(b []byte) (*bytes.Buffer, error) {
	payload := bytes.Join([][]byte{b}, newline)

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(payload); err != nil {
		h.errorf("error writing into io.Writer - %s", err)
		return nil, err
	}

	if err := w.Close(); err != nil {
		h.errorf("error closing write buffer - %s", err)
		return nil, err
	}

	return &buf, nil
}

func (h *SumoLogicHook) upload(b []byte) (err error) {
	buf := bytes.NewBuffer(b)

	if h.gZip {
		buf, err = h.gZipData(b)
		if err != nil {
			h.errorf("error compressing data - %s", err)
			return err
		}
	}

	req, err := http.NewRequest(
		"POST",
		h.endPointURL,
		buf,
	)

	if err != nil {
		h.errorf("creating request - %s", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if h.gZip {
		req.Header.Set("Content-Encoding", "gzip")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		h.errorf("sending request - %s", err)
		return err
	}

	defer resp.Body.Close()
	return nil
}

func (h *SumoLogicHook) Levels() []logrus.Level {
	return h.levels
}

func (h *SumoLogicHook) Flush() (err error) {
	defer func() {
		// Always recover, a panic could be raised if `h`.quit was closed which
		// means the method was called more than once.
		if recover() != nil {
			err = ErrClosed
		}
	}()

	close(h.quit)
	<-h.shutdown
	return
}

// Asychronously send a batched requests.
func (h *SumoLogicHook) sendAsync(msgs []SumoLogicMesssage, wg *sync.WaitGroup, ex *executor) {
	wg.Add(1)

	if !ex.do(func() {
		defer wg.Done()
		defer func() {
			// In case a bug is introduced in the send function that triggers
			// a panic, we don't want this to ever crash the application so we
			// catch it here and log it instead.
			if err := recover(); err != nil {
				h.errorf("panic - %s", err)
			}
		}()
		h.send(msgs)
	}) {
		wg.Done()
		h.errorf("sending messages failed - %s", ErrTooManyRequests)
	}
}

// Send batch request.
func (h *SumoLogicHook) send(msgs []SumoLogicMesssage) {
	const attempts = 10

	b, err := json.Marshal(msgs)
	if err != nil {
		h.errorf("marshalling messages - %s", err)
		return
	}

	for i := 0; i != attempts; i++ {
		if err = h.upload(b); err == nil {
			return
		}

		// Wait for either a retry timeout or the client to be closed.
		select {
		case <-time.After(h.retryAfter(i)):
		case <-h.quit:
			h.errorf("%d messages dropped because they failed to be sent and the client was closed", len(msgs))
			return
		}
	}

	h.errorf("%d messages dropped because they failed to be sent after %d attempts", len(msgs), attempts)
}

func (h *SumoLogicHook) startLoop() {
	go h.loop()
}

func (h *SumoLogicHook) loop() {
	defer close(h.shutdown)
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	// Batch send the current log lines each Interval
	tick := time.NewTicker(h.interval)
	defer tick.Stop()

	ex := newExecutor(h.maxConcurrentRequests)
	defer ex.close()

	mq := messageQueue{
		maxBatchSize:  h.batchSize,
		maxBatchBytes: maxBatchBytes,
	}

	for {
		select {
		case msg := <-h.msgs:
			h.push(&mq, msg, wg, ex)
		case <-tick.C:
			h.flush(&mq, wg, ex)
		case <-h.quit:
			h.debugf("exit requested – draining messages")

			// Drain the msg channel, we have to close it first so no more
			// messages can be pushed and otherwise the loop would never end.
			close(h.msgs)

			for msg := range h.msgs {
				h.push(&mq, msg, wg, ex)
			}

			h.flush(&mq, wg, ex)
			h.debugf("exit")
			return
		}

	}
}

func (h *SumoLogicHook) push(q *messageQueue, msg SumoLogicMesssage, wg *sync.WaitGroup, ex *executor) {

	h.debugf("buffer (%d/%d) %v", len(q.pending), h.batchSize, msg)

	if msgs := q.push(msg); msgs != nil {
		h.debugf("exceeded messages batch limit with batch of %d messages – flushing", len(msgs))
		h.sendAsync(msgs, wg, ex)
	}
}

func (h *SumoLogicHook) flush(q *messageQueue, wg *sync.WaitGroup, ex *executor) {
	if msgs := q.flush(); msgs != nil {
		h.debugf("flushing %d messages", len(msgs))
		h.sendAsync(msgs, wg, ex)

	}
}

func (h *SumoLogicHook) debugf(msg string, args ...interface{}) {
	if h.verbose {
		h.logger.Printf(msg, args...)
	}
}

func (h *SumoLogicHook) log(msg string, args ...interface{}) {
	if h.verbose {
		h.logger.Printf(msg, args...)
	}
}

func (h *SumoLogicHook) errorf(format string, args ...interface{}) {
	h.logger.Errorf(format, args...)
}
