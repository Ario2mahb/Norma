package accountmon

import (
	"math"
	"sort"
	"testing"
	"time"

	"github.com/Fantom-foundation/Norma/driver"
	mon "github.com/Fantom-foundation/Norma/driver/monitoring"
	"github.com/Fantom-foundation/Norma/driver/monitoring/utils"
	"github.com/golang/mock/gomock"
	"golang.org/x/exp/slices"
)

// Unfortunatley, gomock can not (yet) create mocks for generic interfaces.
// So we need to write our own fake sensors for this test.

var (
	testAccountMetric = mon.Metric[mon.Account, mon.Series[mon.Time, int]]{
		Name:        "TestAccountMetric",
		Description: "A test metric for this unit test.",
	}
)

type testSensor struct {
	next int
}

func (s *testSensor) ReadValue() (int, error) {
	s.next++
	return s.next, nil
}

type testSensorFactory struct{}

func (f *testSensorFactory) CreateSensor(driver.Application, int) (utils.Sensor[int], error) {
	return &testSensor{}, nil
}

func TestAppSourceRetrievesSensorData(t *testing.T) {
	ctrl := gomock.NewController(t)
	net := driver.NewMockNetwork(ctrl)

	config1 := driver.ApplicationConfig{Name: "A", Accounts: 2}
	config2 := driver.ApplicationConfig{Name: "B", Accounts: 1}
	config3 := driver.ApplicationConfig{Name: "C", Accounts: 3}

	app1 := driver.NewMockApplication(ctrl)
	app2 := driver.NewMockApplication(ctrl)
	app3 := driver.NewMockApplication(ctrl)

	app1.EXPECT().Config().AnyTimes().Return(&config1)
	app2.EXPECT().Config().AnyTimes().Return(&config2)
	app3.EXPECT().Config().AnyTimes().Return(&config3)

	net.EXPECT().RegisterListener(gomock.Any()).AnyTimes()
	net.EXPECT().UnregisterListener(gomock.Any()).AnyTimes()
	net.EXPECT().GetActiveNodes().Return([]driver.Node{}).AnyTimes() // Because of the monitors default log consumer
	net.EXPECT().GetActiveApplications().Return([]driver.Application{app1, app2}).AnyTimes()

	monitor, err := mon.NewMonitor(net, mon.MonitorConfig{OutputDir: t.TempDir()})
	if err != nil {
		t.Fatalf("failed to start monitor instance: %v", err)
	}
	source := newPeriodicAccountDataSource[int](testAccountMetric, monitor, 50*time.Millisecond, &testSensorFactory{})

	// Check that existing apps are tracked.
	subjects := source.GetSubjects()
	sort.Slice(subjects, func(i, j int) bool { return subjects[i].Less(&subjects[j]) })
	want := []mon.Account{
		{App: mon.App("A"), Id: 0},
		{App: mon.App("A"), Id: 1},
		{App: mon.App("B"), Id: 0},
	}
	if !slices.Equal(subjects, want) {
		t.Errorf("invalid list of subjects, wanted %v, got %v", want, subjects)
	}

	// Simulate the creation of a node after source initialization.
	source.(driver.NetworkListener).AfterApplicationCreation(app3)

	// Check that subject list has updated.
	subjects = source.GetSubjects()
	sort.Slice(subjects, func(i, j int) bool { return subjects[i].Less(&subjects[j]) })
	want = append(want, []mon.Account{
		{App: mon.App("C"), Id: 0},
		{App: mon.App("C"), Id: 1},
		{App: mon.App("C"), Id: 2},
	}...)
	if !slices.Equal(subjects, want) {
		t.Errorf("invalid list of subjects, wanted %v, got %v", want, subjects)
	}

	time.Sleep(200 * time.Millisecond)
	if err := source.Shutdown(); err != nil {
		t.Errorf("erros encountered during shutdown: %v", err)
	}

	// Check that subjects are still all there.
	subjects = source.GetSubjects()
	sort.Slice(subjects, func(i, j int) bool { return subjects[i].Less(&subjects[j]) })
	if !slices.Equal(subjects, want) {
		t.Errorf("invalid list of subjects, wanted %v, got %v", want, subjects)
	}

	for _, subject := range subjects {
		data, exists := source.GetData(subject)
		if data == nil || !exists {
			t.Errorf("no data found for node %v", subject)
			continue
		}
		subrange := data.GetRange(mon.Time(0), mon.Time(math.MaxInt64))
		if len(subrange) == 0 {
			t.Errorf("no data collected for node %v", subject)
		}
		for i, point := range subrange {
			if got, want := point.Value, i+1; got != want {
				t.Errorf("unexpected value collected for subject %v: wanted %d, got %d", subject, want, got)
			}
		}
	}
}