package projectgeneration
var genericFiles = []fileGen{
	{
		path:      "readme",
		extension: ".md",
	},
	{
		path:      "contributing",
		extension: ".md",
	},
	{
		path:      "license",
		extension: ".md",
	},
	{
		path:      ".gitignore",
		extension: "",
	},
	{
		path:      ".github/PULL_REQUEST_TEMPLATE",
		extension: ".md",
	},
	{
		path:      ".github/ISSUES_TEMPLATE",
		extension: ".md",
	},
}

var applicationFiles = []fileGen{
	{
		path:      "ci/build",
		extension: ".yml",
	},
	{
		path:      "ci/unit",
		extension: ".yml",
	},
	{
		path:      "ci/scripts/build",
		extension: ".sh",
	},
	{
		path:      "ci/scripts/unit",
		extension: ".sh",
	},
	{
		path:      "Dockerfile.concourse",
		extension: "",
	},

	// TODO Make file
	// TODO {appname}.Nomad
	// TODO Config
	// TODO Main
}

var controllerFiles = []fileGen{
	{
		path:      "handlers/handlers",
		extension: ".go",
	},
	{
		path:      "handlers/handlers_test",
		extension: ".go",
	},
	{
		path:      "routes/routes",
		extension: ".go",
	},
	{
		path:      "routes/routes_test",
		extension: ".go",
	},
	{
		path:      "mapper/mapper",
		extension: ".go",
	},
	{
		path:      "mapper/mapper_test",
		extension: ".go",
	},
}

var apiFiles = []fileGen{
	{
		// TODO Swagger spec
		// TODO api/API.go
		// TODO api/API_test.go
		// TODO api/Hello.go
		// TODO api/hello_test.go
	},
}

var eventFiles = []fileGen{
	{
		// Todo event/
		// TODO Event
		// TODO Consumer
		// TODO Consumer_test
		// TODO handler
	},
}
