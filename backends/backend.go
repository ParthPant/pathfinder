package backends

import (
	"context"
)

type IFileSystemBackend interface {
	Ls(ctx context.Context, input LsInput) (LsResult, error)
	Read(ctx context.Context, input ReadInput) (ReadResult, error)
	Grep(ctx context.Context, input GrepInput) (GrepResult, error)
	Glob(ctx context.Context, input GlobInput) (GlobResult, error)
	Write(ctx context.Context, input WriteInput) (WriteResult, error)
	Edit(ctx context.Context, input EditInput) (EditResult, error)

	GetRoot() string
}

type IExecutionBackend interface {
	Execute(ctx context.Context, input ExecuteInput) (ExecuteResult, error)
}

type FileSystemError struct {
	e string
}

func NewFSError(e string) FileSystemError {
	return FileSystemError{
		e,
	}
}

func (e FileSystemError) Error() string {
	return e.e
}
