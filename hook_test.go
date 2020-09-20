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
	t.Run("Should flush the logs when Flush is called", func(t *testing.T) {
		// t.Skip()
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
		hook.verbose = true

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
				got = b
			}
		}()
		hook.Flush()

		assert.JSONEq(t, want, got)
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
			  "tags": null
			}
		]`
		body, server := mockServer()
		defer server.Close()

		m.Lock()
		hook := New(server.URL, "admin-lambda-test", logrus.InfoLevel)
		hook.interval = 100 * time.Millisecond
		hook.verbose = true
		m.Unlock()

		log := logrus.New()
		log.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.RFC3339, FullTimestamp: true})
		log.Hooks.Add(hook)

		log.WithFields(logrus.Fields{
			"name": "kate",
			"age":  33,
		}).Error("Hello world!")

		go func() {
			for b := range body {
				got = b
			}
		}()
		time.Sleep(150 * time.Millisecond)
		hook.Flush()

		assert.JSONEq(t, want, got)
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
			  "tags": null
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
			  "tags": null
			}]`

		body, server := mockServer()
		defer server.Close()

		hook := New(server.URL, "admin-lambda-test", logrus.InfoLevel)
		hook.verbose = true
		hook.size = 1

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
