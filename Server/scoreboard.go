/* Functions to track quiz scores.

*/

package main

import "fmt"


// Create a scoreboard.
func CreateScoreboard(cmdProc *CommandProcessor) *Scoreboard {
    var p Scoreboard
    p.scores = make([]int, 4)
    p.requests = make(chan func(), 1000)

    go p.run()

    cmdProc.AddCommand(p.commandAdd, "Give points to a team", "+", LEX_TEAM, LEX_UINT)
    cmdProc.AddCommand(p.commandSub, "Deduct points from a team", "-", LEX_TEAM, LEX_UINT)
    cmdProc.AddCommand(p.commandScore, "Show team scores", "score")

    return &p
}


// Add points to the specified team.
func (this *Scoreboard) Add(team int, points int) {
    this.requests <- func() {
        this.scores[team] += points
        this.printLocal()
    }
}


// Print out the current scores.
func (this *Scoreboard) Print() {
    this.requests <- func() {
        this.printLocal()
    }
}


// Scoreboard object.
type Scoreboard struct {
    scores []int
    requests chan func()  // All requests are handling in the central Go routine.
}


// Internals.

// Handles requests in a single thread.
// Never returns. Should be called as a Go routine.
func (this *Scoreboard) run() {
    // Process incoming messages forever.
    for {
        request := <-this.requests
        request()
    }
}


// Print out the current scores.
// Must only be called from our central thread.
func (this *Scoreboard) printLocal() {
    fmt.Printf("Scores:\n")
    fmt.Printf("B: %3d\n", this.scores[0])
    fmt.Printf("G: %3d\n", this.scores[1])
    fmt.Printf("R: %3d\n", this.scores[2])
    fmt.Printf("Y: %3d\n", this.scores[3])
}


// Command handler for adding points to the specified team.
func (this *Scoreboard) commandAdd(value ...int) {
    this.Add(value[0], value[1])
}


// Command handler for subtracting points from the specified team.
func (this *Scoreboard) commandSub(value ...int) {
    this.Add(value[0], -value[1])
}


// Command handler for printing scores.
func (this *Scoreboard) commandScore(value ...int) {
    this.Print()
}
