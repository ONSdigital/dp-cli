package project_generation

var genericFiles = []fileGen{
	{
		templatePath: "generic/README.md",
		outputPath:   "README",
		extension:    ".md",
		filePrefix:   "",
	},
	{
		templatePath: "generic/CONTRIBUTING.md",
		outputPath:   "CONTRIBUTING",
		extension:    ".md",
		filePrefix:   "",
	},
	{
		templatePath: "generic/LICENSE.md",
		outputPath:   "LICENSE",
		extension:    ".md",
		filePrefix:   "",
	},
	{
		templatePath: "generic/.gitignore",
		outputPath:   ".gitignore",
		extension:    "",
		filePrefix:   "",
	},
	{
		templatePath: "generic/.github/PULL_REQUEST_TEMPLATE.md",
		outputPath:   ".github/PULL_REQUEST_TEMPLATE",
		extension:    ".md",
		filePrefix:   "",
	},
	{
		templatePath: "generic/.github/ISSUES_TEMPLATE.md",
		outputPath:   ".github/ISSUES_TEMPLATE",
		extension:    ".md",
		filePrefix:   "",
	},
}

var applicationFiles = []fileGen{
	{
		templatePath: "base-app/ci/build.yml",
		outputPath:   "ci/build",
		extension:    ".yml",
		filePrefix:   "",
	},
	{
		templatePath: "base-app/ci/unit.yml",
		outputPath:   "ci/unit",
		extension:    ".yml",
		filePrefix:   "",
	},
	{
		templatePath: "base-app/ci/audit.yml",
		outputPath:   "ci/audit",
		extension:    ".yml",
		filePrefix:   "",
	},
	{
		templatePath: "base-app/ci/scripts/build.sh",
		outputPath:   "ci/scripts/build",
		extension:    ".sh",
		filePrefix:   "",
		executable:   true,
	},
	{
		templatePath: "base-app/ci/scripts/unit.sh",
		outputPath:   "ci/scripts/unit",
		extension:    ".sh",
		filePrefix:   "",
		executable:   true,
	},
	{
		templatePath: "base-app/ci/scripts/audit.sh",
		outputPath:   "ci/scripts/audit",
		extension:    ".sh",
		filePrefix:   "",
		executable:   true,
	},
	{
		templatePath: "base-app/config/config.go",
		outputPath:   "config/config",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "base-app/config/config_test.go",
		outputPath:   "config/config_test",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "base-app/Dockerfile.concourse",
		outputPath:   "Dockerfile.concourse",
		extension:    "",
		filePrefix:   "",
	},
	{
		templatePath: "base-app/Makefile",
		outputPath:   "Makefile",
		extension:    "",
		filePrefix:   "",
	},
	{
		templatePath: "base-app/nomad",
		outputPath:   "",
		extension:    ".nomad",
		filePrefix:   "",
	},
	{
		templatePath: "base-app/README.md",
		outputPath:   "README",
		extension:    ".md",
		filePrefix:   "",
	},
}

var controllerFiles = []fileGen{
	{
		templatePath: "controller/config/config.go",
		outputPath:   "config/config",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "controller/handlers/handlers.go",
		outputPath:   "handlers/handlers",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "controller/handlers/handlers_test.go",
		outputPath:   "handlers/handlers_test",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "controller/routes/routes.go",
		outputPath:   "routes/routes",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "controller/mapper/mapper.go",
		outputPath:   "mapper/mapper",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "controller/mapper/mapper_test.go",
		outputPath:   "mapper/mapper_test",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "controller/main.go",
		outputPath:   "main",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "controller/README.md",
		outputPath:   "README",
		extension:    ".md",
		filePrefix:   "",
	},
}

var apiFiles = []fileGen{
	{
		templatePath: "api/api/api.go",
		outputPath:   "api/api",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "api/api/api_test.go",
		outputPath:   "api/api_test",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "api/api/hello.go",
		outputPath:   "api/hello",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "api/api/hello_test.go",
		outputPath:   "api/hello_test",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "api/service/service.go",
		outputPath:   "service/service",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "api/main.go",
		outputPath:   "main",
		extension:    ".go",
		filePrefix:   "",
	},
	{
		templatePath: "api/swagger.yaml",
		outputPath:   "swagger",
		extension:    ".yaml",
		filePrefix:   "",
	},
}

var eventFiles = []fileGen{
	//{
	// Todo event/
	// TODO Event
	// TODO Consumer
	// TODO Consumer_test
	// TODO handler
	//},
}
