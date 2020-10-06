package sumologrus

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func gUnzipData(data []byte) (resData []byte, err error) {
	b := bytes.NewBuffer(data)

	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func mockServer() (chan []byte, *httptest.Server) {
	done := make(chan []byte)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := logrus.New()
		responseData, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}

		done <- responseData
	}))

	return done, server
}

func TestHook(t *testing.T) {
	t.Run("Should error if invalid configs", func(t *testing.T) {
		expectedErrorString1 := "NewWithConfig: negative or 0 time intervals are not supported Config.Interval: -100ms"
		expectedErrorString2 := "NewWithConfig: negative or 0 batch sizes are not supported Config.BatchSize: -10"
		_, err1 := NewWithConfig(makeConfig(Config{
			EndPointURL: "https://example.com",
			Host:        "admin-lambda-test",
			Level:       logrus.InfoLevel,
			Tags:        []string{"tag1", "tag2"},
			Interval:    -100 * time.Millisecond,
		}))

		_, err2 := NewWithConfig(makeConfig(Config{
			EndPointURL: "https://example.com",
			Host:        "admin-lambda-test",
			Level:       logrus.InfoLevel,
			Tags:        []string{"tag1", "tag2"},
			BatchSize:   -10,
		}))

		assert.EqualError(t, err1, expectedErrorString1)
		assert.EqualError(t, err2, expectedErrorString2)
	})

	t.Run("Should flush the logs when Flush is called", func(t *testing.T) {
		var m sync.Mutex
		var got []byte
		want := `[
			{
			  "data": {
				"fields": {
				  "age": 33,
				  "name": "kate"
				},
				"message": "Hello world!"
			  },
			  "host": "admin-lambda-test",
			  "level": "ERROR",
			  "tags": null
			},
			{
			  "data": {
				"fields": {
				  "age": 32,
				  "name": "sawyer"
				},
				"message": "Hello world!"
			  },
			  "host": "admin-lambda-test",
			  "level": "ERROR",
			  "tags": null
			}
		]`

		body, server := mockServer()
		defer server.Close()

		hook := New(server.URL, "admin-lambda-test", logrus.InfoLevel)

		log := logrus.New()
		log.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.RFC3339, FullTimestamp: true})
		log.Hooks.Add(hook)

		log.WithFields(logrus.Fields{
			"name": "kate",
			"age":  33,
		}).Error("Hello world!")

		log.WithFields(logrus.Fields{
			"name": "sawyer",
			"age":  32,
		}).Error("Hello world!")

		go func() {
			for b := range body {
				m.Lock()
				got = b
				m.Unlock()
			}
		}()
		hook.Flush()

		m.Lock()
		got, err := gUnzipData(got) // uncompress recieved data to be compared as JSON
		assert.Nil(t, err)
		assert.JSONEq(t, want, string(got))
		m.Unlock()
	})

	t.Run("Should flush the logs after interval", func(t *testing.T) {
		var m sync.Mutex
		var got []byte
		want := `[
			{
			  "data": {
				"fields": {
				  "age": 33,
				  "name": "kate"
				},
				"message": "Hello world!"
			  },
			  "host": "admin-lambda-test",
			  "level": "ERROR",
			  "tags": ["tag1", "tag2"]
			}
		]`
		body, server := mockServer()
		defer server.Close()

		hook, _ := NewWithConfig(makeConfig(Config{
			EndPointURL: server.URL,
			Host:        "admin-lambda-test",
			Level:       logrus.InfoLevel,
			Tags:        []string{"tag1", "tag2"},
			Interval:    100 * time.Millisecond,
			Verbose:     true,
		}))

		log := logrus.New()
		log.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.RFC3339, FullTimestamp: true})
		log.Hooks.Add(hook)

		log.WithFields(logrus.Fields{
			"name": "kate",
			"age":  33,
		}).Error("Hello world!")

		go func() {
			for b := range body {
				m.Lock()
				got = b
				m.Unlock()
			}
		}()
		time.Sleep(150 * time.Millisecond)
		hook.Flush()

		m.Lock()
		got, err := gUnzipData(got) // uncompress recieved data to be compared as JSON
		assert.Nil(t, err)
		assert.JSONEq(t, want, string(got))
		m.Unlock()
	})

	t.Run("Should flush the logs if batch size is reached", func(t *testing.T) {
		var m sync.Mutex
		var got []byte
		want := `[
			{
			  "data": {
				"fields": {
				  "age": 33,
				  "name": "kate"
				},
				"message": "Hello world!"
			  },
			  "host": "admin-lambda-test",
			  "level": "ERROR",
			  "tags": ["tag1", "tag2"]
			}]`

		body, server := mockServer()
		defer server.Close()

		hook, _ := NewWithConfig(makeConfig(Config{
			EndPointURL: server.URL,
			Host:        "admin-lambda-test",
			Level:       logrus.InfoLevel,
			Tags:        []string{"tag1", "tag2"},
			BatchSize:   1,
		}))

		log := logrus.New()
		log.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.RFC3339, FullTimestamp: true})
		log.Hooks.Add(hook)

		log.WithFields(logrus.Fields{
			"name": "kate",
			"age":  33,
		}).Error("Hello world!")

		go func() {
			for b := range body {
				m.Lock()
				got = b
				m.Unlock()
			}
		}()

		time.Sleep(100 * time.Millisecond)
		hook.Flush()

		m.Lock()
		got, err := gUnzipData(got) // uncompress recieved data to be compared as JSON
		assert.Nil(t, err)
		assert.JSONEq(t, want, string(got))
		m.Unlock()

	})
}
