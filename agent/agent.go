package agent

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aasssddd/snap-plugin-lib-go/v1/plugin"
	"github.com/gobwas/glob"
)

var log *logrus.Entry

type PreviousData struct {
	Data   float64
	Create time.Time
}

type ProcessorConfig struct {
	ProcessNamespaces       []string
	ExceptsList             []glob.Glob
	ExcludeKeywordsList     []glob.Glob
	AverageList             []glob.Glob
	IsEmptyNamespaceInclude bool
}

func init() {
	log = logrus.New().WithField("processor", "average")
	log.Logger.Out = os.Stdout
}

func NewProcessorConfig(cfg plugin.Config) (*ProcessorConfig, error) {
	namespacesConfig, err := cfg.GetString("collect.namespaces")
	if err != nil {
		return nil, errors.New("Unable to read namespaces config: " + err.Error())
	}
	processNamespaces := strings.Split(strings.Replace(namespacesConfig, " ", "", -1), ",")

	isEmptyNamespaceInclude, err := cfg.GetBool("collect.include_empty_namespace")
	if err != nil {
		isEmptyNamespaceInclude = false
	}

	if isEmptyNamespaceInclude {
		processNamespaces = append(processNamespaces, "")
	}

	excepts, err := cfg.GetString("collect.exclude_metrics.except")
	if err != nil {
		excepts = ""
	}

	exceptsList := []glob.Glob{}
	for _, except := range strings.Split(strings.Replace(excepts, " ", "", -1), ",") {
		if except == "" {
			continue
		}

		g, err := glob.Compile(except)
		if err != nil {
			return nil, fmt.Errorf("Unable to compile pattern %s: %s", except, err.Error())
		}

		exceptsList = append(exceptsList, g)
	}
	log.Infof("Process excepts list: %s", excepts)

	averages, err := cfg.GetString("average")
	if err != nil {
		averages = ""
	}

	averageList := []glob.Glob{}
	for _, average := range strings.Split(strings.Replace(averages, " ", "", -1), ",") {
		if average == "" {
			continue
		}

		g, err := glob.Compile(average)
		if err != nil {
			return nil, fmt.Errorf("Unable to compile pattern %s: %s", average, err.Error())
		}

		averageList = append(averageList, g)
	}
	log.Infof("Process average list: %s", averages)

	log.Infof("Process namespaces: %+v", processNamespaces)
	excludeMetricsConfig, err := cfg.GetString("collect.exclude_metrics")
	if err != nil {
		excludeMetricsConfig = ""
	}

	excludeKeywordsList := []glob.Glob{}
	for _, exclude := range strings.Split(strings.Replace(excludeMetricsConfig, " ", "", -1), ",") {
		if exclude == "" {
			continue
		}

		g, err := glob.Compile(exclude)
		if err != nil {
			return nil, fmt.Errorf("Unable to compile pattern %s: %s", exclude, err.Error())
		}
		excludeKeywordsList = append(excludeKeywordsList, g)
	}
	log.Infof("Process exclude keywords list: %s", excludeMetricsConfig)

	return &ProcessorConfig{
		ProcessNamespaces:       processNamespaces,
		ExcludeKeywordsList:     excludeKeywordsList,
		ExceptsList:             exceptsList,
		AverageList:             averageList,
		IsEmptyNamespaceInclude: isEmptyNamespaceInclude,
	}, nil
}

// Processor test processor
type SnapProcessor struct {
	Cache map[string]*PreviousData
}

// NewProcessor generate processor
func NewProcessor() plugin.Processor {
	return &SnapProcessor{
		Cache: make(map[string]*PreviousData),
	}
}

func (p *SnapProcessor) isNamespacesCollected(config *ProcessorConfig, metricNamespace string, podNamespace string) bool {
	emptyNamespace := podNamespace == ""
	if config.IsEmptyNamespaceInclude && emptyNamespace {
		return true
	}

	needCollect := inArray(podNamespace, config.ProcessNamespaces)
	if !needCollect {
		log.Infof("%s\n is not in collect namespaces, Do not need to be average processing", metricNamespace)
	}

	return needCollect
}

func (p *SnapProcessor) isMetricNamespacesIncluded(config *ProcessorConfig, metricNamespace string) bool {
	if !isKeywordMatch(metricNamespace, config.ExcludeKeywordsList) || isKeywordMatch(metricNamespace, config.ExceptsList) {
		return true
	}

	log.Infof("%s\n is not in metric namespaces, Do not need to be average processing", metricNamespace)
	return false
}

func (p *SnapProcessor) isDataNull(data interface{}) bool {
	if data == nil {
		return true
	}

	return false
}

// Process test process function
func (p *SnapProcessor) Process(mts []plugin.Metric, cfg plugin.Config) ([]plugin.Metric, error) {
	config, err := NewProcessorConfig(cfg)
	if err != nil {
		return mts, errors.New("Unable to create processor config: " + err.Error())
	}

	metrics := []plugin.Metric{}
	for _, mt := range mts {
		metricNamespace := strings.Join(mt.Namespace.Strings(), "/")
		if p.isDataNull(mt.Data) {
			log.Errorf("Skipping average process for %s", metricNamespace)
			continue
		}

		podNamespace, _ := mt.Tags["io.kubernetes.pod.namespace"]
		if p.isNamespacesCollected(config, metricNamespace, podNamespace) && p.isMetricNamespacesIncluded(config, metricNamespace) {
			if isKeywordMatch(metricNamespace, config.AverageList) {
				mt.Data = p.CalculateAverageData(mt)
				mt.Tags["average_process"] = "true"
			}
			metrics = append(metrics, mt)
		}
	}

	return metrics, nil
}

/*
	GetConfigPolicy() returns the configPolicy for your plugin.
	A config policy is how users can provide configuration info to
	plugin. Here you define what sorts of config info your plugin
	needs and/or requires.
*/
func (p *SnapProcessor) GetConfigPolicy() (plugin.ConfigPolicy, error) {
	policy := plugin.NewConfigPolicy()
	return *policy, nil
}

func (p *SnapProcessor) CalculateAverageData(mt plugin.Metric) float64 {
	namespaces := mt.Namespace.Strings()
	mapKey := strings.Join(namespaces, "/")
	averageData := float64(0)
	previousData, ok := p.Cache[mapKey]
	if ok {
		log.Debugf("Find %s previous cache metric value: %+v", mapKey, previousData)
		diffSeconds := mt.Timestamp.Sub(previousData.Create).Seconds()
		diffValue := (convertInterface(mt.Data) - previousData.Data)
		if diffSeconds > 0 && diffValue > 0 {
			averageData = (convertInterface(mt.Data) - previousData.Data) / diffSeconds
			log.Debugf("Calculate %s averageData(%f) on %s", mapKey, averageData, mt.Timestamp)
		}
	} else {
		previousData = &PreviousData{}
		p.Cache[mapKey] = previousData
	}

	previousData.Data = convertInterface(mt.Data)
	previousData.Create = mt.Timestamp

	log.Debugf("Cache this time metric %s value: %+v", mapKey, previousData)
	return averageData
}

func isKeywordMatch(keyword string, patterns []glob.Glob) bool {
	for _, pattern := range patterns {
		if pattern.Match(keyword) {
			return true
		}
	}

	return false
}

func convertInterface(data interface{}) float64 {
	if data == nil {
		log.Errorf("Data is empty : Type %T", data)
		return float64(0)
	}

	switch data.(type) {
	case int:
		return float64(data.(int))
	case int8:
		return float64(data.(int8))
	case int16:
		return float64(data.(int16))
	case int32:
		return float64(data.(int32))
	case int64:
		return float64(data.(int64))
	case uint64:
		return float64(data.(uint64))
	case float32:
		return float64(data.(float32))
	case float64:
		return float64(data.(float64))
	default:
		return float64(0)
	}
}

func inArray(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
