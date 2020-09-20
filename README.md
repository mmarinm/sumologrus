# SumoLogic Hook for [Logrus](https://github.com/sirupsen/logrus) <img src="http://i.imgur.com/hTeVwmJ.png" width="40" height="40" alt=":walrus:" class="emoji" title=":walrus:"/>

## Description
sumologrus is async Logrus hook. Logs are flushed to Sumologic periodically (5 sec), if size of the batch reaches 250 logs or by explicitly calling Flush method


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
	sumoLogicHook := sumologrus.NewWithConfig(sumologrus.Config{
		EndPointURL: endpoint, 
		Tags: []string{"tag1", "tag2"},
		Host: host, 
		Level: logrus.InfoLevel, 
		Interval: 3 * time.Second,
		BatchSize: 10,
		Verbose: true,
	})
	defer sumoLogicHook.Flush()
	log.Hooks.Add(sumoLogicHook)

	log.WithFields(logrus.Fields{
		"name": "sawyer",
		"age":  29,
	}).Error("Hello world!")
}
```

