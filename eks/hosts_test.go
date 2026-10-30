package eks

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestHostsLeadingNewline(t *testing.T) {
	Convey("Given hostsLeadingNewline", t, func() {
		Convey("When the file ends with a newline, no leading newline is added", func() {
			So(hostsLeadingNewline([]byte("127.0.0.1 localhost\n")), ShouldEqual, "")
		})

		Convey("When the file does not end with a newline, a leading newline is added", func() {
			So(hostsLeadingNewline([]byte("127.0.0.1 localhost")), ShouldEqual, "\n")
		})

		Convey("When the file is empty, no leading newline is added", func() {
			So(hostsLeadingNewline([]byte("")), ShouldEqual, "")
		})
	})
}
