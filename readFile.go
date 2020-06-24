package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"log"
	"strings"
	"reflect"
)

func main() {
	inputpath := os.Args[1]
	
	// Simplest way to Read
	data, err := ioutil.ReadFile(inputpath)
	if err != nil {
		log.Fatalf("Read failed with '%s' \n", err)
	}

	fmt.Printf("Readed '%d' bytes from '%s':\n\"%s\"\n", len(data), inputpath, data)

	pathsplit := strings.Split(inputpath, ".")
	outputpath := fmt.Sprintf("%s2.txt", pathsplit[0])

	f, err := os.Create(outputpath)
	if err != nil {
		log.Fatalf("Open failed with '%s'\n", err)
	}
	defer f.Close()
	
	nWritten, err := f.Write(data)
	if err != nil {
		log.Fatalf("Write failed with '%s'\n", err)
	}

	fmt.Printf("written %d bytes in %s\n", nWritten, outputpath)

	st, err := os.Lstat(inputpath)
	if err != nil {
		log.Fatalf("Get file size failed with '%s'\n", err)
	}
	fmt.Printf("Size of '%s' - %d bytes\n",inputpath, st.Size())

	st, err = os.Lstat(outputpath)
	if err != nil {
		log.Fatalf("Get file size failed with '%s'\n", err)
	}
	fmt.Printf("Size of '%s' - %d bytes\n",outputpath, st.Size())
	
	fmt.Printf("Variabile 'st' is type of : %s\n", reflect.TypeOf(st))
}
