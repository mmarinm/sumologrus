# sumologrus
[![deepcode](https://www.deepcode.ai/api/gh/badge?key=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJwbGF0Zm9ybTEiOiJnaCIsIm93bmVyMSI6Im1tYXJpbm0iLCJyZXBvMSI6InN1bW9sb2dydXMiLCJpbmNsdWRlTGludCI6ZmFsc2UsImF1dGhvcklkIjoyMjg0NiwiaWF0IjoxNjAwNjUwOTIyfQ.mkL4nI8anw9ebEVB6yeAfW4uIt_BqkX7sgpC6rgO7UQ)](https://www.deepcode.ai/app/gh/mmarinm/sumologrus/_/dashboard?utm_content=gh%2Fmmarinm%2Fsumologrus) [![codecov](https://codecov.io/gh/mmarinm/sumologrus/branch/master/graph/badge.svg)](https://codecov.io/gh/mmarinm/sumologrus) [![mmarinm](https://circleci.com/gh/mmarinm/sumologrus.svg?style=shield)](https://app.circleci.com/pipelines/gh/mmarinm/sumologrus)

SumoLogic Hook for [Logrus](https://github.com/sirupsen/logrus) <img src="http://i.imgur.com/hTeVwmJ.png" width="40" height="40" alt=":walrus:" class="emoji" title=":walrus:"/>

## Description
sumologrus is async hook that helps upload logs collected with Logrus logger to [Sumologic HTTP Source](https://help.sumologic.com/03Send-Data/Sources/02Sources-for-Hosted-Collectors/HTTP-Source/Upload-Data-to-an-HTTP-Source) . Logs are flushed to Sumologic periodically (5 sec), if size of the batch reaches 250 logs or by explicitly calling Flush method


## Configuration

The following tables list the configurable parameters 

| Parameter | Description | Default |
| ----- | ----------- | ------ |
|`EndpointURL`|Sumologic endpoint|`""`|
|`Tags`|Sumologic tags|`[]`|
|`Host`|Sumologic host|`""`|
|`Level`|Log Level|`logrus.PanicLevel`|
|`Interval`|Time interval to flush logs |`5s`|
|`BatchSize`|Limits number of batched logs|`100`|
|`Verbose`|Enables Sumorus hook logs|`false`|


## Usage

```go
package main

import (
	"github.com/sirupsen/logrus"
	"github.com/mmarinm/sumologrus"
)

var endpoint string = "YOUR_SUMOLOGIC_HTTP_HOSTED_ENDPOINT"
var host = "YOUR_HOST_NAME"

func main() {
	log := logrus.New()
	sumoLogicHook := sumologrus.New(endpoint, host, logrus.InfoLevel, "tag1", "tag2")
	defer sumoLogicHook.Flush()
	log.Hooks.Add(sumoLogicHook)

	log.WithFields(logrus.Fields{
		"name": "hurley",
		"age":  29,
	}).Error("Hello world!")
}
```


### Override default Config values: 

```go
package main

import (
	"github.com/sirupsen/logrus"
	"github.com/mmarinm/sumologrus"
	"time"
)

var endpoint string = "YOUR_SUMOLOGIC_HTTP_HOSTED_ENDPOINT"
var host = "YOUR_HOST_NAME"

func main() {
	log := logrus.New()
	sumoLogicHook, err := sumologrus.NewWithConfig(sumologrus.Config{
		EndPointURL: endpoint, 
		Tags: []string{"tag1", "tag2"},
		Host: host, 
		Level: logrus.InfoLevel, 
		Interval: 3 * time.Second,
		BatchSize: 10,
		Verbose: true,
	})
	if err != nil {
		panic(err)
	}
	
	defer sumoLogicHook.Flush()
	log.Hooks.Add(sumoLogicHook)

	log.WithFields(logrus.Fields{
		"name": "sawyer",
		"age":  29,
	}).Error("Hello world!")
}
```

