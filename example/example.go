package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	ginprometheus "github.com/spechtlabs/go-gin-prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func main() {
	r := gin.New()

	// Optional
	testMetric := &ginprometheus.Metric{
		ID:          "1234",                          // optional string
		Name:        "test_metric",                   // required string
		Description: "Counter test metric",           // required string
		Type:        ginprometheus.MetricTypeCounter, // required string
	}

	testMetric2 := &ginprometheus.Metric{
		ID:          "1235",                          // Identifier
		Name:        "test_metric_2",                 // Metric Name
		Description: "Summary test metric",           // Help Description
		Type:        ginprometheus.MetricTypeSummary, // type associated with prometheus collector
	}

	r.Use(
		ginprometheus.GinPrometheusMiddleware(r, "gin",
			ginprometheus.WithCustomMetric(testMetric),     // Optional: additional custom metrics
			ginprometheus.WithCustomMetric(testMetric2),    // Optional: additional custom metric
			ginprometheus.WithRegisterer(metrics.Registry), // Optional: use the K8s controller metrics registry
			ginprometheus.WithLowCardinalityUrl(),          // Optional: replace the url parameters with their keys to reduce the metric cardinality
		),
	)

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, "Hello world!")
	})

	r.GET("/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.JSON(200, fmt.Sprintf("Hello %s!", name))
	})

	r.GET("/:name/:surname", func(c *gin.Context) {
		name := c.Param("name")
		surname := c.Param("surname")
		c.JSON(200, fmt.Sprintf("Hello %s %s!", name, surname))
	})

	_ = r.Run(":29090")
}
