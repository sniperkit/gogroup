[![](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/aletheia7/gogroup) 

#### Documentation

gogroup allows running a group of goroutines. The gogroup.Group  waits for
all goroutines to end. All goroutines in the group are signaled through a
context to end gracefully when one goroutine ends.

#### Example

```go
package main

import (
	"context"
	"fmt"
	"gogroup"
	"log"
	"time"
)

func main() {
	log.SetFlags(log.Lshortfile)
	g := gogroup.New(context.Background())
	g.Add_signals(gogroup.Unix)
	go hi(g)
	g.Go(&Do{7})
	g.Go(&Do{8})
	log.Println("wait done:", g.Wait())
}

type Do struct {
	i int
}

func hi(g *gogroup.Group) {
	key := g.Register()
	defer g.Unregister(key)
	for {
		select {
		case <-g.Ctx.Done():
			return
		case <-time.After(1 * time.Second):
			log.Println("hi")
		}
	}
}

func (i *Do) Run(g *gogroup.Group) {
	defer log.Println(i.i, "done")
	ct := 0
	for {
		select {
		case <-g.Ctx.Done():
			return
		case t := <-time.After(1 * time.Second):
			ct++
			log.Println(i.i, t.Format("3:04:05 pm"), ct)
			if i.i == 8 {
				g.Set_err(fmt.Errorf("some err"))
			}
		}
	}
}
```


#### License 

Use of this source code is governed by a BSD-2-Clause license that can be
found in the LICENSE file.

[![BSD-2-Clause License](img/osi_logo_100X133_90ppi_0.png)](https://opensource.org/)
