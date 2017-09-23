package agent

import (
	"errors"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aasssddd/snap-plugin-lib-go/v1/plugin"
	"github.com/gobwas/glob"
	logging "github.com/op/go-logging"
)

var log = logging.MustGetLogger("processor")
var logFormatter = logging.MustStringFormatter(
	` %{level:.1s}%{time:0102 15:04:05.999999} %{pid} %{shortfile}] %{message}`,
)

type FileLog struct {
	Name    string
	Logger  *logging.Logger
	LogFile *os.File
}

type PreviousData struct {
	Data   float64
	Create time.Time
}

// Processor test processor
type SnapProcessor struct {
	Cache map[string]PreviousData
	Log   *FileLog
}

// NewProcessor generate processor
func NewProcessor() plugin.Processor {
	return &SnapProcessor{
		Cache: make(map[string]PreviousData),
	}
}

func NewLogger(filesPath string, name string) (*FileLog, error) {
	logDirPath := path.Join(filesPath, "log")
	if _, err := os.Stat(logDirPath); os.IsNotExist(err) {
		os.Mkdir(logDirPath, 0777)
	}

	logFilePath := path.Join(logDirPath, name+".log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, errors.New("Unable to create log file:" + err.Error())
	}

	fileLog := logging.NewLogBackend(logFile, "["+name+"]", 0)
	fileLogLevel := logging.AddModuleLevel(fileLog)
	fileLogLevel.SetLevel(logging.ERROR, "")
	fileLogBackend := logging.NewBackendFormatter(fileLog, logFormatter)

	log.SetBackend(logging.SetBackend(fileLogBackend))

	return &FileLog{
		Name:    name,
		Logger:  log,
		LogFile: logFile,
	}, nil
}

// Process test process function
func (p *SnapProcessor) Process(mts []plugin.Metric, cfg plugin.Config) ([]plugin.Metric, error) {
	if p.Log == nil {
		processLog, err := NewLogger("/tmp", "processor")
		if err != nil {
			return mts, errors.New("Error creating process logger: " + err.Error())
		}
		p.Log = processLog
	}

	log := p.Log.Logger
	log.Infof("Process received metric size: %d", len(mts))
	namespacesConfig, err := cfg.GetString("collect.namespaces")
	if err != nil {
		return mts, errors.New("Unable to read namespaces config: " + err.Error())
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

	exceptsList := strings.Split(strings.Replace(excepts, " ", "", -1), ",")

	average, err := cfg.GetString("average")
	if err != nil {
		average = ""
	}
	averageList := strings.Split(strings.Replace(average, " ", "", -1), ",")

	// processNamespaces = append(processNamespaces, "")
	log.Infof("Process namespaces: %+v", processNamespaces)
	excludeMetricsConfig, err := cfg.GetString("collect.exclude_metrics")

	if err != nil {
		return mts, errors.New("Unable to read filterMetricKeywords config: " + err.Error())
	}
	excludeKeywordsList := strings.Split(strings.Replace(excludeMetricsConfig, " ", "", -1), ",")
	log.Infof("Process filterMetricKeywords: %+v", excludeKeywordsList)

	metrics := []plugin.Metric{}
	for _, mt := range mts {
		podNamespace, _ := mt.Tags["io.kubernetes.pod.namespace"]
		if (isEmptyNamespaceInclude && podNamespace == "") || inArray(podNamespace, processNamespaces) {
			if !isKeywordMatch(strings.Join(mt.Namespace.Strings(), "/"), excludeKeywordsList) ||
				isKeywordMatch(strings.Join(mt.Namespace.Strings(), "/"), exceptsList) {
				if isKeywordMatch(strings.Join(mt.Namespace.Strings(), "/"), averageList) {
					mt.Data = p.caluAverageData(mt, log)
				}
				metrics = append(metrics, mt)
			}
		}
	}

	log.Infof("Process filter metric size %d: ", len(metrics))
	// log.Infof("Process filter metric %+v: ", metrics)
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

func (p *SnapProcessor) caluAverageData(
	mt plugin.Metric,
	log *logging.Logger) float64 {
	namespaces := mt.Namespace.Strings()
	mapKey := strings.Join(namespaces, "/")
	averageData := float64(-1)
	previousData, ok := p.Cache[mapKey]
	if ok {
		log.Infof("Find %s previous cache metric vaule: %+v", mapKey, previousData)
		diffSeconds := mt.Timestamp.Sub(previousData.Create).Seconds()
		diffValue := (convertInterface(mt.Data) - previousData.Data)
		if diffSeconds > 0 && diffValue > 0 {
			averageData = (convertInterface(mt.Data) - previousData.Data) / diffSeconds
			log.Infof("Calculate %s averageData(%f) on %s", mapKey, averageData, mt.Timestamp)
		}
	}

	p.Cache[mapKey] = PreviousData{
		Data:   convertInterface(mt.Data),
		Create: mt.Timestamp,
	}
	log.Infof("Cache this time metric vaule: %+v", p.Cache[mapKey])
	return averageData
}

func isKeywordMatch(keyword string, patterns []string) bool {
	isMatched := false
	for _, pattern := range patterns {
		g := glob.MustCompile(pattern)
		isMatched = isMatched || g.Match(keyword)
	}
	return isMatched

}

func convertInterface(data interface{}) float64 {
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
