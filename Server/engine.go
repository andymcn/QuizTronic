/* Functions to run central system engine.

The engine receives asynchronous button press messages and commands from stdin, and forwards them to entities that
have registered interest in them. It also provides an access point for those entities to affect the state of the
buzzers.

All engine functions and methods must be called only in the main thread, unless otherwise stated.

*/

package main

import "bufio"
import "fmt"
import "os"
import "sort"
import "strings"


// Create the engine and associated swarm.
func CreateEngine() (*Engine, *Swarm) {
    var p Engine
    p.rawCmdLines = make(chan string, 10)
    p.pressIds = make(chan int, 100)
    p.commands = make(map[byte]*cmdInfo)

    swarm := CreateSwarm(&p)
    p.swarm = swarm

    p.RegisterCmd(p.usage, "Help", '?')

    return &p, swarm
}


// Start processing requests.
// Only returns on program exit.
func (this *Engine) Run() {
    // Start inputting command lines from stdin.
    go this.processStdin()

    // Process incoming messages until exit.
    for {
        select {
        case cmd := <-this.rawCmdLines:
            // Command line received.
            if cmd == ExitCommand {
                // Quit command given.
                return
            }

            this.processCommand(cmd)

        case buttonId := <-this.pressIds:
            // A button has been pressed.
            if this.buttonHandler != nil {
                // Tell our registered handler about it.
                this.buttonHandler(buttonId)
            }
        }
    }
}


// Register the given command handler.
// The command is specified as a single leading character of the command line. There can only ever be one handler for
// and given command character at a time.
// All command handler callbacks will occur within the main engine thread.
func (this *Engine) RegisterCmd(handler CmdHandler, help string, cmd byte, args ...ArgType) {
    _, ok := this.commands[cmd]
    if ok {
        fmt.Printf("Error: Request to register already registered command %v\n", cmd)
    }

    var p cmdInfo
    p.handler = handler
    p.helpText = help
    p.initialChar = cmd
    p.argTypes = args
    this.commands[cmd] = &p
}

// Function to handle a specific command.
type CmdHandler func (argValues []int)


// Deregister the given, previously registered command handler.
func (this *Engine) DeregisterCmd(handler CmdHandler, cmd byte) {
    _, ok := this.commands[cmd]
    if !ok {
        fmt.Printf("Error: Request to deregister undefined command %v\n", cmd)
        return
    }

    delete(this.commands, cmd)
}


// Register the given button press handler.
// There can only be a single receiver registered at a time.
// All button press handler callbacks will occur within the main engine thread.
func (this *Engine) RegisterButtons(handler ButtonHandler) {
    if this.buttonHandler != nil {
        fmt.Printf("Error: Clashing button handler. Have %v, want to reg %v\n",
            this.buttonHandler, handler)
    }

    this.buttonHandler = handler
}

// Function to handle button press events.
type ButtonHandler func (id int)


// Deregister the given, previously registered button press handler.
func (this *Engine) DeregisterButtons(handler ButtonHandler) {
    this.buttonHandler = nil
}


// Send a mode message to the specified buzzer.
// Returns false if the specified buzzer cannot be found.
func (this *Engine) SetMode(buzzerId int, ledOn bool, buzzerOn bool) bool {
    // Just forward to our Swarm.
    return this.swarm.SetMode(buzzerId, ledOn, buzzerOn)
}


// Send a mode message to all connected buzzers.
func (this *Engine) SetModeAll(ledOn bool, buzzerOn bool) {
    // Just forward to our Swarm.
    this.swarm.SetModeAll(ledOn, buzzerOn)
}


// Handle a button press event from the specified buzzer.
// May be called from any thread.
func (this *Engine) ButtonPress(buzzerId int) {
    // Just add the button ID to our incoming list.
    this.pressIds <- buzzerId
}


// Quiz engine.
type Engine struct {
    rawCmdLines chan string
    pressIds chan int  // Button ID for each press event.
    buttonHandler ButtonHandler
    swarm *Swarm
    commands map[byte]*cmdInfo  // Indexed by leading char.
}

// Info needed for a single command.
type cmdInfo struct {
    handler CmdHandler
    helpText string
    initialChar byte
    argTypes []ArgType
}


// Internals.

const (
    ExitCommand string = "quit"
)


// Parse the given command line and call the registered handler.
func (this *Engine) processCommand(cmdLine string) {
    // We identify the command by the leading character.
    cmdChar := ParseUserCmd(cmdLine)

    cmd, ok := this.commands[cmdChar]
    if !ok {
        fmt.Printf("Unrecognised command, ? for help: %s\n", cmdLine)
        return
    }

    argValues, ok := ParseUserArgs(cmdLine, cmd.argTypes)
    if !ok {
        // Error has already been reported.
        return
    }
    // TODO: Parse args.
    cmd.handler(argValues)
}


// Read stdin and report all resulting command lines to the main thread.
// Never returns. Should be called as a Go routine.
func (this *Engine) processStdin() {
    stdin := bufio.NewReader(os.Stdin)

    for {
        text, _ := stdin.ReadString('\n')
        text = strings.TrimSpace(text)

        // Ignore blank lines.
        if text != "" {
            this.rawCmdLines <- text
        }
    }
}


// Print a usage message for our commands.
func (this *Engine) usage([]int) {
    fmt.Printf("Usage:\n")
    fmt.Printf("  %-16s  Exit\n", ExitCommand)

    // Before printing commands, sort by command char.
    keys := make([]byte, 0, len(this.commands))
    for key := range this.commands {
        keys = append(keys, key)
    }

    sort.Slice(keys, func(i, j int) bool {
        return keys[i] < keys[j]
    })

    // Now we can print our commands.
    for _, key := range keys {
        cmd := this.commands[key]

        // Get usage info for arguments, if any.
        args := ArgUsage(cmd.argTypes)

        fmt.Printf("  %c%-15s  %s\n", cmd.initialChar, args, cmd.helpText)
    }
}
