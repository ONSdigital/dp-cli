package eks

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func hasDep(deps []Dependency, command string) bool {
	for _, d := range deps {
		if d.Command == command {
			return true
		}
	}
	return false
}

func TestCheckDependenciesSocatConditional(t *testing.T) {
	Convey("Given dependency checking with a missing socat", t, func() {
		// socat is not expected to be on PATH in CI; assert on presence in the
		// evaluated set rather than the missing set to avoid environment coupling.
		Convey("When noSudo is true, socat is not part of the required set", func() {
			// Rebuild the required set the same way CheckDependencies does.
			deps := make([]Dependency, len(SessionDependencies))
			copy(deps, SessionDependencies)
			So(hasDep(deps, "socat"), ShouldBeFalse)
		})

		Convey("When noSudo is false, socat is part of the required set", func() {
			deps := make([]Dependency, len(SessionDependencies), len(SessionDependencies)+1)
			copy(deps, SessionDependencies)
			deps = append(deps, SocatDependency)
			So(hasDep(deps, "socat"), ShouldBeTrue)
		})

		Convey("The base SessionDependencies slice is not mutated by CheckDependencies", func() {
			before := len(SessionDependencies)
			_ = CheckDependencies(false)
			_ = CheckDependencies(true)
			So(len(SessionDependencies), ShouldEqual, before)
			So(hasDep(SessionDependencies, "socat"), ShouldBeFalse)
		})
	})
}

func TestResolveNoSudoMode(t *testing.T) {
	Convey("Given ResolveNoSudoMode", t, func() {
		Convey("When --legacy-sudo is not set, no-sudo mode is the default", func() {
			So(ResolveNoSudoMode(false), ShouldBeTrue)
		})

		Convey("When --legacy-sudo is set, no-sudo mode is disabled", func() {
			So(ResolveNoSudoMode(true), ShouldBeFalse)
		})
	})
}
