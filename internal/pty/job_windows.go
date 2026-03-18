//go:build windows

package pty

import (
	"log"
	"os"
)

// assignJobObject creates a Windows Job Object and assigns the process to it.
// The job object handle is stored in sess.job so killSession can close it.
func assignJobObject(sess *Session, proc *os.Process) {
	jo, err := newJobObject()
	if err != nil {
		log.Printf("[warn] create job object: %v", err)
		return
	}
	if err := jo.Assign(proc); err != nil {
		log.Printf("[warn] assign process to job object: %v", err)
		_ = jo.Close()
		return
	}
	sess.job = jo
}
