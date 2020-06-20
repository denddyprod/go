package main

import (
	"fmt"
)

type List struct {
	data int
	next *List
}

func (l *List) add(data int) {
	v := &List{data : data,}

	for l.next != nil {
		l = l.next
	}

	l.next = v

}

func (l *List) printList() {
	for l != nil {
		fmt.Println(l.data)
		l = l.next;
	}
}

func main() {
	l := List{1, nil}
	l.add(4)
	l.add(4)
	l.printList()
}
