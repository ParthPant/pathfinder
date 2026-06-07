package backends

import "github.com/ParthPant/pathfinder/tools"

func ExecuteToolDefinition(e IExecutionBackend) (tools.FunctionDefinition, error) {
	return tools.NewFunctionDefinition("execute",
		"Run a unix shell command.",
		tools.ParamsFor[ExecuteInput](),
		false,
		e.Execute,
	)
}

func PWDToolDefinition(fs IFileSystemBackend) (tools.FunctionDefinition, error) {
	return tools.NewFunctionDefinition("pwd",
		"Get the path of current working directory.",
		tools.ParamsFor[PWDInput](),
		false,
		fs.PWD,
	)
}

func LsToolDefinition(fs IFileSystemBackend) (tools.FunctionDefinition, error) {
	return tools.NewFunctionDefinition("ls",
		"List files and directories in the specified directory (non-recursive)",
		tools.ParamsFor[LsInput](),
		false,
		fs.Ls,
	)
}

func ReadToolDefinition(fs IFileSystemBackend) (tools.FunctionDefinition, error) {
	return tools.NewFunctionDefinition("read",
		"Read file content for the requested line range.",
		tools.ParamsFor[ReadInput](),
		false,
		fs.Read,
	)
}

func GrepToolDefinition(fs IFileSystemBackend) (tools.FunctionDefinition, error) {
	return tools.NewFunctionDefinition("grep",
		"Search for a literal text pattern in files.",
		tools.ParamsFor[GrepInput](),
		false,
		fs.Grep,
	)
}

func GlobToolDefinition(fs IFileSystemBackend) (tools.FunctionDefinition, error) {
	return tools.NewFunctionDefinition("glob",
		"Find files matching a glop pattern",
		tools.ParamsFor[GlobInput](),
		false,
		fs.Glob,
	)
}

func WriteToolDefinition(fs IFileSystemBackend) (tools.FunctionDefinition, error) {
	return tools.NewFunctionDefinition("write",
		"Create a new file with content.",
		tools.ParamsFor[WriteInput](),
		false,
		fs.Write,
	)
}

func EditToolDefinition(fs IFileSystemBackend) (tools.FunctionDefinition, error) {
	return tools.NewFunctionDefinition("edit",
		"Edit a file by replacing string occurrences",
		tools.ParamsFor[EditInput](),
		false,
		fs.Edit,
	)
}
