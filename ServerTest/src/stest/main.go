package main

import "fmt"
import "net"


func handleConnection(conn net.Conn) {
    buf := make([]byte, 1)
    for {
        n, err := conn.Read(buf)

        if n > 0 {
            fmt.Printf("%02x\n", buf[0])
        }

        if err != nil {
            fmt.Printf("Error: %v\n", err)
            return
        }
    }
}


func main() {
    fmt.Printf("Fuck off\n")

    netListener, err := net.Listen("tcp", ":9753")
    if err != nil {
        fmt.Printf("Failed to listen: %v\n", err)
        return
    }

    fmt.Printf("Listening on %s\n", netListener.Addr())

    for {
        fmt.Printf("Accepting\n")
        conn, err := netListener.Accept()
        if err != nil {
            fmt.Printf("Failed to accept: %v\n", err)
        } else {
            fmt.Printf("We got one\n")
            go handleConnection(conn)
        }
    }
}
