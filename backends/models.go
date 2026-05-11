package backends

type FileInfo struct {
	Name       string `json:"name"`
	IsDir      bool   `json:"is_dir"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at"`
}

type LsInput struct {
	Path string `json:"path" tool:"Absolute directory path to list files from,required"`
}

type LsResult struct {
	Entries []FileInfo `json:"entries"`
	Error   error      `json:"error,omitzero"`
}

type ReadInput struct {
	Path   string `json:"path" tool:"Absolute or relative file path.,required"`
	Offset int    `json:"offset" tool:"Line offset to start reading from (0-indexed)." default:"0"`
	Limit  int    `json:"limit" tool:"Maximum number of lines to read.,required" default:"20000"`
}

type ReadResult struct {
	FileInfo    `json:"file_info"`
	FileContent string `json:"file_content"`
	Error       error  `json:"error,omitzero"`
}

type GrepMatch struct {
	Path  string `json:"path"`
	Line  int    `json:"line"`
	Text  string `json:"text"`
	Error error  `json:"error"`
}

type GrepInput struct {
	Pattern string  `json:"pattern" tool:"Literal string to search for (NOT regex)"`
	Path    string  `json:"path" tool:"Directory or file path to search in. Defaults to current directory" default:"."`
	Glob    *string `json:"glob,omitzero" tool:"Optional glob pattern to filter which files to search"`
}

type GrepResult struct {
	Matches []GrepMatch `json:"matches"`
	Error   error       `json:"error"`
}

type GlobInput struct {
	Pattern string  `json:"pattern" tool:"Glob pattern to match files against like '*.py' or '**/*.txt'."`
	Path    *string `json:"path,omitzero" tool:"Base directory to search from. Defaults to root ('/')"`
}

type GlobResult struct {
	Matches []FileInfo `json:"matches"`
	Error   error      `json:"error"`
}

type WriteInput struct {
	Path    string `json:"path" tool:"Path where the new files will be created,required"`
	Content string `json:"content" tool:"Text content to write to the file.,required"`
}

type WriteResult struct {
	Path  string `json:"path"`
	Error error  `json:"error"`
}

type EditInput struct {
	Path       string `json:"path" tool:"Path to the file to edit.,required"`
	OldString  string `json:"old_string" tool:"The text to search for and replace.,required"`
	NewString  string `json:"new_string" tool:"The replacement text.,required"`
	ReplaceAll bool   `json:"replace_all,omitzero" tool:"if 'true', relace all occurrences, if 'false' (default), replace only if exactly one occurrence exists." default:"false"`
}
type EditResult struct {
	Path        string `json:"path"`
	Occurrences int    `json:"occurrences"`
	Error       error  `json:"error"`
}

type ExecuteInput struct {
	Command string `json:"command" tool:"A Unix shell command to run on the computer.,required"`
}

type ExecuteResult struct {
	Command  string `json:"command"`
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
	Error    error  `json:"error"`
}
