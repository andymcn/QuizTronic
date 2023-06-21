package main

import "bufio"
import "fmt"
import "net"
import "os"
import "strconv"
import "time"


func main() {
    id, ok := handleArgs()
    if !ok { return }

    conn := connect()
    if conn == nil { return }

    if !handshake(conn, id) {
        conn.Close()
        return
    }

    go handleRecv(conn)
    go handleHeartbeat(conn)

    handleSend(conn)
}


func handleArgs() (id byte, ok bool) {
    if len(os.Args) != 2 {
        usage(os.Args[0])
        return 0, false
    }

    id_str := os.Args[1]
    id_int, err := strconv.Atoi(id_str)
    if (err != nil) || (id_int < 0) || (id_int > 255) {
        fmt.Printf("Invalid ID \"%s\", should be a byte value\n", id_str)
        usage(os.Args[0])
        return 0, false
    }

    return byte(id_int), true
}


func usage(progName string) {
    fmt.Printf("Usage:\n")
    fmt.Printf("%s <button_id>\n", progName)
}


func connect() *net.TCPConn {
    serverAddr, err := net.ResolveTCPAddr("tcp", "localhost:9753")

    if err != nil {
        fmt.Printf("ResolveTCPAddr failed:", err.Error())
        return nil
    }

    conn, err := net.DialTCP("tcp", nil, serverAddr)
    if err != nil {
        fmt.Printf("Dial failed:", err.Error())
        return nil
    }

    return conn
}


func handshake(conn *net.TCPConn, id byte) bool {
    // First we send the protocol version we're using.
    _, err := conn.Write([]byte{4})
    if err != nil {
        fmt.Printf("Protocol version write failed: %v\n", err)
        return false
    }

    // Next we send our ID.
    msg := 0x80 | id
    _, err = conn.Write([]byte{msg})
    if err != nil {
        fmt.Printf("Button ID write failed: %v\n", err)
        return false
    }

    return true
}


func handleRecv(conn *net.TCPConn) {
    buffer := make([]byte, 1)

    for {
        _, err := conn.Read(buffer)
        if err != nil {
            fmt.Printf("Read failed: %v\n", err)
            return
        }

        b := buffer[0]
        if (b < 0x20) || (b > 0x23) {
            fmt.Printf("Received unexpected %02x\n", b)
        } else {
            led := (b & 1) != 0
            buzzer := (b & 2) != 0
            fmt.Printf("Status led:%v buzzer:%v\n", led, buzzer)
        }
    }
}


func handleHeartbeat(conn *net.TCPConn) {
    for {
        time.Sleep(time.Second)

        // Send heartbeat message.
        _, err := conn.Write([]byte{0x31})
        if err != nil {
            fmt.Printf("Heartbeat write failed: %v\n", err)
        }
    }
}


func handleSend(conn *net.TCPConn) {
    stdin := bufio.NewReader(os.Stdin)

    for {
        stdin.ReadString('\n')

        // Send button press message.
        _, err := conn.Write([]byte{0x30})
        if err != nil {
            fmt.Printf("Button press write failed: %v\n", err)
            return
        }
    }
}
