package sumologrus

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sirupsen/logrus"
)

func mockServer() (chan string, *httptest.Server) {
	done := make(chan string)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := bytes.NewBuffer(nil)
		io.Copy(buf, r.Body)
		done <- buf.String()
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
			Verbose:     true,
		}))

		_, err2 := NewWithConfig(makeConfig(Config{
			EndPointURL: "https://example.com",
			Host:        "admin-lambda-test",
			Level:       logrus.InfoLevel,
			Tags:        []string{"tag1", "tag2"},
			Verbose:     true,
			BatchSize:   -10,
		}))

		assert.EqualError(t, err1, expectedErrorString1)
		assert.EqualError(t, err2, expectedErrorString2)
	})
	t.Run("Should flush the logs when Flush is called", func(t *testing.T) {
		var m sync.Mutex
		var got, want string
		want = `[
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
		assert.JSONEq(t, want, got)
		m.Unlock()
	})

	t.Run("Should flush the logs after interval", func(t *testing.T) {
		var m sync.Mutex
		var got string
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
		assert.JSONEq(t, want, got)
		m.Unlock()
	})

	t.Run("Should flush the logs if batch size is reached", func(t *testing.T) {
		var m sync.Mutex
		var got1, got2, want1, want2 string
		want1 = `[
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
		want2 = `[
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

		log.WithFields(logrus.Fields{
			"name": "sawyer",
			"age":  32,
		}).Error("Hello world!")

		cnt := 0
		go func() {
			for b := range body {
				m.Lock()
				cnt++
				if cnt == 1 {
					got1 = b
				}
				if cnt == 2 {
					got2 = b
				}

				m.Unlock()
			}
		}()

		time.Sleep(50 * time.Millisecond)
		m.Lock()
		assert.JSONEq(t, want1, got1)
		assert.JSONEq(t, want2, got2)
		m.Unlock()
		hook.Flush()
	})
}
