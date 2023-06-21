/* Initialisation.

Ready
> cont
Q1 ...
<wait>
Buzzer
Pressed by B4
> yes
Update score
Select bonus topic from:
1. Foo
2. Bar
3. Wibble
4. Sport
> 3
Q1B ...
<wait>
Pressed: B-A G-? R-? Y-?
<wait>
Pressed: B-A G-? R-C Y-?
<wait>
Pressed: B-A G-? R-C Y-A
<wait>
Pressed: B-A G-A R-C Y-A
Correct answer ...
Answer: B-2 G-1 R-X Y-1
Update score
> cont
Q2 ...
Buzzer
<wait>
Pressed by B4
> no
<wait>
Pressed by R3
> no
<wait>
> give-up
Correct answer ...
> cont
Q3 ...


*/

package main

import "fmt"
import "net"
import "os"


func main() {
    engine, swarm := CreateEngine()
    scoreboard := CreateScoreboard(engine)
    scoreboard.Print()

    CreateTestMode(engine)
    CreateMultipleChoice(engine, scoreboard)
    CreateQuickFire(engine, scoreboard)

    go listen(swarm)

    engine.Run()
}


func listen(swarm *Swarm) {
    // Listen for incoming connections.
    listener, err := net.Listen("tcp", ":9753")
    if err != nil {
        fmt.Println("Error listening:", err.Error())
        os.Exit(1)
    }

    // Close the listener when the application closes.
    defer listener.Close()
    fmt.Printf("Listening for buzzers\n")

    for {
        // Listen for an incoming connection.
        conn, err := listener.Accept()
        if err != nil {
            fmt.Println("Error accepting: ", err.Error())
            listener.Close()
            return
        }

        // Handle connections in a new goroutine.
        HandleNode(conn, swarm)
    }
}
