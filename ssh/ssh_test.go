package ssh

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetPortArguments(t *testing.T) {
	Convey("Given port forwarding arguments need to be translated", t, func() {

		Convey("When a single port is provided", func() {

			sshArgs, err := getSSHPortArguments("11400")
			Convey("Then there should be no error returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the ssh forwarding ports should match", func() {
				So(sshArgs, ShouldNotBeEmpty)
				So(sshArgs, ShouldHaveLength, 2)
				So(sshArgs[0], ShouldResemble, "-L")
				So(sshArgs[1], ShouldResemble, "11400:localhost:11400")
			})
		})

		Convey("When different local and remote ports are provided", func() {

			sshArgs, err := getSSHPortArguments("11500:11400")
			Convey("Then there should be no error returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the ssh forwarding ports should differ", func() {
				So(sshArgs, ShouldNotBeEmpty)
				So(sshArgs, ShouldHaveLength, 2)
				So(sshArgs[0], ShouldResemble, "-L")
				So(sshArgs[1], ShouldResemble, "11500:localhost:11400")
			})
		})

		Convey("When full local port, remote host and port are provided", func() {

			sshArgs, err := getSSHPortArguments("11500:hosty:11400")
			Convey("Then there should be no error returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the ssh forwarding ports should differ", func() {
				So(sshArgs, ShouldNotBeEmpty)
				So(sshArgs, ShouldHaveLength, 2)
				So(sshArgs[0], ShouldResemble, "-L")
				So(sshArgs[1], ShouldResemble, "11500:hosty:11400")
			})
		})

		Convey("When invalid arguments are supplied", func() {
			invalidCases := []string{
				"moo",
				":123",
				"123:",
				"123::123",
				"123:hosty:nope",
				"123:hosty:",
				"123:hosty:3:4",
				"",
			}

			for _, invalid := range invalidCases {
				sshArgs, err := getSSHPortArguments(invalid)
				Convey(fmt.Sprintf("Then there should be an error returned for '%s'", invalid), func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "is not a valid port forwarding argument")
				})

				Convey(fmt.Sprintf("Then there should be no ssh arguments returned for '%s'", invalid), func() {
					So(sshArgs, ShouldBeNil)
				})
			}
		})
	})
}
