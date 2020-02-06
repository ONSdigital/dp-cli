package projectgeneration

var genericFiles = []fileGen{
	{
		templatePath: "readme",
		outputPath:   "readme",
		extension:    ".md",
		filePrefix:   "",
	},
	{
		templatePath: "contributing",
		outputPath:   "contributing",
		extension:    ".md",
		filePrefix:   "",
	},
	{
		templatePath: "license",
		outputPath:   "license",
		extension:    ".md",
		filePrefix:   "",
	},
	{
		templatePath: ".gitignore",
		outputPath:   ".gitignore",
		extension:    "",
		filePrefix:   "",
	},
	{
		templatePath: ".github/PULL_REQUEST_TEMPLATE",
		outputPath:   ".github/PULL_REQUEST_TEMPLATE",
		extension:    ".md",
		filePrefix:   "",
	},
	{
		templatePath: ".github/ISSUES_TEMPLATE",
		outputPath:   ".github/ISSUES_TEMPLATE",
		extension:    ".md",
		filePrefix:   "",
	},
}

var applicationFiles = []fileGen{
	{
		templatePath: "ci/build",
		outputPath:   "ci/build",
		extension:    ".yml",
		filePrefix:   "",
	},
	{
		templatePath: "ci/unit",
		outputPath:   "ci/unit",
		extension:    ".yml",
		filePrefix:   "",
	},
	{
		templatePath: "ci/scripts/build",
		outputPath:   "ci/scripts/build",
		extension:    ".sh",
		filePrefix:   "",
	},
	{
		templatePath: "ci/scripts/unit",
		outputPath:   "ci/scripts/unit",
		extension:    ".sh",
		filePrefix:   "",
	},
	{
		templatePath: "config/config",
		outputPath:   "config/config",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "config/config_test",
		outputPath:   "config/config_test",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "Dockerfile.concourse",
		outputPath:   "Dockerfile.concourse",
		extension:    "",
		filePrefix:   "",
	},
	{
		templatePath: "Makefile",
		outputPath:   "Makefile",
		extension:    "",
		filePrefix:   "",
	},
	{
		templatePath: "nomad",
		outputPath:   "",
		extension:    ".nomad",
		filePrefix:   "",
	},
}

var controllerFiles = []fileGen{
	{
		templatePath: "handlers/handlers",
		outputPath:   "handlers/handlers",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "handlers/handlers_test",
		outputPath:   "handlers/handlers_test",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "routes/routes",
		outputPath:   "routes/routes",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "routes/routes_test",
		outputPath:   "routes/routes_test",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "mapper/mapper",
		outputPath:   "mapper/mapper",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "mapper/mapper_test",
		outputPath:   "mapper/mapper_test",
		extension:    ".go",
		filePrefix:   "",
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
