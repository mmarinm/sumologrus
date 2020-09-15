# SumoLogic Hook for [Logrus](https://github.com/sirupsen/logrus) <img src="http://i.imgur.com/hTeVwmJ.png" width="40" height="40" alt=":walrus:" class="emoji" title=":walrus:"/>

## Description
sumologrus is async Logrus hook. Logs are flushed to Sumologic periodically (5 sec), if size of the batch reaches 250 logs or by explicitly calling Flush method


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
	sumoLogicHook := sumologrus.NewSumoLogicHook(endpoint, host, logrus.InfoLevel, "tag1", "tag2")
	defer sumoLogicHook.Flush()
	log.Hooks.Add(sumoLogicHook)

	log.WithFields(logrus.Fields{
		"name": "hurley",
		"age":  29,
	}).Error("Hello world!")
}
```