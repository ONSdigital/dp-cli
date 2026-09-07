package eks

import (
	"context"
	"fmt"
	"net"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// listen is a test helper that binds a TCP port via net.ListenConfig (the
// context-aware API the linter requires).
func listen(addr string) (net.Listener, error) {
	var lc net.ListenConfig
	return lc.Listen(context.Background(), "tcp", addr)
}

func TestAllocateLocalPort(t *testing.T) {
	Convey("Given a configured port range", t, func() {
		origBase, origMax := basePort, maxPort
		t.Cleanup(func() { basePort, maxPort = origBase, origMax })

		Convey("When a port in the range is already bound", func() {
			// Reserve one port, then constrain the range to [reserved, reserved+2].
			blocker, err := listen("127.0.0.1:0")
			So(err, ShouldBeNil)
			defer blocker.Close()
			reserved := blocker.Addr().(*net.TCPAddr).Port

			basePort, maxPort = reserved, reserved+2

			Convey("Then AllocateLocalPort skips the bound port", func() {
				port, err := AllocateLocalPort()
				So(err, ShouldBeNil)
				So(port, ShouldNotEqual, reserved)
				So(port, ShouldBeGreaterThan, reserved)
			})
		})

		Convey("When AllocateAndHoldLocalPort holds a port", func() {
			// Use an ephemeral port to seed a known-free value, then release it and
			// point the range at it so the allocator can grab it.
			seed, err := listen("127.0.0.1:0")
			So(err, ShouldBeNil)
			free := seed.Addr().(*net.TCPAddr).Port
			seed.Close()
			basePort, maxPort = free, free

			port, ln, err := AllocateAndHoldLocalPort()
			So(err, ShouldBeNil)
			So(port, ShouldEqual, free)
			defer ln.Close()

			Convey("Then the held port cannot be bound again while held", func() {
				_, err := listen(fmt.Sprintf("127.0.0.1:%d", port))
				So(err, ShouldNotBeNil)
			})

			Convey("And a subsequent allocation on the same single-port range fails", func() {
				_, _, err := AllocateAndHoldLocalPort()
				So(err, ShouldNotBeNil)
			})
		})

		Convey("When the whole range is exhausted", func() {
			blocker, err := listen("127.0.0.1:0")
			So(err, ShouldBeNil)
			defer blocker.Close()
			p := blocker.Addr().(*net.TCPAddr).Port
			basePort, maxPort = p, p // single port, already taken

			_, err = AllocateLocalPort()
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldContainSubstring, "no available ports")
		})
	})
}
