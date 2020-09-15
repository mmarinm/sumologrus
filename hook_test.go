package sumologrus

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func mockServer() (chan []byte, *httptest.Server) {
	done := make(chan []byte, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := bytes.NewBuffer(nil)
		io.Copy(buf, r.Body)

		var v interface{}
		err := json.Unmarshal(buf.Bytes(), &v)
		if err != nil {
			panic(err)
		}

		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			panic(err)
		}

		done <- b
	}))

	return done, server
}

func TestHook(t *testing.T) {
	t.Run("Should flush the logs when Flush is called", func(t *testing.T) {
		var got []byte
		want := formatBytes([]byte(`[
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
		]`))
		body, server := mockServer()
		defer server.Close()

		hook := NewSumoLogicHook(server.URL, "admin-lambda-test", logrus.InfoLevel)

		hook.verbose = true

		log := logrus.New()
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
		hook.Flush()

		assert.Equal(t, got, want)
	})

	t.Run("Should flush the logs after interval", func(t *testing.T) {
		var got []byte
		want := formatBytes([]byte(`[
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
		]`))
		body, server := mockServer()
		defer server.Close()

		hook := NewSumoLogicHook(server.URL, "admin-lambda-test", logrus.InfoLevel)
		hook.interval = 100 * time.Millisecond
		hook.verbose = true

		log := logrus.New()
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

		assert.Equal(t, got, want)
	})

	t.Run("Should flush the logs if batch size is reached", func(t *testing.T) {
		var got, want bytes.Buffer
		kate := formatBytes([]byte(`[
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
		]`))

		sawyer := formatBytes([]byte(`[
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
		]`))

		want.Write(kate)
		want.Write(sawyer)

		body, server := mockServer()
		defer server.Close()

		hook := NewSumoLogicHook(server.URL, "admin-lambda-test", logrus.InfoLevel)
		hook.verbose = true
		hook.size = 1

		log := logrus.New()
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
				got.Write(b)
			}
		}()

		time.Sleep(250 * time.Millisecond)

		assert.Equal(t, got.Bytes(), want.Bytes())

		hook.Flush()
	})
}

func JSONBytesEqual(a, b []byte) (bool, error) {
	var j, j2 interface{}
	if err := json.Unmarshal(a, &j); err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, &j2); err != nil {
		return false, err
	}
	return reflect.DeepEqual(j2, j), nil
}

func formatBytes(b []byte) []byte {
	var v interface{}
	err := json.Unmarshal(b, &v)
	if err != nil {
		panic(err)
	}

	f, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}

	return f
}
