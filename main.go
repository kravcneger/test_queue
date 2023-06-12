package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"
	"sync"
	"time"
)

type Response struct {
	Body   string `json:"body"`
	Status int    `json:"status"`
}

var mQ *multipleQueue

func putToQueue(r *http.Request) *Response {
	qName := path.Base(r.Host)
	val := r.URL.Query().Get("v")
	if val == "" {
		return &Response{
			Body:   "Bad request",
			Status: 400,
		}
	}
	mQ.Push(qName, val)
	return &Response{
		Body:   "",
		Status: 200,
	}
}

func deQueue(r *http.Request) *Response {
	qName := path.Base(r.Host)
	if qName == "" {
		return &Response{
			Body:   "",
			Status: 400,
		}
	}
	timeout := r.URL.Query().Get("timeout")
	var delay time.Duration
	if timeout != "" {
		num, err := strconv.Atoi(timeout)
		if err != nil {
			return &Response{
				Body:   "",
				Status: 400,
			}
		}
		delay = time.Duration(num)
	}
	if delay != 0 {
		ch := make(chan *Element)
		go func() {
			ch <- mQ.Dequeue(qName)
		}()
		select {
		case <-time.After(delay * time.Second):
			return &Response{
				Body:   "",
				Status: 404,
			}
		case elem := <-ch:
			return &Response{
				Body:   elem.value,
				Status: 200,
			}
		}

	} else {

		el := mQ.Dequeue(qName)
		return &Response{
			Body:   el.value,
			Status: 200,
		}
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	var response *Response
	switch r.Method {
	case http.MethodPut:
		response = putToQueue(r)
	case http.MethodGet:
		response = deQueue(r)
	}

	json, err := json.Marshal(response)
	if err != nil {
		fmt.Println(err)
		return
	}
	w.Write(json)
}

func main() {
	mQ = NewMultipleQueue()
	http.HandleFunc("/", handle)

	port := flag.String("port", "80", "pass port number")
	flag.Parse()

	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

type Element struct {
	value string
	next  *Element
}

type multipleQueue struct {
	queues map[string]chan *Element
	mu     sync.Mutex
}

func NewMultipleQueue() *multipleQueue {
	return &multipleQueue{
		queues: map[string]chan *Element{},
	}
}

func (mq *multipleQueue) Push(key string, value string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	element := &Element{}
	element.value = value
	if !mq.isExist(key) {
		mq.queues[key] = make(chan *Element, 1)
		mq.queues[key] <- element
		return
	}
	if len(mq.queues[key]) == 0 {
		mq.queues[key] <- element
	} else {
		head := <-mq.queues[key]

		last := head
		for last.next != nil {
			last = last.next
		}
		last.next = element
		mq.queues[key] <- head
	}
}

func (mq *multipleQueue) Dequeue(key string) *Element {
	mq.mu.Lock()

	if !mq.isExist(key) {
		mq.queues[key] = make(chan *Element, 1)
	}

	if len(mq.queues[key]) == 1 {
		head := <-mq.queues[key]
		if head.next != nil {
			mq.queues[key] <- head.next
		}
		mq.mu.Unlock()
		return head
	}
	mq.mu.Unlock()
	return <-mq.queues[key]
}

func (mq *multipleQueue) isExist(name string) bool {
	_, ok := mq.queues[name]
	return ok
}
