package scp

import (
	"os"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWithCWD(t *testing.T) {
	Convey("Given we want to test withCWD in a given dir", t, func() {

		So(os.Chdir("/var/tmp"), ShouldBeNil)
		origPath, err := os.Getwd()
		So(err, ShouldBeNil)

		Convey("When a full path is provided", func() {

			path, err := withCWD("/a/b")

			Convey("Then the original path should be returned", func() {
				So(err, ShouldBeNil)
				So(path, ShouldResemble, "/a/b")
			})
		})

		Convey("When a partial path is provided, below the current dir", func() {

			path, err := withCWD("a/b")

			Convey("Then the path should be returned as within the original dir", func() {
				So(err, ShouldBeNil)
				So(path, ShouldEqual, origPath+"/a/b")
			})
		})

		Convey("When a path dot-escaping the current path is provided", func() {

			path, err := withCWD("../a/b")

			Convey("Then the path should be returned outside the original dir", func() {
				// remove last directory element (ready for `../`)
				parentPath := origPath[0:strings.LastIndex(origPath, string(os.PathSeparator))]

				So(err, ShouldBeNil)
				So(path, ShouldResemble, parentPath+"/a/b")
			})
		})
	})
}
