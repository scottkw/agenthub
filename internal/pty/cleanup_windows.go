//go:build windows

package pty

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	jobObjectLimitKillOnJobClose = 0x00002000
	// processAccessForJob is the access rights needed to assign a process to a
	// Job Object.
	processAccessForJob = windows.PROCESS_TERMINATE | windows.SYNCHRONIZE | windows.PROCESS_DUP_HANDLE
)

// jobObject wraps a Windows Job Object handle.
// When the handle is closed, all assigned processes are terminated.
type jobObject struct {
	handle windows.Handle
}

// newJobObject creates a new Job Object configured to kill all assigned
// processes when the last handle is closed (JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE).
func newJobObject() (*jobObject, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("CreateJobObject: %w", err)
	}

	info := windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
		LimitFlags: jobObjectLimitKillOnJobClose,
	}
	extInfo := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: info,
	}

	_, err = windows.SetInformationJobObject(
		h,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&extInfo)),
		uint32(unsafe.Sizeof(extInfo)),
	)
	if err != nil {
		_ = windows.CloseHandle(h)
		return nil, fmt.Errorf("SetInformationJobObject: %w", err)
	}

	return &jobObject{handle: h}, nil
}

// Assign adds a process to the job object.
func (j *jobObject) Assign(p *os.Process) error {
	handle, err := windows.OpenProcess(
		processAccessForJob,
		false,
		uint32(p.Pid),
	)
	if err != nil {
		return fmt.Errorf("OpenProcess: %w", err)
	}
	defer windows.CloseHandle(handle)

	if err := windows.AssignProcessToJobObject(j.handle, handle); err != nil {
		return fmt.Errorf("AssignProcessToJobObject: %w", err)
	}
	return nil
}

// Close closes the job object handle, which triggers termination of all
// assigned processes.
func (j *jobObject) Close() error {
	if j.handle == 0 {
		return nil
	}
	err := windows.CloseHandle(j.handle)
	j.handle = 0
	return err
}

// killSession terminates a session via its Job Object on Windows.
func killSession(s *Session) error {
	// Close the job object — triggers kill of all assigned processes.
	if s.job != nil {
		if jo, ok := s.job.(*jobObject); ok {
			_ = jo.Close()
		}
	}

	// Close the PTY.
	if s.pty != nil {
		_ = s.pty.Close()
	}

	// Cancel the context.
	if s.cancel != nil {
		s.cancel()
	}

	return nil
}
