# go-gin-prometheus
[![Go Reference](https://pkg.go.dev/badge/github.com/spechtlabs/go-gin-prometheus.svg)](https://pkg.go.dev/github.com/spechtlabs/go-gin-prometheus) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Gin Web Framework Prometheus metrics exporter

_Forked from [zsais/go-gin-prometheus](https://github.com/zsais/go-gin-prometheus/)_

Modified to use a more modern builder pattern to pass in configurations 

## Installation

```bash
go get github.com/spechtlabs/go-gin-prometheus
```

## Usage

```go
package main

import (
	"github.com/gin-gonic/gin"
	ginprometheus "github.com/spechtlabs/go-gin-prometheus"
)

func main() {
	r := gin.New()

	r.Use(ginprometheus.GinPrometheusMiddleware(r, "gin"))

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, "Hello world!")
	})

	r.Run(":29090")
}
```

See the [example.go file](example/example.go)

## Preserving a low cardinality for the request counter

The request counter (`requests_total`) has a `url` label which,
although desirable, can become problematic in cases where your
application uses templated routes expecting a great number of
variations, as Prometheus explicitly recommends against metrics having
high cardinality dimensions:

https://prometheus.io/docs/practices/naming/#labels

If you have for instance a `/customer/:name` templated route and you
don't want to generate a time series for every possible customer name,
you could supply this mapping function to the middleware:

```go
package main

import (
	"strings"

	"github.com/gin-gonic/gin"
	ginprometheus "github.com/spechtlabs/go-gin-prometheus"
)

func main() {
	r := gin.New()

	mapURL := func(c *gin.Context) string {
		url := c.Request.URL.Path
		for _, p := range c.Params {
			if p.Key == "name" {
				url = strings.Replace(url, p.Value, ":name", 1)
				break
			}
		}
		return url
	}

	r.Use(ginprometheus.GinPrometheusMiddleware(r, "gin",
		ginprometheus.WithRequestCounterURLLabelMappingFn(mapURL),
	))

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, "Hello world!")
	})

	r.Run(":29090")
}
```

which would map `/customer/alice` and `/customer/bob` to their
template `/customer/:name`, and thus preserve a low cardinality for
our metrics.

To replace every route parameter with its name, use
`ginprometheus.WithLowCardinalityUrl()` instead of writing the mapping
function yourself.
