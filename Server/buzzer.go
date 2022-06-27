/* Functions for communicating with physical buzzers.

*/

package main

import "fmt"
import "net"


// External interface.

// Create a Buzzer object based on the given connection and start processing incoming messages.
func HandleNode(conn net.Conn, controller *Controller, swarm *Swarm) {
    var p Buzzer
    p.conn = conn
    p.controller = controller
    p.swarm = swarm
    p.id = 0xFF
    p.sends = make(chan []byte, 100)

    // Since all messages are single bytes, we only read 1 byte at a time from our connection.
    p.buffer = make([]byte, 1)

    go p.processIncoming()
    go p.processOutgoing()
}


// Send a mode message to this Buzzer.
// This may be slow, call as a Go routine if appropriate.
func (this *Buzzer) SetMode(ledOn bool, buzzerOn bool) {
    var b byte = 0x20

    if ledOn { b |= 1 }
    if buzzerOn { b |= 2 }

    // fmt.Printf("Set buzzer %s mode %x\n", this.ID(), b)
    this.sends <- []byte{b}
}


// Disconnect from this buzzer.
func (this *Buzzer) Disconnect() {
    this.conn.Close()
    this.swarm.Disconnected(this.id, this)
}


// Return this buzzer's ID is human readable form.
func (this *Buzzer) ID() string {
    return BuzzerIdToString(this.id)
}


// Convert the given buzzer ID to a string.
func BuzzerIdToString(id int) string {
    team := (id >> 4) & 7
    return fmt.Sprintf("%s%d", _teamLetters[team], id & 15)
}


// Convert the given team ID to a string.
func TeamIdToString(id int) string {
    return _teamLetters[id]
}


// Object to represent a physical buzzer with which we're communicating.
type Buzzer struct {
    conn net.Conn
    controller *Controller
    id int
    swarm *Swarm
    buzzerVersion byte
    buffer []byte  // Storage for incoming messages.
    sends chan []byte  // Bytes to send, which should be synchronised.
}


// Internals.

// We always expect all buzzers contacted to be on the latest firmware version.
const (
    BuzzerExpectedVersion = 4
)

// Team letters for printing buzzer IDs.
// TODO: Use this same definition for command parsing buzzer IDs.
var _teamLetters = []string{"B", "G", "R", "Y", "x", "x", "x", "x"}


// Handle outgoing messages.
// Only returns on connection error. Should be called as a Go routine.
func (this *Buzzer) processOutgoing() {
    // Now process outgoing messages forever.
    for {
        b := <-this.sends
        _, err := this.conn.Write(b)
        if err != nil {
            fmt.Printf("Failure to send mode message to buzzer %d, disconnecting\n", this.id)
            this.Disconnect()
            return
        }
    }
}



// Handles incoming requests.
// Only returns on connection error. Should be called as a Go routine.
func (this *Buzzer) processIncoming() {
    // First get handshake out of the way.
    if !this.processHandshake() { return }

    // Now process incoming messages forever.
    for {
        // Get the next message byte.
        b, ok := this.getMessageByte()
        if !ok { return }

        this.swarm.Received(this.id)
        msg, _ := this.decodeMessage(b)

        switch msg {
        case MsgHeartbeat:
            // Nothing to do for a heartbeat.

        case MsgButtonPress:
            // Button press. This needs to be reported.
            // fmt.Printf("Button press from %s\n", this.ID())
            this.controller.ButtonPress(this.id)

        case MsgError:
            // Error message. This needs to be reported.
            // TODO
            fmt.Printf("Error message received from %s\n", this.ID())

        default:
            fmt.Printf("Unrecognised message 0x%02X received from %s\n", b, this.ID())
        }
    }
}


// Handle the incoming handshake messages from this new connection.
// Returns true on success, false on failure.
func (this *Buzzer) processHandshake() bool {
    // First we need a version byte.
    b, ok := this.getMessageByte()
    if !ok { return false }

    this.swarm.Received(this.id)
    msg, value := this.decodeMessage(b)
    if msg != MsgVersion {
        fmt.Printf("Expected version from new buzzer, got 0x%02X\n", value)
        return false
    }

    this.buzzerVersion = value

    // Next we need an ID.
    b, ok = this.getMessageByte()
    if !ok { return false }

    msg, value = this.decodeMessage(b)
    if msg != MsgId {
        fmt.Printf("Expected ID from new buzzer, got 0x%02X\n", value)
        return false
    }

    this.id = int(value)

    if this.buzzerVersion == BuzzerExpectedVersion {
        fmt.Printf("Found buzzer %s (v:%d)\n", this.ID(), this.buzzerVersion)
    } else {
        fmt.Printf("Found buzzer %s with unexpected version %d\n", this.ID(), this.buzzerVersion)
    }

    this.swarm.NewBuzzer(this.id, this)

    return true
}


// Decode the given received message byte.
func (this *Buzzer) decodeMessage(b byte) (msg MsgTypeEnum, param byte) {
    // Check for known messages.
    switch {
    case b < 0x20:
        // Version message.
        return MsgVersion, b

    case (b & 0x80) == 0x80:
        // ID message.
        id := b & 0x7F
        return MsgId, id

    case b == 0x30:
        // Button press message.
        return MsgButtonPress, 0

    case b == 0x31:
        // Heartbeat.
        return MsgHeartbeat, 0

    case b == 0x7F:
        // Error message.
        return MsgError, 0

    default:
        fmt.Printf("Unrecognised message 0x%02X from buzzer %s\n", b, this.ID())
        return MsgUnknown, b
    }
}

const (
    MsgVersion = iota
    MsgId
    MsgHeartbeat
    MsgButtonPress
    MsgError
    MsgUnknown
)

type MsgTypeEnum int


// Get the next incoming message, waiting until one is received.
func (this *Buzzer) getMessageByte() (b byte, ok bool) {
    // Get the next message byte.
    _, err := this.conn.Read(this.buffer)
    if err != nil {
        fmt.Printf("Failure receiving from %s\n", this.ID())
        this.Disconnect()
        return 0, false
    }

    return this.buffer[0], true
}
