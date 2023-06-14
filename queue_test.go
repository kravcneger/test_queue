package main

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestPushAndWaitDequeue(t *testing.T) {
	var wg sync.WaitGroup
	mQ := NewMultipleQueue()
	result := []string{}
	expected := []string{"val1", "val2", "val3", "val4"}
	wg.Add(2)

	go func() {

		for i := 0; i < 4; i++ {
			fmt.Println("Dequeue", i)
			e := mQ.WaitDequeue("q1")
			result = append(result, e.value)
		}
		wg.Done()
	}()

	go func() {
		time.Sleep(200 * time.Millisecond)
		for i := 0; i < 4; i++ {
			fmt.Println("push", i)
			mQ.Push("q1", "val"+strconv.Itoa(i+1))
		}
		wg.Done()
	}()

	wg.Wait()

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Elements should be %s instead %s", expected, result)
	}

	mQ.Push("q1", "val1")
	mQ.Push("q1", "val2")

	elem := mQ.WaitDequeue("q1")
	if elem.value != "val1" {
		t.Errorf("Element should be %s instead %s", "val1", elem.value)
	}

	elem = mQ.WaitDequeue("q1")
	if elem.value != "val2" {
		t.Errorf("Element should be %s instead %s", "val2", elem.value)
	}
}

func TestPushAndDequeue(t *testing.T) {

	mQ := NewMultipleQueue()
	elem := mQ.Dequeue("q1")
	if elem != nil {
		t.Errorf("Element should nil")
	}

	mQ.Push("q1", "val1")
	mQ.Push("q1", "val2")
	elem = mQ.Dequeue("q1")
	if elem.value != "val1" {
		t.Errorf("Element should val1")
	}

	elem = mQ.Dequeue("q1")
	if elem.value != "val2" {
		t.Errorf("Element should val2")
	}

	elem = mQ.Dequeue("q1")
	if elem != nil {
		t.Errorf("Element should nil")
	}
}

func TestNewRuquestsQueue(t *testing.T) {
	rQ := NewRuquestsQueue()
	ch1 := rQ.GetChannel("q1")
	ch2 := rQ.GetChannel("q1")
	ch3 := rQ.GetChannel("q1")
	ch4 := rQ.GetChannel("q1")
	ch5 := rQ.GetChannel("q1")

	go rQ.PushFirst("q1", "v1")
	go rQ.PushFirst("q1", "v2")
	go rQ.PushFirst("q1", "v3")
	go rQ.PushFirst("q1", "v4")
	go rQ.PushFirst("q1", "v5")
	for {
		select {
		case <-ch1:
			fmt.Println("case1")
		case <-ch2:
			fmt.Println("case2")
		case <-ch3:
			fmt.Println("case3")
		case <-ch4:
			fmt.Println("case4")
		case <-ch5:
			fmt.Println("case5")
			return
		}

	}
}
