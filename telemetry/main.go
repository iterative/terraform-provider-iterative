package main

import (
    "fmt"
    "os"
)





func main() {
    // TODO: save file with user UUID
    if os.Getenv("TPI_NO_ANALYTICS") == "" && os.Getenv("TPI_TEST") == "" {
        fmt.Println(SendEvent(NewEvent()))
    } 
}


// https://pkg.go.dev/github.com/hashicorp/go-retryablehttp
