package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"
	"io/ioutil"
)

type Item struct {
	ID    int
	Name string
	Pickup_Time int
	Packaging_Time int
}

func PickerItems(done <-chan bool) <-chan Item {
	items := make(chan Item)
	var itemsTolookup []Item
	data, _ := ioutil.ReadFile("package.json")
	json.Unmarshal(data, &itemsTolookup)
	go func() {
		for _, item := range itemsTolookup {
			duration := time.Duration(item.Pickup_Time)*time.Second
			time.Sleep(duration);
			select {
			case <-done:
				return
			case items <- item:
			}
		}
		close(items)
	}()
	return items
}

func PackagerItems(done <-chan bool, items <-chan Item, workerID int) <-chan int {
	itemss := make(chan int)
	go func() {
		for item := range items {
			select {
			case <-done:
				return
			case itemss <- item.ID:
				duration := time.Duration(item.Packaging_Time)*time.Second
				time.Sleep(duration)
				fmt.Printf("Worker #%d: Name %s  PickupTime %d \n", workerID, item.Name,item.Packaging_Time)
			}
		}
		close(itemss)
	}()
	return itemss
}

func merge(done <-chan bool, channels ...<-chan int) <-chan int {
	var wg sync.WaitGroup

	wg.Add(len(channels))
	outgoingItems := make(chan int)
	multiplex := func(c <-chan int) {
		defer wg.Done()
		for i := range c {
			select {
			case <-done:
				return
			case outgoingItems <- i:
			}
		}
	}
	for _, c := range channels {
		go multiplex(c)
	}
	go func() {
		wg.Wait()
		close(outgoingItems)
	}()
	return outgoingItems
}

func main() {
	cpu := runtime.NumCPU()
	runtime.GOMAXPROCS(cpu)

	done := make(chan bool)
	defer close(done)

	start := time.Now()
	items := PickerItems(done)

	workers := make([]<-chan int, 2)
	for i := 0; i < 2; i++ {
		workers[i] = PackagerItems(done, items, i)
	}

	numItems := 0
	for range merge(done, workers...) {
		numItems++
	}
	elapsed := time.Since(start)
	fmt.Printf("Took %s to Shipped %d Items\n", elapsed, numItems)
}