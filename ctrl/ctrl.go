package main

import (
	//"encoding/json"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"time"
)

func main() {
	c, _, err := zk.Connect([]string{"10.48.57.62:2181"}, time.Second) //*10)
	if err != nil {
		panic(err)
	}
	children, stat, ch, err := c.ChildrenW("/")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v %+v\n", children, stat)
	e := <-ch
	fmt.Printf("%+v\n", e)
}
