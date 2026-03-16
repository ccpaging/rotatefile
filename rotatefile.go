// Copyright (C) 2021, ccpaging <ccpaging@gmail.com>.  All rights reserved.

// Package file provides a file writer with buffering, rolling up automatic,
// thread safe(using sync lock) functionality.
//
// Here is a simple example, opening a file and writing some of it.
//
//	file, err := os.Open("file.go") // For read access.
//	if err != nil {
//		log.Fatal(err)
//	}
//
// If the open fails, the error string will be self-explanatory, like
//
//	open file.go: no such directory
//
// If the open success, the file should be really opened just before writing.
//
// The file's data can then be read into a slice of bytes. Read and
// Write take their byte counts from the length of the argument slice.
//
//	data := make([]byte, 100)
//	count, err := file.Write(data)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("read %d bytes: %q\n", count, data[:count])
//

package rotatefile

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strconv"
)

var (
	DefaultFileFlag = os.O_WRONLY | os.O_APPEND | os.O_CREATE

	// permission to:  owner      group      other
	//                 /```\      /```\      /```\
	// octal:            6          6          6
	// binary:         1 1 0      1 1 0      1 1 0
	// what to permit: r w x      r w x      r w x
	// binary         - 1: enabled, 0: disabled
	// what to permit - r: read, w: write, x: execute
	// permission to  - owner: the user that create the file/folder
	//                  group: the users from group that owner is member
	//                  other: all other users
	DefaultFileMode = os.FileMode(0660)

	DefaultLimitSize int64 = 1024 * 1024

	DefaultBufferSize = 2 * os.Getpagesize()
)

// File represents the buffered writer, and rolling up automatic.
type File struct {
	FilePath    string
	FileMode    os.FileMode
	LimitSize   int64
	BackupFiles int

	file *os.File
	size int64

	bufWriter  *bufio.Writer
	Buffersize int
}

// Open opens the named file for writing. If successful, methods on
// the returned file can be used for writing; the associated file
// descriptor has mode os.O_WRONLY|os.O_APPEND|os.O_CREATE.
// If there is an error, it will be of type *PathError.
func Open(filePath string) (*File, error) {
	return OpenFile(filePath, DefaultLimitSize, 1)
}

// OpenFile is the generalized open call; most users will use Open
// instead. It is created with mode perm (before umask) if necessary.
// If successful, methods on the returned File can be used for io.Writer.
func OpenFile(filePath string, limitSize int64, backupFiles int) (*File, error) {
	path := filepath.Dir(filePath)
	if stat, err := os.Stat(path); err != nil {
		return nil, err
	} else if !stat.IsDir() {
		return nil, errors.New("Error: The file path " + filePath + ": is not a directory.")
	}

	if limitSize <= 0 {
		limitSize = DefaultLimitSize
	}
	if backupFiles < 0 {
		backupFiles = 1
	}

	f := &File{
		FilePath:    filePath,
		FileMode:    DefaultFileMode,
		LimitSize:   limitSize,
		BackupFiles: backupFiles,
		Buffersize:  DefaultBufferSize,
	}
	f.size = f.fileSize()
	return f, nil
}

func (f *File) close() (err error) {
	if f.bufWriter != nil {
		f.bufWriter.Flush()
	}

	if f.file != nil {
		err = f.file.Close()
	}

	f.size = 0
	f.file = nil
	f.bufWriter = nil
	return
}

// Close active buffered writer.
func (f *File) Close() error {
	return f.close()
}

func (f *File) open() error {
	if f.file != nil {
		return nil
	}
	file, err := os.OpenFile(f.FilePath, DefaultFileFlag, f.FileMode)
	if err != nil {
		return err
	}

	f.file = file
	f.bufWriter = nil
	if f.Buffersize > 0 {
		f.bufWriter = bufio.NewWriterSize(f.file, f.Buffersize)
	}

	f.size = 0
	if fi, err := f.file.Stat(); err == nil {
		f.size = fi.Size()
	}
	return nil
}

func (f *File) write(b []byte) (n int, err error) {
	if f.bufWriter != nil {
		n, err = f.bufWriter.Write(b)
	} else {
		n, err = f.file.Write(b)
	}

	if err == nil {
		f.size += int64(n)
		f.flush()
	}
	return
}

// Write bytes to file, and rolling up automatic.
func (f *File) Write(b []byte) (n int, err error) {
	if f.LimitSize > 0 && f.size > f.LimitSize {
		f.rolling(f.BackupFiles)
	}

	if err := f.open(); err != nil {
		return 0, err
	}

	return f.write(b)
}

func (f *File) rolling(n int) {
	f.close()

	if n < 1 {
		// no backup file
		os.Remove(f.FilePath)
		return
	}

	ext := filepath.Ext(f.FilePath)                  // save extension like ".log"
	name := f.FilePath[0 : len(f.FilePath)-len(ext)] // dir and name

	var (
		i    int
		err  error
		slot string
	)

	for i = 0; i < n; i++ {
		// File name pattern is "name.<n>.ext"
		slot = name + "." + strconv.Itoa(i+1) + ext
		_, err = os.Stat(slot)
		if err != nil {
			break
		}
	}
	if err == nil {
		// Too much backup files. Remove last one
		os.Remove(slot)
		i--
	}

	for ; i > 0; i-- {
		prev := name + "." + strconv.Itoa(i) + ext
		os.Rename(prev, slot)
		slot = prev
	}

	os.Rename(f.FilePath, name+".1"+ext)
}

func (f *File) flush() {
	if f.bufWriter != nil {
		f.bufWriter.Flush()
	}
	if f.file != nil {
		f.file.Sync()
	}
}

func (f *File) fileSize() int64 {
	if f.file != nil {
		return f.size
	}
	fi, err := os.Stat(f.FilePath)
	if err != nil {
		return f.size
	}
	return fi.Size()
}
