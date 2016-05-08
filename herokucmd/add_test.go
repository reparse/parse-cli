package herokucmd

import (
	"testing"

	"./../parsecli"
	"github.com/facebookgo/ensure"
)

func TestGetRandomAppName(t *testing.T) {
	app := &parsecli.App{Name: "test app", ApplicationID: "123456789"}
	name := getRandomAppName(app)
	ensure.StringContains(t, name, "testapp")
}
