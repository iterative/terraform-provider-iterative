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


// func mayPanic() {
//     panic("a problem")
// }

// func main() {

//     defer func() {
//         if r := recover(); r != nil {
//             fmt.Println("Recovered. Error:\n", r)
//         }
//     }()
//     mayPanic()

//     fmt.Println("After mayPanic()")
// }