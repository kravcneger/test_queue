package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"sync"
	"time"
)

////////////////////////////////////////////////////////////////
////////// Server section //////////////////////////////////////
//////////////////////////////////////////////////////////////

type Response struct {
	Body   string `json:"body"`
	Status int    `json:"status"`
}

var mQ *multipleQueue
var rQ *RuquestsQueue

func getQueueName(r *http.Request) string {
	uri, _ := url.Parse(r.RequestURI)
	return path.Base(uri.Path)
}

func putToQueue(r *http.Request) *Response {
	qName := getQueueName(r)

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
	qName := getQueueName(r)

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
		ch := rQ.GetChannel(qName)
		go rQ.PushFirst(qName, mQ.WaitDequeue(qName).value)

		select {
		case <-time.After(delay * time.Second):
			return &Response{
				Body:   "",
				Status: 404,
			}
		case val := <-ch:
			return &Response{
				Body:   val,
				Status: 200,
			}
		}

	} else {
		el := mQ.Dequeue(qName)
		if el == nil {
			return &Response{
				Body:   "",
				Status: 404,
			}
		} else {
			return &Response{
				Body:   el.value,
				Status: 200,
			}
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
	rQ = NewRuquestsQueue()
	http.HandleFunc("/", handle)

	port := flag.String("port", "80", "pass port number")
	flag.Parse()

	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

///////////////////////////////////////////////////////////
/////////// Ruquests queue ///////////////////////////////////
///////////////////////////////////////////////////////////

type request struct {
	ch   chan string
	next *request
}

type RuquestsQueue struct {
	queues map[string]*request
	mu     sync.Mutex
}

func NewRuquestsQueue() *RuquestsQueue {
	return &RuquestsQueue{
		queues: map[string]*request{},
	}
}

func (q *RuquestsQueue) GetChannel(nameQueue string) chan string {
	q.mu.Lock()
	defer q.mu.Unlock()

	req := &request{}
	req.ch = make(chan string, 1)

	if _, ok := q.queues[nameQueue]; !ok || q.queues[nameQueue] == nil {
		q.queues[nameQueue] = req

	} else {
		end := q.queues[nameQueue]
		for end.next != nil {
			end = end.next
		}
		end.next = req
	}

	return req.ch
}

func (q *RuquestsQueue) PushFirst(nameQueue string, value string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.queues[nameQueue]; !ok || q.queues[nameQueue] == nil {
		panic("GetChannel colling previously was less than Push First")
	}
	var req *request
	req = q.queues[nameQueue]
	q.queues[nameQueue] = q.queues[nameQueue].next

	req.ch <- value
}

// //////////////////////////////////////////////////////////////
// //////// Queue section //////////////////////////////////////
// ////////////////////////////////////////////////////////////
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

// Wait appearance element in a queue and retrun it
func (mq *multipleQueue) WaitDequeue(key string) *Element {
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

// Return nil if Queue empty
func (mq *multipleQueue) Dequeue(key string) *Element {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	if !mq.isExist(key) {
		return nil
	}

	if len(mq.queues[key]) == 1 {
		head := <-mq.queues[key]
		if head.next != nil {
			mq.queues[key] <- head.next
		}
		return head
	}
	return nil
}

func (mq *multipleQueue) isExist(name string) bool {
	_, ok := mq.queues[name]
	return ok
}
