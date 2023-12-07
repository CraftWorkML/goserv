package main

import (
	"fmt"
	"sync"
	"time"
)

/**/

func diffTypes() {
	values := []any{5, "hello", 3.14}
	for _, v := range values {
		fmt.Println(v)
		fmt.Printf("var1 = %T\n", v)
	}
}

func sliceCopying() {
	src := []int{0, 1, 2}
	var dst []int = make([]int, 3, 5)
	copy(dst, src)
	fmt.Println(dst)

}

func closure() {
	var funcs = make([]func(), 0, 5)

	for i := 0; i < 5; i++ {
		funcs = append(funcs, func() { println("counter =", i) })

		// исправляется так:
		//var value = i
		//funcs = append(funcs, func() { println("counter =", value) })
	}

	for _, f := range funcs {
		f()
	}
}

func syncer() {
	var (
		c  = sync.NewCond(&sync.Mutex{})
		wg sync.WaitGroup // нужна только для примера

		free = true
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.L.Lock()

		for !free { // проверяем, что ресурс свободен
			c.Wait()
		}
		fmt.Println("work")

		c.L.Unlock()
	}()

	free = false                  // забрали ресурс, чтобы выполнить с ним работу
	<-time.After(1 * time.Second) // эмуляция работы
	free = true                   // освободили ресурс
	c.Signal()                    // оповестили горутину

	wg.Wait()
}

func slice1() {
	a := [5]int{1, 2, 3, 4, 5}
	t := a[3:4:4]

	fmt.Println(t[0])
}

func main() {
	diffTypes()
}
