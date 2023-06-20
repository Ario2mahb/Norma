package utils

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Fantom-foundation/Norma/driver/monitoring"
	"github.com/Fantom-foundation/Norma/driver/monitoring/export"
	"golang.org/x/exp/constraints"
)

// Sensor is an abstraction of some input device capable of probing a node
// for some metric of type T.
type Sensor[T any] interface {
	ReadValue() (T, error)
}

// PeriodicDataSource is a generic data source periodically querying
// node-associated sensors for data.
type PeriodicDataSource[S comparable, T any] struct {
	metric   monitoring.Metric[S, monitoring.Series[monitoring.Time, T]]
	period   time.Duration
	data     map[S]*monitoring.SyncedSeries[monitoring.Time, T]
	dataLock sync.Mutex
	stop     chan bool  // used to signal per-node collectors about the shutdown
	done     chan error // used to signal collector shutdown to source
}

// NewPeriodicDataSource creates a new data source managing per-node sensor
// instances for a given metric and periodically collecting data from those.
func NewPeriodicDataSource[S constraints.Ordered, T any](
	metric monitoring.Metric[S, monitoring.Series[monitoring.Time, T]],
	monitor *monitoring.Monitor,
) *PeriodicDataSource[S, T] {
	return NewPeriodicDataSourceWithPeriod(metric, monitor, time.Second)
}

// NewPeriodicDataSourceWithPeriod is the same as NewPeriodicDataSource but with
// a customizable sampling periode.
func NewPeriodicDataSourceWithPeriod[S constraints.Ordered, T any](
	metric monitoring.Metric[S, monitoring.Series[monitoring.Time, T]],
	monitor *monitoring.Monitor,
	period time.Duration,
) *PeriodicDataSource[S, T] {
	stop := make(chan bool)
	done := make(chan error)

	res := &PeriodicDataSource[S, T]{
		metric: metric,
		period: period,
		data:   map[S]*monitoring.SyncedSeries[monitoring.Time, T]{},
		stop:   stop,
		done:   done,
	}

	monitor.Writer().Add(func() error {
		source := monitoring.Source[S, monitoring.Series[monitoring.Time, T]](res)
		return export.AddSeriesData(monitor.Writer(), source)
	})

	return res
}

func (s *PeriodicDataSource[S, T]) GetMetric() monitoring.Metric[S, monitoring.Series[monitoring.Time, T]] {
	return s.metric
}

func (s *PeriodicDataSource[S, T]) GetSubjects() []S {
	s.dataLock.Lock()
	defer s.dataLock.Unlock()
	res := make([]S, 0, len(s.data))
	for subject := range s.data {
		res = append(res, subject)
	}
	return res
}

func (s *PeriodicDataSource[S, T]) GetData(subject S) (monitoring.Series[monitoring.Time, T], bool) {
	s.dataLock.Lock()
	defer s.dataLock.Unlock()
	res, exists := s.data[subject]
	return res, exists
}

func (s *PeriodicDataSource[S, T]) Shutdown() error {
	close(s.stop)
	<-s.done
	return nil
}

func (s *PeriodicDataSource[S, T]) AddSubject(subject S, sensor Sensor[T]) error {
	// Register a new data series if the subject is new.
	s.dataLock.Lock()
	data := &monitoring.SyncedSeries[monitoring.Time, T]{}
	if _, exist := s.data[subject]; exist {
		s.dataLock.Unlock()
		return fmt.Errorf("sensor for subject %v already present", subject)
	}
	s.data[subject] = data
	s.dataLock.Unlock()

	// Start background routine collecting sensor data.
	go func() {
		var err error
		defer func() {
			s.done <- err
		}()

		var errs []error
		ticker := time.NewTicker(s.period)
		for {
			select {
			case now := <-ticker.C:
				value, err := sensor.ReadValue()
				if err != nil {
					errs = append(errs, err)
				} else {
					data.Append(monitoring.NewTime(now), value)
				}
			case <-s.stop:
				err = errors.Join(errs...)
				return
			}
		}
	}()

	return nil
}