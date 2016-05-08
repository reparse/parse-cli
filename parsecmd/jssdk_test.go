package parsecmd

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"./../parsecli"
	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

var defaultParseConfig = &parsecli.ParseConfig{
	ProjectConfig: &parsecli.ProjectConfig{
		Type:  parsecli.LegacyParseFormat,
		Parse: &parsecli.ParseProjectConfig{},
	},
}

func newJsSdkHarness(t testing.TB) *parsecli.Harness {
	h := parsecli.NewHarness(t)
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/jsVersions")
		rows := jsSDKVersion{JS: []string{"1.2.8", "1.2.9", "1.2.10", "1.2.11", "0.2.0"}}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, rows))),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	return h
}

func newJsSdkHarnessError(t testing.TB) *parsecli.Harness {
	h := parsecli.NewHarness(t)
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/jsVersions")
		return &http.Response{
			StatusCode: http.StatusExpectationFailed,
			Body:       ioutil.NopCloser(strings.NewReader(`{"error":"something is wrong"}`)),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	return h
}

func newJsSdkHarnessWithConfig(t testing.TB) (*parsecli.Harness, *parsecli.Context) {
	h := newJsSdkHarness(t)
	h.MakeEmptyRoot()

	ensure.Nil(t, parsecli.CloneSampleCloudCode(h.Env, true))
	h.Out.Reset()

	c, err := parsecli.ConfigFromDir(h.Env.Root)
	ensure.Nil(t, err)

	config, ok := (c).(*parsecli.ParseConfig)
	ensure.True(t, ok)

	return h, &parsecli.Context{Config: config}
}

func TestGetAllJSVersions(t *testing.T) {
	t.Parallel()
	h := newJsSdkHarness(t)
	defer h.Stop()
	j := jsSDKCmd{}
	versions, err := j.getAllJSSdks(h.Env)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, versions, []string{"1.2.11", "1.2.10", "1.2.9", "1.2.8", "0.2.0"})
}

func TestGetAllJSVersionsError(t *testing.T) {
	t.Parallel()
	h := newJsSdkHarnessError(t)
	defer h.Stop()
	j := jsSDKCmd{}
	_, err := j.getAllJSSdks(h.Env)
	ensure.Err(t, err, regexp.MustCompile(`something is wrong`))
}

func TestPrintVersions(t *testing.T) {
	t.Parallel()
	h, c := newJsSdkHarnessWithConfig(t)
	defer h.Stop()
	j := jsSDKCmd{}
	ensure.Nil(t, j.printVersions(h.Env, c))
	ensure.DeepEqual(t, h.Out.String(),
		`   1.2.11
   1.2.10
   1.2.9
   1.2.8
   0.2.0
`)
}

func TestSetVersionInvalid(t *testing.T) {
	t.Parallel()
	h, c := newJsSdkHarnessWithConfig(t)
	defer h.Stop()
	j := jsSDKCmd{newVersion: "1.2.12"}
	ensure.Err(t, j.setVersion(h.Env, c), regexp.MustCompile("Invalid SDK version selected"))
}

func TestSetVersionNoneSelected(t *testing.T) {
	t.Parallel()
	h := parsecli.NewHarness(t)
	defer h.Stop()

	c := &parsecli.Context{Config: defaultParseConfig}
	var j jsSDKCmd

	c.Config.GetProjectConfig().Parse.JSSDK = "1.2.1"
	ensure.Nil(t, j.getVersion(h.Env, c))
	ensure.DeepEqual(t, "Current JavaScript SDK version is 1.2.1\n",
		h.Out.String())

	c.Config.GetProjectConfig().Parse.JSSDK = ""
	ensure.Err(t, j.getVersion(h.Env, c),
		regexp.MustCompile("JavaScript SDK version not set for this project."))
}

func TestSetValidVersion(t *testing.T) {
	t.Parallel()

	h, c := newJsSdkHarnessWithConfig(t)
	defer h.Stop()
	j := jsSDKCmd{newVersion: "1.2.11"}
	ensure.Nil(t, j.setVersion(h.Env, c))
	ensure.DeepEqual(t, h.Out.String(), "Current JavaScript SDK version is 1.2.11\n")

	content, err := ioutil.ReadFile(filepath.Join(h.Env.Root, parsecli.ParseProject))
	ensure.Nil(t, err)
	ensure.DeepEqual(t, string(content), `{
  "project_type": 1,
  "parse": {
    "jssdk": "1.2.11"
  }
}`)
}

// NOTE: testing for legacy config format
func newLegacyJsSdkHarnessWithConfig(t testing.TB) (*parsecli.Harness, *parsecli.Context) {
	h := newJsSdkHarness(t)
	h.MakeEmptyRoot()

	ensure.Nil(t, os.Mkdir(filepath.Join(h.Env.Root, parsecli.ConfigDir), 0755))
	path := filepath.Join(h.Env.Root, parsecli.LegacyConfigFile)
	ensure.Nil(t, ioutil.WriteFile(path,
		[]byte(`{
		"global": {
			"parseVersion" : "1.2.9"
		}
	}`),
		0600))

	c, err := parsecli.ConfigFromDir(h.Env.Root)
	ensure.Nil(t, err)

	config, ok := (c).(*parsecli.ParseConfig)
	ensure.True(t, ok)

	return h, &parsecli.Context{Config: config}
}

func TestLegacySetValidVersion(t *testing.T) {
	t.Parallel()

	h, c := newLegacyJsSdkHarnessWithConfig(t)
	defer h.Stop()
	j := jsSDKCmd{newVersion: "1.2.11"}
	ensure.Nil(t, j.setVersion(h.Env, c))
	ensure.DeepEqual(t, h.Out.String(), "Current JavaScript SDK version is 1.2.11\n")
}
