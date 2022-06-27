/* Functions to control quiz state.

When told to ask a question the controller disables outputs on all buzzers and waits for the first button press. Upon
receipt outputs are enabled for the sending buzzer and the buzzer's team is recorded. The user then indicates whether
the answer given was correct or not. If it was correct, the answering team is given a point. If the answer was wrong,
the buzzers are asked again, but the team that gave the incorrect answer may not answer again. If a second team
answers incorrectly, they also may not answer, and so on.

*/

package main

import "fmt"


// Create a scoreboard.
func CreateController(cmdProc *CommandProcessor, scoreboard *Scoreboard) *Controller {
    var p Controller
    p.state = ConStIdle
    p.scoreboard = scoreboard
    p.requests = make(chan func(), 1000)

    cmdProc.AddCommand(p.commandIdle, "Enter idle mode", "idle")
    cmdProc.AddCommand(p.commandTest, "Enter test mode", "test")
    cmdProc.AddCommand(p.commandAskNoDouble, "Ask a question with no double marks", "qn")
    cmdProc.AddCommand(p.commandAsk, "Ask a question with double marks for the specified team", "q", LEX_TEAM)
    cmdProc.AddCommand(p.commandCorrect, "The last answer given was correct", "y")
    cmdProc.AddCommand(p.commandIncorrect, "The last answer given was wrong", "n")

    return &p
}


// Set the swarm for this controller and start processing.
func (this *Controller) Run(swarm *Swarm) {
    this.swarm = swarm
    go this.run()
}


// Receive a button press from the specified buzzer.
func (this *Controller) ButtonPress(buzzerId int) {
    this.requests <- func() {
        // What we do depends on our current state.
        switch this.state {
        case ConStTest:
            this.testPress(buzzerId)

        case ConStAsked:
            this.recvAnswer(buzzerId)

        default:
            // In all other modes we can ignore button presses.
        }
    }
}


// Quiz controller.
type Controller struct {
    state ConStTypeEnum
    testState map[int]bool  // Buzzer state when in test mode. Buzzer ID => on state.
    swarm *Swarm
    scoreboard *Scoreboard
    doubleTeam int  // The ID of the team that scores double for the current question. <0 for none.
    lastAnswerTeam int  // ID of the team that last answered a question.
    teamsAllowed []bool  // Whether each team is allowed to answer. Indexed by team ID.
    presses chan int  // BUtton presses received from buzzers. Value sent is buzzer ID.
    requests chan func()  // All requests are handling in the central Go routine.
}


// Internals.

const (
    ConStIdle = iota  // Doing nothing.
    ConStTest  // Testing buzzers.
    ConStAsked  // Question has been asked, waiting for an answer.
    ConStAnswered  // Answer has been given.
)

type ConStTypeEnum int


// Handles requests in a single thread.
// Never returns. Should be called as a Go routine.
func (this *Controller) run() {
    // Process incoming messages forever.
    for {
        select {
        case request := <-this.requests:
            request()
        }
    }
}


// Transition to the specified state.
func (this *Controller) changeState(newState ConStTypeEnum) {
    // We always need to disable outputs for all buzzers.


    // What to do depends on the state we're going into.
    switch newState {
    case ConStIdle:
        fmt.Printf("Idle mode\n")
        this.swarm.SetModeAll(false, false)

    case ConStTest:
        // Reset buzzer states.
        fmt.Printf("Test mode\n")
        this.testState = make(map[int]bool)
        this.swarm.SetModeAll(false, false)

    case ConStAsked:
        fmt.Printf("Waiting for button answer\n")
        this.swarm.SetModeAll(false, false)

    case ConStAnswered:
        // Nothing to do.

    default:
        // Nothing to do in any other states.
    }

    this.state = newState
}


// Handle a button press in test mode.
func (this *Controller) testPress(buzzerId int) {
    // We toggle outputs on/off with each button press.
    // Find current state. No record counts as off.
    on, _ := this.testState[buzzerId]

    // Toggle buzzer outputs.
    newState := !on
    this.swarm.SetMode(buzzerId, newState, newState)

    // Record state for next time.
    this.testState[buzzerId] = newState
}


// Handle a button press in response to a question.
func (this *Controller) recvAnswer(buzzerId int) {
    // Check if the buzzer's team is allowed to answer.
    team := buzzerId >> 4

    if !this.teamsAllowed[team] {
        // Team is not allowed to answer, ignore press.
        return
    }

    this.lastAnswerTeam = team

    // Turn on just that one buzzer.
    this.changeState(ConStAnswered)
    this.swarm.SetMode(buzzerId, true, true)

    fmt.Printf("Answer from %s\n", BuzzerIdToString(buzzerId))
}


// Command handler for entering idle mode.
// May be called from any thread context.
func (this *Controller) commandIdle(value ...int) {
    this.requests <- func() {
        this.changeState(ConStIdle)
    }
}


// Command handler for entering test mode.
// May be called from any thread context.
func (this *Controller) commandTest(value ...int) {
    this.requests <- func() {
        this.changeState(ConStTest)
    }
}


// Command handler for entering ask question mode.
// May be called from any thread context.
func (this *Controller) commandAsk(value ...int) {
    this.requests <- func() {
        this.changeState(ConStAsked)
        this.doubleTeam = value[0]
        this.teamsAllowed = []bool{true, true, true, true, false, false, false, false}
    }
}


// Command handler for entering ask question mode with no double team.
// May be called from any thread context.
func (this *Controller) commandAskNoDouble(value ...int) {
    this.requests <- func() {
        this.changeState(ConStAsked)
        this.doubleTeam = -1
        this.teamsAllowed = []bool{true, true, true, true, false, false, false, false}
    }
}


// Command handler for reporting a correct answer.
// May be called from any thread context.
func (this *Controller) commandCorrect(value ...int) {
    this.requests <- func() {
        if this.doubleTeam == this.lastAnswerTeam {
            // Double marks.
            fmt.Printf("Double marks to %s\n", TeamIdToString(this.lastAnswerTeam))
            this.scoreboard.Add(this.lastAnswerTeam, 2)
        } else {
            // Normal marks.
            fmt.Printf("1 mark to %s\n", TeamIdToString(this.lastAnswerTeam))
            this.scoreboard.Add(this.lastAnswerTeam, 1)
        }
    }
}


// Command handler for reporting an incorrect answer.
// May be called from any thread context.
func (this *Controller) commandIncorrect(value ...int) {
    this.requests <- func() {
        // We ask again, with the answering team disabled.
        this.teamsAllowed[this.lastAnswerTeam] = false
        this.changeState(ConStAsked)
    }
}
