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

var done = make(chan struct{})

type directory struct {
  name string
  size int64
}
// du  is a program that gets a list of directories from the command line
// and traverse each one of them to return their total file size
func main() {
	flag.Parse()
	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"."}
	}
  dirs := map[string]int64{
    "." : 0,
  }
  fileSizes := make(chan directory)
  var n sync.WaitGroup
  for _, dir := range roots {
    dirs[dir] = 0
    n.Add(1)
    go parseDir(dir, dir, &n, fileSizes)
  }
  go func() {
    os.Stdin.Read(make([]byte,1))
    close(done)
  }()
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
    case <-done:
      for range fileSizes {
        // do nothing
      }
      return
    case <-tick:
      fmt.Printf("number of files: %d counted, size: %.1f GB\n", nFiles, float64(filesize)/1e9)
    case dirent, ok := <- fileSizes:
      if !ok {
        break loop // fileSizes was closed
      }
      dirname := dirent.name
      dirs[dirname] += dirent.size
      nFiles++
      filesize += dirent.size
    }
  }
  fmt.Printf("number of files: %d total, size: %.1f GB\n", nFiles, float64(filesize)/1e9)
  for n, s := range dirs {
    fmt.Printf("dir: %s, size: %.1f GB\n", n, float64(s)/1e9)
  }
}

func cancelled() bool {
  select {
  case <-done:
    return true
  default:
    return false
  }
}

func parseDir(maindir string, dir string, n *sync.WaitGroup, fileSizes chan directory) {
  defer n.Done()
  if cancelled() {
    return
  }
  for _, entry := range dirents(dir) {
    if entry.IsDir() {
      n.Add(1)
      subdir := filepath.Join(dir, entry.Name())
      parseDir(maindir, subdir, n, fileSizes)
    } else {
      var direntry directory
      direntry.name = maindir
      direntry.size = entry.Size()
      fileSizes <-direntry
    }
  }
}

var sema = make(chan struct{}, 20) // concurrency-limiting counting semaphore

// dirent return entries of directory dir
func dirents (dir string) []os.FileInfo {
  select {
	case sema <- struct{}{}: // acquire token
	case <-done:
		return nil // cancelled
	}
	defer func() { <-sema }() // release token
  entries, err := ioutil.ReadDir(dir)
  if err != nil {
    fmt.Fprintf(os.Stderr, "du1: %v\n", err)
    return nil
  }
  return entries
}
