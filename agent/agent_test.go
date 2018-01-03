package agent

import (
	"fmt"
	"testing"
	"time"

	"github.com/jpra1113/snap-plugin-lib-go/v1/plugin"
	. "github.com/smartystreets/goconvey/convey"
)

func TestProcessor(t *testing.T) {
	Convey("Test Processor", t, func() {
		Convey("Test Process", func() {
			p := NewProcessor()
			cfg := plugin.Config{
				"collect.namespaces":              "default, hyperpilot",
				"collect.include_empty_namespace": true,
				"collect.exclude_metrics":         "intel/docker/spec/*, intel/procfs/*, intel/docker/stats/*",
				"collect.exclude_metrics.except":  "*perc, *percentage",
				"average.exclude_metrics":         "*prec, *precentage",
			}

			// in, out, out, in, in, out, out, out, out
			mts := []plugin.Metric{
				plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "docker", "spec", "perc"),
					Config:    map[string]interface{}{"pw": "123aB"},
					Data:      789,
					Tags:      map[string]string{"io.kubernetes.pod.namespace": "hyperpilot"},
					Unit:      "int",
					Timestamp: time.Now(),
				},
				plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "docker", "spec", "hmm"),
					Config:    map[string]interface{}{"pw": "123aB"},
					Data:      789,
					Unit:      "int",
					Timestamp: time.Now(),
				},
				plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "procfs", "cpu", "guest_nice"),
					Config:    map[string]interface{}{"pw": "123aB"},
					Data:      789,
					Unit:      "int",
					Timestamp: time.Now(),
				},
				plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "procfs", "cpu", "guest_nice_percentage"),
					Config:    map[string]interface{}{"pw": "123aB"},
					Data:      789,
					Unit:      "int",
					Timestamp: time.Now(),
				},
				plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "docker", "stats", "cgroups", "cpu_stats", "percentage"),
					Config:    map[string]interface{}{"pw": "123aB"},
					Data:      123,
					Tags:      map[string]string{"io.kubernetes.pod.namespace": "default"},
					Unit:      "int",
					Timestamp: time.Now(),
				},
				plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "docker", "stats", "cgroups", "cpu_stats", "cpu_shares"),
					Config:    map[string]interface{}{"pw": "123aB"},
					Data:      123,
					Tags:      map[string]string{"io.kubernetes.pod.namespace": "default"},
					Unit:      "int",
					Timestamp: time.Now(),
				},
				plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "docker", "spec", "size_root"),
					Config:    map[string]interface{}{"pw": "123aB"},
					Data:      456,
					Tags:      map[string]string{"io.kubernetes.pod.namespace": "default"},
					Unit:      "int",
					Timestamp: time.Now(),
				},
				plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "docker", "spec", "size_rw"),
					Config:    map[string]interface{}{"pw": "123aB"},
					Data:      789,
					Tags:      map[string]string{"io.kubernetes.pod.namespace": "hyperpilot"},
					Unit:      "int",
					Timestamp: time.Now(),
				},
				plugin.Metric{
					Namespace: plugin.NewNamespace("intel", "docker", "spec", "size_rw"),
					Config:    map[string]interface{}{"pw": "123aB"},
					Data:      789,
					Tags:      map[string]string{"io.kubernetes.pod.namespace": "haha"},
					Unit:      "int",
					Timestamp: time.Now(),
				},
			}
			result, err := p.Process(mts, cfg)

			for _, item := range result {
				fmt.Println(item.Namespace)
			}

			Convey("Should only process 1 data", func() {
				So(len(result), ShouldEqual, 3)
			})
			Convey("No error returned", func() {
				So(err, ShouldBeNil)
			})

		})
	})
}
