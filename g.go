// Copyright 2016 aletheia7. All rights reserved. Use of this source code is
// governed by a BSD-2-Clause license that can be found in the LICENSE file.

// Package gogroup provides synchronization, error propagation, and Context
// cancelation for groups of goroutines working on subtasks of a common task.
package gogroup

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Line_end byte

const (
	None      Line_end = iota
	Unix               // \n
	Osx                // \r
	Windows            // \r\n
	Backspace          // \b\b
)

// A Group is a collection of goroutines working on subtasks that are part of
// the same overall task.
//
type Group struct {
	Cancel        context.CancelFunc
	Ctx           context.Context
	wg            sync.WaitGroup
	err_once      sync.Once
	err           error
	wait_lock     sync.Mutex
	wait_index    int
	wait_register map[int]bool
}

// New returns a new Group and an associated Context derived from ctx.
// The derived Context is canceled the first time a function passed to Go
// returns a non-nil error or the first time Wait returns, whichever occurs
// first. New must be called to make a Group.
// ctx can be nil. If nil, context.Background() will be used.
//
func New(ctx context.Context) *Group {
	var con context.Context
	var cancel context.CancelFunc
	if ctx == nil {
		con, cancel = context.WithCancel(context.Background())
	} else {
		con, cancel = context.WithCancel(ctx)
	}
	return &Group{
		Ctx:           con,
		Cancel:        cancel,
		wait_register: map[int]bool{},
	}
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
//
func (g *Group) Wait() error {
	g.wg.Wait()
	if g.Cancel != nil {
		g.Cancel()
	}
	return g.err
}

type Grouper interface {
	Run(g *Group)
}

// Register increments the internal sync.WaitGroup. Unregister() must be
// called with the returned int to end Group.Wait(). goroutines using
// Register/Unregister should end upon receipt from the Group.Ctx.Done()
// channel.
//
func (g *Group) Register() int {
	g.wait_lock.Lock()
	defer g.wait_lock.Unlock()
	g.wg.Add(1)
	g.wait_index++
	g.wait_register[g.wait_index] = true
	return g.wait_index
}

// Unregister decrements the internal sync.WaitGroup and called
// Group.Cancel(). It is safe to call Unregister multiple times.
//
func (g *Group) Unregister(index int) {
	g.wait_lock.Lock()
	defer g.wait_lock.Unlock()
	if g.wait_register[index] {
		delete(g.wait_register, index)
		g.wg.Done()
		if g.Cancel != nil {
			g.Cancel()
		}
	}
}

// Go calls the given function in a new goroutine.
// The first call to return cancels the group. A Grouper should receive on
// Group.Ctx.Done() to gracefully end.
//
func (g *Group) Go(f Grouper) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		f.Run(g)
		if g.Cancel != nil {
			g.Cancel()
		}
	}()
}

// Set_err will set the returned error for the first time called.
//
func (g *Group) Set_err(err error) {
	g.err_once.Do(func() {
		g.err = err
	})
}

// Add_signals will call Group.Cancel() when an os.Interrupt or
// syscall.SIGTERM signal is received.
// An "end" character will be output to os.Stderr upon receiving an
// os.Interrupt or syscall.SIGTERM signal.
//
func (g *Group) Add_signals(end Line_end) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		ch := make(chan os.Signal, 1)
		defer close(ch)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(ch)
		select {
		case <-g.Ctx.Done():
		case <-ch:
			switch end {
			case None:
			case Backspace:
				fmt.Fprintf(os.Stderr, "\b\b")
			case Unix:
				fmt.Fprintf(os.Stderr, "\n")
			case Osx:
				fmt.Fprintf(os.Stderr, "\r")
			case Windows:
				fmt.Fprintf(os.Stderr, "\r\n")
			}
		}
		if g.Cancel != nil {
			g.Cancel()
		}
	}()
}
