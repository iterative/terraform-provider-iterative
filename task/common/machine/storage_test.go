package machine_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"terraform-provider-iterative/task/common/machine"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferExcludes(t *testing.T) {
	tests := []struct {
		description string
		exclude     []string
		expect      []string
	}{{
		description: "Test builtin rules to exclude terraform files.",
		exclude:     nil,
		expect: []string{
			"/a.txt",
			"/temp",
			"/temp/a.txt",
			"/temp/b.txt",
		},
	}, {
		description: "Test excluding using glob patterns.",
		exclude:     []string{"**.txt"},
		expect: []string{
			"/temp", // directory still gets transfered.
		},
	}, {
		description: "Test explicitly anchored excludes.",
		exclude:     []string{"/a.txt"},
		expect: []string{
			"/temp",
			"/temp/a.txt",
			"/temp/b.txt",
		},
	}, {
		description: "Test implicitly anchored excludes.",
		exclude:     []string{"a.txt"},
		expect: []string{
			"/temp",
			"/temp/a.txt",
			"/temp/b.txt",
		},
	}}
	ctx := context.Background()
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			dst := t.TempDir()
			err := machine.Transfer(ctx, "./testdata/transferTest", dst, test.exclude)
			require.NoError(t, err)
			require.ElementsMatch(t, test.expect, listDir(dst))
		})
	}
}

func listDir(dir string) []string {
	var entries []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		entries = append(entries, strings.TrimPrefix(path, dir))
		return nil
	})
	if err != nil {
		panic(err)
	}
	return entries
}
