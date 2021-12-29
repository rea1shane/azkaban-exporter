package azkaban

import (
	"context"
	"github.com/go-kratos/kratos/pkg/sync/errgroup"
	"github.com/morikuni/failure"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rea1shane/basexporter/required/functions"
	"github.com/rea1shane/basexporter/required/structs"
	"github.com/rea1shane/basexporter/util"
	log "github.com/sirupsen/logrus"
	"time"
)

var (
	runningExecs          []int
	totalSucceededCounter = cmap.New()
	totalFailedCounter    = cmap.New()
	totalKilledCounter    = cmap.New()
)

const (
	subsystem  = "flow"
	startIndex = 0
	listLength = 1
)

func init() {
	functions.RegisterCollector(subsystem, util.DefaultEnabled, NewAzkabanCollector)
}

type azkabanCollector struct {
	logger          *log.Entry
	new             util.TypedDesc
	preparing       util.TypedDesc
	running         util.TypedDesc
	succeeded       util.TypedDesc
	failed          util.TypedDesc
	unknow          util.TypedDesc
	killed          util.TypedDesc
	running0        util.TypedDesc
	running60       util.TypedDesc
	running300      util.TypedDesc
	running1440     util.TypedDesc
	runningDuration util.TypedDesc
	totalSucceeded  util.TypedDesc
	totalFailed     util.TypedDesc
	totalKilled     util.TypedDesc
	lastStatus      util.TypedDesc
	lastDuration    util.TypedDesc
}

func NewAzkabanCollector(namespace string, logger *log.Entry) (structs.Collector, error) {
	var (
		labelProject     = []string{"project"}
		labelProjectFlow = []string{"project", "flow"}
	)

	return &azkabanCollector{
		logger: logger,
		new: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "new"),
				"The number of never run flows", labelProject, nil),
			ValueType: prometheus.GaugeValue,
		},
		preparing: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "preparing"),
				"The number of preparing start flows", labelProject, nil),
			ValueType: prometheus.GaugeValue,
		},
		running: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "running"),
				"The number of running flows.", labelProject, nil),
			ValueType: prometheus.GaugeValue,
		},
		succeeded: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "succeeded"),
				"The number of flows that last status is succeeded.", labelProject, nil),
			ValueType: prometheus.GaugeValue,
		},
		failed: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "failed"),
				"The number of flows that last status is failed.", labelProject, nil),
			ValueType: prometheus.GaugeValue,
		},
		unknow: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "unknow"),
				"The number of flows that last status is unknow.", labelProject, nil),
			ValueType: prometheus.GaugeValue,
		},
		killed: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "killed"),
				"The number of flows that last status is killed.", labelProject, nil),
			ValueType: prometheus.GaugeValue,
		},
		running0: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "running_0"),
				"The number of running flows that duration in [0, 60) mins.", labelProject, nil),
			ValueType: prometheus.GaugeValue,
		},
		running60: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "running_60"),
				"The number of running flows that duration in [60, 300) mins.", labelProject, nil),
			ValueType: prometheus.GaugeValue,
		},
		running300: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "running_300"),
				"The number of running flows that duration in [300, 1440) mins.", labelProject, nil),
			ValueType: prometheus.GaugeValue,
		},
		running1440: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "running_1440"),
				"The number of running flows that duration over 1440 mins.", labelProject, nil),
			ValueType: prometheus.GaugeValue,
		},
		totalSucceeded: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "succeeded_total"),
				"The total number of succeeded.", labelProject, nil),
			ValueType: prometheus.CounterValue,
		},
		totalFailed: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "failed_total"),
				"The total number of failed.", labelProject, nil),
			ValueType: prometheus.CounterValue,
		},
		totalKilled: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "killed_total"),
				"The total number of killed.", labelProject, nil),
			ValueType: prometheus.CounterValue,
		},
		runningDuration: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "running_duration_ms"),
				"Duration of each running flows. (unit: ms)", labelProjectFlow, nil),
			ValueType: prometheus.GaugeValue,
		},
		lastStatus: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "last_status"),
				"Flow last execution status flag. (-2=KILLED / -1=UNKNOW / 0=FAILED / 1=SUCCEEDED / 2=RUNNING / 3=PREPARING / 4=NEW)", labelProjectFlow, nil),
			ValueType: prometheus.GaugeValue,
		},
		lastDuration: util.TypedDesc{
			Desc: prometheus.NewDesc(prometheus.BuildFQName(namespace, subsystem, "last_duration_ms"),
				"Flow last execution duration which finished. (unit: ms)", labelProjectFlow, nil),
			ValueType: prometheus.GaugeValue,
		},
	}, nil
}

func (c azkabanCollector) Update(ch chan<- prometheus.Metric) error {
	var (
		azkaban = GetAzkaban()

		projectsWithFlows = make(chan ProjectWithFlows)
		executions        = make(chan Execution)

		newCounter       = cmap.New()
		preparingCounter = cmap.New()
		runningCounter   = cmap.New()
		succeededCounter = cmap.New()
		failedCounter    = cmap.New()
		unknowCounter    = cmap.New()
		killedCounter    = cmap.New()

		running0Counter       = cmap.New()
		running60Counter      = cmap.New()
		running300Counter     = cmap.New()
		running1440Counter    = cmap.New()
		runningAttemptCounter = cmap.New()

		runningDurationRecorder = cmap.New()
		lastStatusRecorder      = cmap.New()
		lastDurationRecorder    = cmap.New()
	)
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5000*time.Millisecond)
	defer cancelFunc()
	group := errgroup.WithCancel(ctx)
	group.Go(func(ctx context.Context) error {
		err := azkaban.GetProjectWithFlows(ctx, projectsWithFlows)
		close(projectsWithFlows)
		return err
	})
	group.Go(func(ctx context.Context) error {
		var projectNames []string
		g := errgroup.WithCancel(ctx)
		for projectWithFlows := range projectsWithFlows {
			projectName := projectWithFlows.ProjectName
			flowIds := projectWithFlows.FlowIds
			projectNames = append(projectNames, projectName)

			newCounter.Set(projectName, 0)
			preparingCounter.Set(projectName, 0)
			runningCounter.Set(projectName, 0)
			succeededCounter.Set(projectName, 0)
			failedCounter.Set(projectName, 0)
			unknowCounter.Set(projectName, 0)
			killedCounter.Set(projectName, 0)

			running0Counter.Set(projectName, 0)
			running60Counter.Set(projectName, 0)
			running300Counter.Set(projectName, 0)
			running1440Counter.Set(projectName, 0)
			runningAttemptCounter.Set(projectName, 0)

			runningDurationRecorder.Set(projectName, cmap.New())
			lastStatusRecorder.Set(projectName, cmap.New())
			lastDurationRecorder.Set(projectName, cmap.New())

			totalSucceededCounter.SetIfAbsent(projectName, 0)
			totalFailedCounter.SetIfAbsent(projectName, 0)
			totalKilledCounter.SetIfAbsent(projectName, 0)
			g.Go(func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					for _, flowId := range flowIds {
						fid := flowId
						g.Go(func(ctx context.Context) error {
							select {
							case <-ctx.Done():
								return ctx.Err()
							default:
								return azkaban.GetExecutions(ctx, projectName, fid, startIndex, listLength, executions)
							}
						})
					}
					return nil
				}
			})
		}
		err := g.Wait()
		close(executions)
		removeKeys(totalSucceededCounter, projectNames)
		removeKeys(totalFailedCounter, projectNames)
		removeKeys(totalKilledCounter, projectNames)
		return err
	})
	for execution := range executions {
		switch execution.Status {
		case "NEW":
			lastStatusSecondMap, _ := lastStatusRecorder.Get(execution.ProjectName)
			lastStatusSecondMap.(cmap.ConcurrentMap).Set(execution.FlowID, 4)
			n, _ := newCounter.Get(execution.ProjectName)
			newCounter.Set(execution.ProjectName, n.(int)+1)
		case "PREPARING":
			lastStatusSecondMap, _ := lastStatusRecorder.Get(execution.ProjectName)
			lastStatusSecondMap.(cmap.ConcurrentMap).Set(execution.FlowID, 3)
			p, _ := preparingCounter.Get(execution.ProjectName)
			preparingCounter.Set(execution.ProjectName, p.(int)+1)
		case "RUNNING":
			if _, ok := findInt(runningExecs, execution.ExecID); !ok {
				runningExecs = append(runningExecs, execution.ExecID)
			}
			lastStatusSecondMap, _ := lastStatusRecorder.Get(execution.ProjectName)
			lastStatusSecondMap.(cmap.ConcurrentMap).Set(execution.FlowID, 2)
			runningTime := time.Now().UnixMilli() - execution.StartTime
			runningDurationSecondMap, _ := runningDurationRecorder.Get(execution.ProjectName)
			runningDurationSecondMap.(cmap.ConcurrentMap).Set(execution.FlowID, runningTime)
			if inRange(runningTime, 0, 3600000) {
				r0, _ := running0Counter.Get(execution.ProjectName)
				running0Counter.Set(execution.ProjectName, r0.(int)+1)
			} else if inRange(runningTime, 3600000, 18000000) {
				r60, _ := running60Counter.Get(execution.ProjectName)
				running60Counter.Set(execution.ProjectName, r60.(int)+1)
			} else if inRange(runningTime, 18000000, 86400000) {
				r300, _ := running300Counter.Get(execution.ProjectName)
				running300Counter.Set(execution.ProjectName, r300.(int)+1)
			} else {
				r1440, _ := running1440Counter.Get(execution.ProjectName)
				running1440Counter.Set(execution.ProjectName, r1440.(int)+1)
			}
			r, _ := runningCounter.Get(execution.ProjectName)
			runningCounter.Set(execution.ProjectName, r.(int)+1)
		case "SUCCEEDED":
			if index, ok := findInt(runningExecs, execution.ExecID); ok {
				value, _ := totalSucceededCounter.Get(execution.ProjectName)
				totalSucceededCounter.Set(execution.ProjectName, value.(int)+1)
				runningExecs = append(runningExecs[:index], runningExecs[index+1:]...)
			}
			lastStatusSecondMap, _ := lastStatusRecorder.Get(execution.ProjectName)
			lastStatusSecondMap.(cmap.ConcurrentMap).Set(execution.FlowID, 1)
			lastDurationSecondMap, _ := lastDurationRecorder.Get(execution.ProjectName)
			lastDurationSecondMap.(cmap.ConcurrentMap).Set(execution.FlowID, execution.EndTime-execution.StartTime)
			s, _ := succeededCounter.Get(execution.ProjectName)
			succeededCounter.Set(execution.ProjectName, s.(int)+1)
		case "FAILED":
			if index, ok := findInt(runningExecs, execution.ExecID); ok {
				value, _ := totalFailedCounter.Get(execution.ProjectName)
				totalFailedCounter.Set(execution.ProjectName, value.(int)+1)
				runningExecs = append(runningExecs[:index], runningExecs[index+1:]...)
			}
			lastStatusSecondMap, _ := lastStatusRecorder.Get(execution.ProjectName)
			lastStatusSecondMap.(cmap.ConcurrentMap).Set(execution.FlowID, 0)
			lastDurationSecondMap, _ := lastDurationRecorder.Get(execution.ProjectName)
			lastDurationSecondMap.(cmap.ConcurrentMap).Set(execution.FlowID, execution.EndTime-execution.StartTime)
			f, _ := failedCounter.Get(execution.ProjectName)
			failedCounter.Set(execution.ProjectName, f.(int)+1)
		case "KILLED":
			if index, ok := findInt(runningExecs, execution.ExecID); ok {
				value, _ := totalKilledCounter.Get(execution.ProjectName)
				totalKilledCounter.Set(execution.ProjectName, value.(int)+1)
				runningExecs = append(runningExecs[:index], runningExecs[index+1:]...)
			}
			lastStatusSecondMap, _ := lastStatusRecorder.Get(execution.ProjectName)
			lastStatusSecondMap.(cmap.ConcurrentMap).Set(execution.FlowID, -2)
			lastDurationSecondMap, _ := lastDurationRecorder.Get(execution.ProjectName)
			lastDurationSecondMap.(cmap.ConcurrentMap).Set(execution.FlowID, execution.EndTime-execution.StartTime)
			k, _ := killedCounter.Get(execution.ProjectName)
			killedCounter.Set(execution.ProjectName, k.(int)+1)
		default:
			lastStatusSecondMap, _ := lastStatusRecorder.Get(execution.ProjectName)
			lastStatusSecondMap.(cmap.ConcurrentMap).Set(execution.FlowID, -1)
			u, _ := unknowCounter.Get(execution.ProjectName)
			unknowCounter.Set(execution.ProjectName, u.(int)+1)
		}
	}
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			newCounter.IterCb(func(projectName string, v interface{}) {
				ch <- c.new.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			preparingCounter.IterCb(func(projectName string, v interface{}) {
				ch <- c.preparing.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			runningCounter.IterCb(func(projectName string, v interface{}) {
				ch <- c.running.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			succeededCounter.IterCb(func(projectName string, v interface{}) {
				ch <- c.succeeded.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			failedCounter.IterCb(func(projectName string, v interface{}) {
				ch <- c.failed.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			unknowCounter.IterCb(func(projectName string, v interface{}) {
				ch <- c.unknow.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			killedCounter.IterCb(func(projectName string, v interface{}) {
				ch <- c.killed.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			running0Counter.IterCb(func(projectName string, v interface{}) {
				ch <- c.running0.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			running60Counter.IterCb(func(projectName string, v interface{}) {
				ch <- c.running60.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			running300Counter.IterCb(func(projectName string, v interface{}) {
				ch <- c.running300.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			running1440Counter.IterCb(func(projectName string, v interface{}) {
				ch <- c.running1440.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			totalSucceededCounter.IterCb(func(projectName string, v interface{}) {
				ch <- c.totalSucceeded.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			totalFailedCounter.IterCb(func(projectName string, v interface{}) {
				ch <- c.totalFailed.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			totalKilledCounter.IterCb(func(projectName string, v interface{}) {
				ch <- c.totalKilled.MustNewConstMetric(float64(v.(int)), projectName)
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			runningDurationRecorder.IterCb(func(projectName string, secondMap interface{}) {
				secondMap.(cmap.ConcurrentMap).IterCb(func(flowId string, v interface{}) {
					ch <- c.runningDuration.MustNewConstMetric(float64(v.(int64)), projectName, flowId)
				})
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			lastStatusRecorder.IterCb(func(projectName string, secondMap interface{}) {
				secondMap.(cmap.ConcurrentMap).IterCb(func(flowId string, v interface{}) {
					ch <- c.lastStatus.MustNewConstMetric(float64(v.(int)), projectName, flowId)
				})
			})
			return nil
		}
	})
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			lastDurationRecorder.IterCb(func(projectName string, secondMap interface{}) {
				secondMap.(cmap.ConcurrentMap).IterCb(func(flowId string, v interface{}) {
					ch <- c.lastDuration.MustNewConstMetric(float64(v.(int64)), projectName, flowId)
				})
			})
			return nil
		}
	})
	err := group.Wait()
	if ctx.Err() != nil {
		return failure.Wrap(ctx.Err())
	}
	return err
}

// inRange determine whether a number belongs to a range.
// Will determine target number in [start number, end number)
func inRange(target int64, start int64, end int64) bool {
	if end <= start {
		panic("Wrong value of arguments.")
	} else {
		return target >= start && target < end
	}
}

func findInt(slice []int, val int) (int, bool) {
	for index, item := range slice {
		if item == val {
			return index, true
		}
	}
	return -1, false
}

func findString(slice []string, val string) (int, bool) {
	for index, item := range slice {
		if item == val {
			return index, true
		}
	}
	return -1, false
}

// removeKeys if key not in slice, delete.
func removeKeys(m cmap.ConcurrentMap, s []string) {
	keys := m.Keys()
	for _, key := range keys {
		if _, ok := findString(s, key); !ok {
			m.Remove(key)
		}
	}
}
