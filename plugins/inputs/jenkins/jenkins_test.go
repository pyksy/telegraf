// Test Suite
package jenkins

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/testutil"
)

func TestJobRequest(t *testing.T) {
	tests := []struct {
		input         jobRequest
		hierarchyName string
		URL           string
	}{
		{
			jobRequest{},
			"",
			"",
		},
		{
			jobRequest{
				name:    "1",
				parents: []string{"3", "2"},
			},
			"3/2/1",
			"/job/3/job/2/job/1/api/json",
		},
		{
			jobRequest{
				name:    "job 3",
				parents: []string{"job 1", "job 2"},
			},
			"job 1/job 2/job 3",
			"/job/job%201/job/job%202/job/job%203/api/json",
		},
	}
	for _, test := range tests {
		hierarchyName := test.input.hierarchyName()
		address := test.input.url()
		if hierarchyName != test.hierarchyName {
			t.Errorf("Expected %s, got %s\n", test.hierarchyName, hierarchyName)
		}

		if test.URL != "" && address != test.URL {
			t.Errorf("Expected %s, got %s\n", test.URL, address)
		}
	}
}

func TestResultCode(t *testing.T) {
	tests := []struct {
		input  string
		output int
	}{
		{"SUCCESS", 0},
		{"Failure", 1},
		{"NOT_BUILT", 2},
		{"UNSTABLE", 3},
		{"ABORTED", 4},
	}
	for _, test := range tests {
		output := mapResultCode(test.input)
		if output != test.output {
			t.Errorf("Expected %d, got %d\n", test.output, output)
		}
	}
}

type mockHandler struct {
	// responseMap is the path to response interface
	// we will output the serialized response in json when serving http
	// example '/computer/api/json': *gojenkins.
	responseMap map[string]interface{}
}

func (h mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	o, ok := h.responseMap[r.URL.RequestURI()]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	b, err := json.Marshal(o)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(b) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Write(b) //nolint:errcheck // ignore the returned error as the tests will fail anyway
}

func TestGatherNodeData(t *testing.T) {
	tests := []struct {
		name    string
		input   mockHandler
		output  *testutil.Accumulator
		wantErr bool
	}{
		{
			name: "bad node data",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": struct{}{},
					"/computer/api/json": nodeResponse{
						Computers: []node{
							{},
							{},
							{},
						},
					},
				},
			},
			wantErr: true,
			output: &testutil.Accumulator{
				Metrics: []*testutil.Metric{
					{
						Tags: map[string]string{
							"source": "127.0.0.1",
						},
						Fields: map[string]interface{}{
							"busy_executors":  0,
							"total_executors": 0,
						},
					},
				},
			},
		},
		{
			name: "empty monitor data",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": struct{}{},
					"/computer/api/json": nodeResponse{
						Computers: []node{
							{DisplayName: "master"},
							{DisplayName: "node1"},
						},
					},
				},
			},
			output: &testutil.Accumulator{},
		},
		{
			name: "filtered nodes (excluded)",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": struct{}{},
					"/computer/api/json": nodeResponse{
						BusyExecutors:  4,
						TotalExecutors: 8,
						Computers: []node{
							{DisplayName: "ignore-1"},
							{DisplayName: "ignore-2"},
						},
					},
				},
			},
			output: &testutil.Accumulator{
				Metrics: []*testutil.Metric{
					{
						Tags: map[string]string{
							"source": "127.0.0.1",
						},
						Fields: map[string]interface{}{
							"busy_executors":  4,
							"total_executors": 8,
						},
					},
				},
			},
		},
		{
			name: "filtered nodes (included)",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": struct{}{},
					"/computer/api/json": nodeResponse{
						BusyExecutors:  4,
						TotalExecutors: 8,
						Computers: []node{
							{DisplayName: "filtered-1"},
							{DisplayName: "filtered-1"},
						},
					},
				},
			},
			output: &testutil.Accumulator{
				Metrics: []*testutil.Metric{
					{
						Tags: map[string]string{
							"source": "127.0.0.1",
						},
						Fields: map[string]interface{}{
							"busy_executors":  4,
							"total_executors": 8,
						},
					},
				},
			},
		},
		{
			name: "normal data collection",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": struct{}{},
					"/computer/api/json": nodeResponse{
						BusyExecutors:  4,
						TotalExecutors: 8,
						Computers: []node{
							{
								DisplayName: "master",
								MonitorData: monitorData{
									HudsonNodeMonitorsArchitectureMonitor: "linux",
									HudsonNodeMonitorsResponseTimeMonitor: &responseTimeMonitor{
										Average: 10032,
									},
									HudsonNodeMonitorsDiskSpaceMonitor: &nodeSpaceMonitor{
										Path: "/path/1",
										Size: 123,
									},
									HudsonNodeMonitorsTemporarySpaceMonitor: &nodeSpaceMonitor{
										Path: "/path/2",
										Size: 245,
									},
									HudsonNodeMonitorsSwapSpaceMonitor: &swapSpaceMonitor{
										SwapAvailable:   212,
										SwapTotal:       500,
										MemoryAvailable: 101,
										MemoryTotal:     500,
									},
								},
								Offline: false,
							},
						},
					},
				},
			},
			output: &testutil.Accumulator{
				Metrics: []*testutil.Metric{
					{
						Tags: map[string]string{
							"source": "127.0.0.1",
						},
						Fields: map[string]interface{}{
							"busy_executors":  4,
							"total_executors": 8,
						},
					},
					{
						Tags: map[string]string{
							"node_name": "master",
							"arch":      "linux",
							"status":    "online",
							"disk_path": "/path/1",
							"temp_path": "/path/2",
							"source":    "127.0.0.1",
						},
						Fields: map[string]interface{}{
							"response_time":    int64(10032),
							"disk_available":   float64(123),
							"temp_available":   float64(245),
							"swap_available":   float64(212),
							"swap_total":       float64(500),
							"memory_available": float64(101),
							"memory_total":     float64(500),
						},
					},
				},
			},
		},
		{
			name: "slave is offline",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": struct{}{},
					"/computer/api/json": nodeResponse{
						BusyExecutors:  4,
						TotalExecutors: 8,
						Computers: []node{
							{
								DisplayName:  "slave",
								MonitorData:  monitorData{},
								NumExecutors: 1,
								Offline:      true,
							},
						},
					},
				},
			},
			output: &testutil.Accumulator{
				Metrics: []*testutil.Metric{
					{
						Tags: map[string]string{
							"source": "127.0.0.1",
						},
						Fields: map[string]interface{}{
							"busy_executors":  4,
							"total_executors": 8,
						},
					},
					{
						Tags: map[string]string{
							"node_name": "slave",
							"status":    "offline",
						},
						Fields: map[string]interface{}{
							"num_executors": 1,
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ts := httptest.NewServer(test.input)
			defer ts.Close()
			j := &Jenkins{
				Log:             testutil.Logger{},
				URL:             ts.URL,
				ResponseTimeout: config.Duration(time.Microsecond),
				NodeExclude:     []string{"ignore-1", "ignore-2"},
				NodeInclude:     []string{"master", "slave"},
			}
			te := j.initialize(&http.Client{Transport: &http.Transport{}})
			acc := new(testutil.Accumulator)
			j.gatherNodesData(acc)
			if err := acc.FirstError(); err != nil {
				te = err
			}

			if !test.wantErr && te != nil {
				t.Fatalf("%s: failed %s, expected to be nil", test.name, te.Error())
			} else if test.wantErr && te == nil {
				t.Fatalf("%s: expected err, got nil", test.name)
			}
			if test.output == nil && len(acc.Metrics) > 0 {
				t.Fatalf("%s: collected extra data %s", test.name, acc.Metrics)
			} else if test.output != nil && len(test.output.Metrics) > 0 {
				for i := 0; i < len(test.output.Metrics); i++ {
					for k, m := range test.output.Metrics[i].Tags {
						if acc.Metrics[i].Tags[k] != m {
							t.Fatalf("%s: tag %s metrics unmatch Expected %s, got %s\n", test.name, k, m, acc.Metrics[0].Tags[k])
						}
					}
					for k, m := range test.output.Metrics[i].Fields {
						if acc.Metrics[i].Fields[k] != m {
							t.Fatalf("%s: field %s metrics unmatch Expected %v(%T), got %v(%T)\n",
								test.name, k, m, m, acc.Metrics[0].Fields[k], acc.Metrics[0].Fields[k])
						}
					}
				}
			}
		})
	}
}

func TestLabels(t *testing.T) {
	input := mockHandler{
		responseMap: map[string]interface{}{
			"/api/json": struct{}{},
			"/computer/api/json": nodeResponse{
				BusyExecutors:  4,
				TotalExecutors: 8,
				Computers: []node{
					{
						DisplayName: "master",
						AssignedLabels: []label{
							{"project_a"},
							{"testing"},
						},
						MonitorData: monitorData{
							HudsonNodeMonitorsResponseTimeMonitor: &responseTimeMonitor{
								Average: 54321,
							},
						},
					},
					{
						DisplayName: "secondary",
						MonitorData: monitorData{
							HudsonNodeMonitorsResponseTimeMonitor: &responseTimeMonitor{
								Average: 12345,
							},
						},
					},
				},
			},
		},
	}

	expected := []telegraf.Metric{
		testutil.MustMetric("jenkins",
			map[string]string{
				"source": "127.0.0.1",
			},
			map[string]interface{}{
				"busy_executors":  4,
				"total_executors": 8,
			},
			time.Unix(0, 0),
		),
		testutil.MustMetric("jenkins_node",
			map[string]string{
				"node_name": "master",
				"status":    "online",
				"source":    "127.0.0.1",
				"labels":    "project_a,testing",
			},
			map[string]interface{}{
				"num_executors": int64(0),
				"response_time": int64(54321),
			},
			time.Unix(0, 0),
		),
		testutil.MustMetric("jenkins_node",
			map[string]string{
				"node_name": "secondary",
				"status":    "online",
				"source":    "127.0.0.1",
				"labels":    "none",
			},
			map[string]interface{}{
				"num_executors": int64(0),
				"response_time": int64(12345),
			},
			time.Unix(0, 0),
		),
	}

	ts := httptest.NewServer(input)
	defer ts.Close()
	j := &Jenkins{
		Log:             testutil.Logger{},
		URL:             ts.URL,
		ResponseTimeout: config.Duration(time.Microsecond),
		NodeLabelsAsTag: true,
	}
	require.NoError(t, j.initialize(&http.Client{Transport: &http.Transport{}}))
	acc := new(testutil.Accumulator)
	j.gatherNodesData(acc)
	require.NoError(t, acc.FirstError())
	results := acc.GetTelegrafMetrics()
	for _, metric := range results {
		metric.RemoveTag("port")
	}
	testutil.RequireMetricsEqual(t, expected, results, testutil.IgnoreTime())
}

func TestInitialize(t *testing.T) {
	mh := mockHandler{
		responseMap: map[string]interface{}{
			"/api/json": struct{}{},
		},
	}
	ts := httptest.NewServer(mh)
	defer ts.Close()
	mockClient := &http.Client{Transport: &http.Transport{}}
	tests := []struct {
		// name of the test
		name    string
		input   *Jenkins
		output  *Jenkins
		wantErr bool
	}{
		{
			name: "bad jenkins config",
			input: &Jenkins{
				Log:             testutil.Logger{},
				URL:             "http://a bad url",
				ResponseTimeout: config.Duration(time.Microsecond),
			},
			wantErr: true,
		},
		{
			name: "has filter",
			input: &Jenkins{
				Log:             testutil.Logger{},
				URL:             ts.URL,
				ResponseTimeout: config.Duration(time.Microsecond),
				JobInclude:      []string{"jobA", "jobB"},
				JobExclude:      []string{"job1", "job2"},
				NodeExclude:     []string{"node1", "node2"},
			},
		},
		{
			name: "default config",
			input: &Jenkins{
				Log:             testutil.Logger{},
				URL:             ts.URL,
				ResponseTimeout: config.Duration(time.Microsecond),
			},
			output: &Jenkins{
				Log:               testutil.Logger{},
				MaxConnections:    5,
				MaxSubJobPerLayer: 10,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			te := test.input.initialize(mockClient)
			if !test.wantErr && te != nil {
				t.Fatalf("%s: failed %s, expected to be nil", test.name, te.Error())
			} else if test.wantErr && te == nil {
				t.Fatalf("%s: expected err, got nil", test.name)
			}
			if test.output != nil {
				if test.input.client == nil {
					t.Fatalf("%s: failed %v, jenkins instance shouldn't be nil", test.name, te)
				}
				if test.input.MaxConnections != test.output.MaxConnections {
					t.Fatalf("%s: different MaxConnections Expected %d, got %d\n", test.name, test.output.MaxConnections, test.input.MaxConnections)
				}
			}
		})
	}
}

func TestGatherJobs(t *testing.T) {
	tests := []struct {
		name    string
		input   mockHandler
		output  *testutil.Accumulator
		wantErr bool
	}{
		{
			name: "empty job",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": &jobResponse{},
				},
			},
		},
		{
			name: "bad inner jobs",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "job1"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "jobs has no build",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "job1"},
						},
					},
					"/job/job1/api/json": &jobResponse{},
				},
			},
		},
		{
			name: "bad build info",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "job1"},
						},
					},
					"/job/job1/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 1,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "ignore building job",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "job1"},
						},
					},
					"/job/job1/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 1,
						},
					},
					"/job/job1/1/api/json": &buildResponse{
						Building: true,
					},
				},
			},
		},
		{
			name: "ignore old build",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "job1"},
						},
					},
					"/job/job1/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 2,
						},
					},
					"/job/job1/2/api/json": &buildResponse{
						Building:  false,
						Timestamp: 100,
					},
				},
			},
		},
		{
			name: "gather metrics",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "job1"},
							{Name: "job2"},
						},
					},
					"/job/job1/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 3,
						},
					},
					"/job/job2/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 1,
						},
					},
					"/job/job1/3/api/json": &buildResponse{
						Building:  false,
						Result:    "SUCCESS",
						Duration:  25558,
						Number:    3,
						Timestamp: (time.Now().Unix() - int64(time.Minute.Seconds())) * 1000,
					},
					"/job/job2/1/api/json": &buildResponse{
						Building:  false,
						Result:    "FAILURE",
						Duration:  1558,
						Number:    1,
						Timestamp: (time.Now().Unix() - int64(time.Minute.Seconds())) * 1000,
					},
				},
			},
			output: &testutil.Accumulator{
				Metrics: []*testutil.Metric{
					{
						Tags: map[string]string{
							"name":   "job1",
							"result": "SUCCESS",
						},
						Fields: map[string]interface{}{
							"duration":    int64(25558),
							"number":      int64(3),
							"result_code": 0,
						},
					},
					{
						Tags: map[string]string{
							"name":   "job2",
							"result": "FAILURE",
						},
						Fields: map[string]interface{}{
							"duration":    int64(1558),
							"number":      int64(1),
							"result_code": 1,
						},
					},
				},
			},
		},
		{
			name: "gather metrics for jobs with space",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "job 1"},
						},
					},
					"/job/job%201/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 3,
						},
					},
					"/job/job%201/3/api/json": &buildResponse{
						Building:  false,
						Result:    "SUCCESS",
						Duration:  25558,
						Number:    3,
						Timestamp: (time.Now().Unix() - int64(time.Minute.Seconds())) * 1000,
					},
				},
			},
			output: &testutil.Accumulator{
				Metrics: []*testutil.Metric{
					{
						Tags: map[string]string{
							"name":   "job 1",
							"result": "SUCCESS",
						},
						Fields: map[string]interface{}{
							"duration":    int64(25558),
							"number":      int64(3),
							"result_code": 0,
						},
					},
				},
			},
		},
		{
			name: "gather metrics for nested jobs with space exercising append slice behaviour",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "l1"},
						},
					},
					"/job/l1/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "l2"},
						},
					},
					"/job/l1/job/l2/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "job 1"},
						},
					},
					"/job/l1/job/l2/job/job%201/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "job 2"},
						},
					},
					"/job/l1/job/l2/job/job%201/job/job%202/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 3,
						},
					},
					"/job/l1/job/l2/job/job%201/job/job%202/3/api/json": &buildResponse{
						Building:  false,
						Result:    "SUCCESS",
						Duration:  25558,
						Timestamp: (time.Now().Unix() - int64(time.Minute.Seconds())) * 1000,
					},
				},
			},
			output: &testutil.Accumulator{
				Metrics: []*testutil.Metric{
					{
						Tags: map[string]string{
							"name":    "job 2",
							"parents": "l1/l2/job 1",
							"result":  "SUCCESS",
						},
						Fields: map[string]interface{}{
							"duration":    int64(25558),
							"result_code": 0,
						},
					},
				},
			},
		},
		{
			name: "gather sub jobs, jobs filter",
			input: mockHandler{
				responseMap: map[string]interface{}{
					"/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "apps"},
							{Name: "ignore-1"},
						},
					},
					"/job/ignore-1/api/json": &jobResponse{},
					"/job/apps/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "k8s-cloud"},
							{Name: "chronograf"},
							{Name: "ignore-all"},
						},
					},
					"/job/apps/job/ignore-all/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "1"},
							{Name: "2"},
						},
					},
					"/job/apps/job/ignore-all/job/1/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 1,
						},
					},
					"/job/apps/job/ignore-all/job/2/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 1,
						},
					},
					"/job/apps/job/chronograf/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 1,
						},
					},
					"/job/apps/job/k8s-cloud/api/json": &jobResponse{
						Jobs: []innerJob{
							{Name: "PR-100"},
							{Name: "PR-101"},
							{Name: "PR-ignore2"},
							{Name: "PR 1"},
							{Name: "PR ignore"},
						},
					},
					"/job/apps/job/k8s-cloud/job/PR%20ignore/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 1,
						},
					},
					"/job/apps/job/k8s-cloud/job/PR-ignore2/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 1,
						},
					},
					"/job/apps/job/k8s-cloud/job/PR-100/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 1,
						},
					},
					"/job/apps/job/k8s-cloud/job/PR-101/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 4,
						},
					},
					"/job/apps/job/k8s-cloud/job/PR%201/api/json": &jobResponse{
						LastBuild: jobBuild{
							Number: 1,
						},
					},
					"/job/apps/job/chronograf/1/api/json": &buildResponse{
						Building:  false,
						Result:    "FAILURE",
						Duration:  1558,
						Number:    1,
						Timestamp: (time.Now().Unix() - int64(time.Minute.Seconds())) * 1000,
					},
					"/job/apps/job/k8s-cloud/job/PR-101/4/api/json": &buildResponse{
						Building:  false,
						Result:    "SUCCESS",
						Duration:  76558,
						Number:    4,
						Timestamp: (time.Now().Unix() - int64(time.Minute.Seconds())) * 1000,
					},
					"/job/apps/job/k8s-cloud/job/PR-100/1/api/json": &buildResponse{
						Building:  false,
						Result:    "SUCCESS",
						Duration:  91558,
						Number:    1,
						Timestamp: (time.Now().Unix() - int64(time.Minute.Seconds())) * 1000,
					},
					"/job/apps/job/k8s-cloud/job/PR%201/1/api/json": &buildResponse{
						Building:  false,
						Result:    "SUCCESS",
						Duration:  87832,
						Number:    1,
						Timestamp: (time.Now().Unix() - int64(time.Minute.Seconds())) * 1000,
					},
				},
			},
			output: &testutil.Accumulator{
				Metrics: []*testutil.Metric{
					{
						Tags: map[string]string{
							"name":    "PR 1",
							"parents": "apps/k8s-cloud",
							"result":  "SUCCESS",
						},
						Fields: map[string]interface{}{
							"duration":    int64(87832),
							"number":      int64(1),
							"result_code": 0,
						},
					},
					{
						Tags: map[string]string{
							"name":    "PR-100",
							"parents": "apps/k8s-cloud",
							"result":  "SUCCESS",
						},
						Fields: map[string]interface{}{
							"duration":    int64(91558),
							"number":      int64(1),
							"result_code": 0,
						},
					},
					{
						Tags: map[string]string{
							"name":    "PR-101",
							"parents": "apps/k8s-cloud",
							"result":  "SUCCESS",
						},
						Fields: map[string]interface{}{
							"duration":    int64(76558),
							"number":      int64(4),
							"result_code": 0,
						},
					},
					{
						Tags: map[string]string{
							"name":    "chronograf",
							"parents": "apps",
							"result":  "FAILURE",
						},
						Fields: map[string]interface{}{
							"duration":    int64(1558),
							"number":      int64(1),
							"result_code": 1,
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ts := httptest.NewServer(test.input)
			defer ts.Close()
			j := &Jenkins{
				Log:             testutil.Logger{},
				URL:             ts.URL,
				MaxBuildAge:     config.Duration(time.Hour),
				ResponseTimeout: config.Duration(time.Microsecond),
				JobInclude: []string{
					"*",
				},
				JobExclude: []string{
					"ignore-1",
					"apps/ignore-all/*",
					"apps/k8s-cloud/PR-ignore2",
					"apps/k8s-cloud/PR ignore",
				},
			}
			te := j.initialize(&http.Client{Transport: &http.Transport{}})
			acc := new(testutil.Accumulator)
			j.gatherJobs(acc)
			if err := acc.FirstError(); err != nil {
				te = err
			}
			if !test.wantErr && te != nil {
				t.Fatalf("%s: failed %s, expected to be nil", test.name, te.Error())
			} else if test.wantErr && te == nil {
				t.Fatalf("%s: expected err, got nil", test.name)
			}

			if test.output != nil && len(test.output.Metrics) > 0 {
				// sort metrics
				sort.Slice(acc.Metrics, func(i, j int) bool {
					return strings.Compare(acc.Metrics[i].Tags["name"], acc.Metrics[j].Tags["name"]) < 0
				})
				for i := range test.output.Metrics {
					for k, m := range test.output.Metrics[i].Tags {
						if acc.Metrics[i].Tags[k] != m {
							t.Fatalf("%s: tag %s metrics unmatch Expected %s, got %s\n", test.name, k, m, acc.Metrics[i].Tags[k])
						}
					}
					for k, m := range test.output.Metrics[i].Fields {
						if acc.Metrics[i].Fields[k] != m {
							t.Fatalf("%s: field %s metrics unmatch Expected %v(%T), got %v(%T)\n",
								test.name, k, m, m, acc.Metrics[i].Fields[k], acc.Metrics[0].Fields[k])
						}
					}
				}
			}
		})
	}
}
