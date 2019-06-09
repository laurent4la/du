package main

import (
	"flag"
  "fmt"
  "os"
  "sync"
  "time"
  "io/ioutil"
  "path/filepath"
)

var verbose = flag.Bool("v", false, "show verbose progress message")
// du  is a program that gets a list of directories from the command line
// and traverse each one of them to return their total file size
func main() {
	flag.Parse()
	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}
  fileSizes := make(chan int64)
  var n sync.WaitGroup
  for _, dir := range roots {
    n.Add(1)
    go parseDir(dir, &n, fileSizes)
  }
  go func() {
    n.Wait()
    close(fileSizes)
  }()
  var tick <-chan time.Time
  if *verbose {
    tick = time.Tick(500 * time.Millisecond)
  }
  var nFiles, filesize int64
loop:
  for {
    select {
    case <-tick:
      fmt.Printf("number of files: %d, size: %.1f GB\n", nFiles, float64(filesize)/1e9)
    case size, ok := <- fileSizes:
      if !ok {
        break loop // fileSizes was closed
      }
      nFiles++
      filesize += size
    }
  }
  fmt.Printf("number of files: %d, size: %.1f GB\n", nFiles, float64(filesize)/1e9)
}

func parseDir(dir string, n *sync.WaitGroup, fileSizes chan int64) {
  defer n.Done()
  for _, entry := range dirents(dir) {
    if entry.IsDir() {
      n.Add(1)
      subdir := filepath.Join(dir, entry.Name())
      parseDir(subdir, n, fileSizes)
    } else {
      fileSizes <-entry.Size()
    }
  }
}

// dirent return entries of directory dir
func dirents (dir string) []os.FileInfo {
  entries, err := ioutil.ReadDir(dir)
  if err != nil {
    fmt.Fprintf(os.Stderr, "du1: %v\n", err)
    return nil
  }
  return entries
}
