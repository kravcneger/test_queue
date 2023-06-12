package main

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestPushAndDequeue(t *testing.T) {
	var wg sync.WaitGroup
	mQ := NewMultipleQueue()
	result := []string{}
	expected := []string{"val1", "val2", "val3", "val4"}
	wg.Add(2)

	go func() {

		for i := 0; i < 4; i++ {
			fmt.Println("Dequeue", i)
			e := mQ.Dequeue("q1")
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

	elem := mQ.Dequeue("q1")
	if elem.value != "val1" {
		t.Errorf("Element should be %s instead %s", "val1", elem.value)
	}

	elem = mQ.Dequeue("q1")
	if elem.value != "val2" {
		t.Errorf("Element should be %s instead %s", "val2", elem.value)
	}
}
