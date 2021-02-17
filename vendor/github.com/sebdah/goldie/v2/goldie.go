// Package goldie provides test assertions based on golden files. It's
// typically used for testing responses with larger data bodies.
//
// The concept is straight forward. Valid response data is stored in a "golden
// file". The actual response data will be byte compared with the golden file
// and the test will fail if there is a difference.
//
// Updating the golden file can be done by running `go test -update ./...`.
package goldie

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const (
	// defaultFixtureDir is the folder name for where the fixtures are stored.
	// It's relative to the "go test" path.
	defaultFixtureDir = "testdata"

	// defaultFileNameSuffix is the suffix appended to the fixtures. Set to
	// empty string to disable file name suffixes.
	defaultFileNameSuffix = ".golden"

	// defaultFilePerms is used to set the permissions on the golden fixture
	// files.
	defaultFilePerms os.FileMode = 0644

	// defaultDirPerms is used to set the permissions on the golden fixture
	// folder.
	defaultDirPerms os.FileMode = 0755

	// defaultDiffEngine sets which diff engine to use if not defined.
	defaultDiffEngine = ClassicDiff

	// defaultIgnoreTemplateErrors sets the default value for the
	// WithIgnoreTemplateErrors option.
	defaultIgnoreTemplateErrors = false

	// defaultUseTestNameForDir sets the default value for the
	// WithTestNameForDir option.
	defaultUseTestNameForDir = false

	// defaultUseSubTestNameForDir sets the default value for the
	// WithSubTestNameForDir option.
	defaultUseSubTestNameForDir = false
)

var (
	// update determines if the actual received data should be written to the
	// golden files or not. This should be true when you need to update the
	// data, but false when actually running the tests.
	update = flag.Bool("update", false, "Update golden test file fixture")

	// clean determines if we should remove old golden test files in the output
	// directory or not. This only takes effect if we are updating the golden
	// test files.
	clean = flag.Bool("clean", false, "Clean old golden test files before writing new olds")

	// ts saves the timestamp of the test run, we use ts to mark the
	// modification time of golden file dirs, for cleaning if required by
	// `-clean` flag.
	ts = time.Now()
)

// Goldie is the root structure for the test runner. It provides test assertions based on golden files. It's
// typically used for testing responses with larger data bodies.
type Goldie struct {
	fixtureDir     string
	fileNameSuffix string
	filePerms      os.FileMode
	dirPerms       os.FileMode

	diffEngine           DiffEngine
	diffFn               DiffFn
	ignoreTemplateErrors bool
	useTestNameForDir    bool
	useSubTestNameForDir bool
}

// === Create new testers ==================================

// New creates a new golden file tester. If there is an issue with applying any
// of the options, an error will be reported and t.FailNow() will be called.
func New(t *testing.T, options ...Option) *Goldie {
	g := Goldie{
		fixtureDir:           defaultFixtureDir,
		fileNameSuffix:       defaultFileNameSuffix,
		filePerms:            defaultFilePerms,
		dirPerms:             defaultDirPerms,
		diffEngine:           defaultDiffEngine,
		ignoreTemplateErrors: defaultIgnoreTemplateErrors,
		useTestNameForDir:    defaultUseTestNameForDir,
		useSubTestNameForDir: defaultUseSubTestNameForDir,
	}

	var err error
	for _, option := range options {
		err = option(&g)
		if err != nil {
			t.Error(fmt.Errorf("could not apply option: %w", err))
			t.FailNow()
		}
	}

	return &g
}

// Diff generates a string that shows the difference between the actual and the
// expected. This method could be called in your own DiffFn in case you want
// to leverage any of the engines defined.
func Diff(engine DiffEngine, actual string, expected string) (diff string) {
	switch engine {
	case Simple:
		diff = fmt.Sprintf("Expected: %s\nGot: %s", expected, actual)

	case ClassicDiff:
		diff, _ = difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(expected),
			B:        difflib.SplitLines(actual),
			FromFile: "Expected",
			FromDate: "",
			ToFile:   "Actual",
			ToDate:   "",
			Context:  1,
		})

	case ColoredDiff:
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(actual, expected, false)
		diff = dmp.DiffPrettyText(diffs)
	}

	return diff
}

// Update will update the golden fixtures with the received actual data.
//
// This method does not need to be called from code, but it's exposed so that
// it can be explicitly called if needed. The more common approach would be to
// update using `go test -update ./...`.
func (g *Goldie) Update(t *testing.T, name string, actualData []byte) error {
	goldenFile := g.GoldenFileName(t, name)
	goldenFileDir := filepath.Dir(goldenFile)
	if err := g.ensureDir(goldenFileDir); err != nil {
		return err
	}

	if err := ioutil.WriteFile(goldenFile, actualData, g.filePerms); err != nil {
		return err
	}

	if err := os.Chtimes(goldenFileDir, ts, ts); err != nil {
		return err
	}

	return nil
}

// ensureDir will create the fixture folder if it does not already exist.
func (g *Goldie) ensureDir(loc string) error {
	s, err := os.Stat(loc)

	switch {
	case err != nil && os.IsNotExist(err):
		// the location does not exist, so make directories to there
		return os.MkdirAll(loc, g.dirPerms)

	case err == nil && s.IsDir() && *clean && s.ModTime().UnixNano() != ts.UnixNano():
		if err := os.RemoveAll(loc); err != nil {
			return err
		}
		return os.MkdirAll(loc, g.dirPerms)

	case err == nil && !s.IsDir():
		return newErrFixtureDirectoryIsFile(loc)
	}

	return err
}

// GoldenFileName simply returns the file name of the golden file fixture.
func (g *Goldie) GoldenFileName(t *testing.T, name string) string {
	dir := g.fixtureDir

	if g.useTestNameForDir {
		dir = filepath.Join(dir, strings.Split(t.Name(), "/")[0])
	}

	if g.useSubTestNameForDir {
		n := strings.Split(t.Name(), "/")
		if len(n) > 1 {
			dir = filepath.Join(dir, n[1])
		}
	}

	return filepath.Join(dir, fmt.Sprintf("%s%s", name, g.fileNameSuffix))
}
