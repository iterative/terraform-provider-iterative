package report

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"terraform-provider-iterative/task/common"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParamsArrayToMap(t *testing.T) {
	tests := []struct {
		description string
		params      []string
		expectMap   map[string]interface{}
		expectErr   string
	}{{
		description: "simple test case",
		params: []string{
			"key=value",
			"key2=value2",
		},
		expectMap: map[string]interface{}{
			"key":  "value",
			"key2": "value2",
		},
	}, {
		description: "nested maps",
		params: []string{
			"key=value",
			"key2.subkey=value2",
			"key2.id=id1",
		},
		expectMap: map[string]interface{}{
			"key": "value",
			"key2": map[string]interface{}{
				"subkey": "value2",
				"id":     "id1",
			},
		},
	}, {
		description: "conflicting params",
		params: []string{
			"key=value",
			"key.subkey=value2",
		},
		expectMap: nil,
		expectErr: `conflicting parameters "key" and "key.subkey"`,
	}, {
		description: "overwriting values",
		params: []string{
			"key=value",
			"key2=value2",
			"key=new value",
		},
		expectMap: map[string]interface{}{
			"key":  "new value",
			"key2": "value2",
		},
	}}
	for i, test := range tests {
		t.Run(fmt.Sprintf("test %d: %s", i, test.description), func(t *testing.T) {
			result, err := paramsArrayToMap(test.params)
			if test.expectErr != "" {
				require.EqualError(t, err, test.expectErr)
			} else {
				require.NoError(t, err)
			}
			require.EqualValues(t, test.expectMap, result)
		})
	}
}

func TestSendReport(t *testing.T) {
	received := make(chan map[string]interface{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(
		func(resp http.ResponseWriter, req *http.Request) {
			var body map[string]interface{}
			err := json.NewDecoder(req.Body).Decode(&body)
			require.NoError(t, err)
			received <- body
			resp.Write([]byte("ok"))
		}))
	cmd := New(&common.Cloud{})
	cmd.SetArgs([]string{
		"--url=" + srv.URL,
		"--token=token",
		"--type=data",
		"--repo=repo",
		"-p=key=value",
	})
	err := cmd.Execute()
	require.NoError(t, err)

	select {
	case report := <-received:
		expected := map[string]interface{}{
			"client":   "leo",
			"params":   map[string]interface{}{"key": "value"},
			"repo_url": "repo",
			"type":     "data",
		}
		require.EqualValues(t, expected, report)
	case <-time.After(time.Second):
		t.Error("timeout waiting for report")
	}
}
