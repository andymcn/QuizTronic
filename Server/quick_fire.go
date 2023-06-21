/* Functions to handle quick fire questions.

A quick fire controller lives for arbitrarily many questions.

Operation is as follows:
1. When each question starts all of the buzzers are de-illuminated.
2. When the first player presses their button, it is illuminated and buzzers.
3. We wait for input from the user. While waiting, no further illumination changes occur, but we record relevant
   button presses.
4. If the user indicates the first player was correct, that player gets the marks and the question is over.
5. If the user indicates the first player was wrong, that player's team is blocked and we wait for the first player in
   another team to press their button. That next press may have already happened, while we were waiting for the user's
   decision. In that case, we treat the press as if it happened as soon as the user indicated to continue.
6. We continue in this fashion until a player gets the right answer, all teams have had an incorrect guess or the user
   indicates to stop.

All quick fire functions and methods must be called only in the main thread, unless otherwise stated.

*/

package main

import "fmt"


// Create a quick fire controller.
func CreateQuickFire(engine *Engine, scoreboard *Scoreboard) *QuickFire {
    var p QuickFire
    p.engine = engine
    p.scoreboard = scoreboard

    engine.RegisterCmd(p.commandNewQuestion, "Start a quick fire question", 'f', ARG_MARKS)

    return &p
}


// Start a new quick fire question.
func (this *QuickFire) NewQuestion(marks int) {
    this.marks = marks
    this.ackedPlayer = -1
    // TODO: Remove embedded team counts.
    this.haveTeamsBuzzed = make([]bool, 4)
    this.pendingPresses = make([]int, 0, 4)

    // De-illuminate all buzzers.
    this.engine.SetModeAll(false, false)

    // Register for needed inputs for duration of question.
    this.engine.RegisterCmd(this.commandCancel, "Cancel current question", 'q')
    this.engine.RegisterButtons(this.button)
    this.printWaiting()
}


// The last acknowledge player gave the correct answer.
func (this *QuickFire) Correct() {
    if this.ackedPlayer < 0 {
        // This shouldn't be possible, but paranoia is better than a segfault.
        fmt.Printf("Error: No currently acked player\n")
        return
    }

    // Just give the marks to the currently acked player.
    team, _ := BuzzerIdToTeam(this.ackedPlayer)
    this.scoreboard.Add(team, this.marks)
    this.scoreboard.Print()
    fmt.Printf("Player %s won\n", BuzzerIdToString(this.ackedPlayer))

    this.finish()
}


// The last acknowledged player gave the correct answer.
func (this *QuickFire) Incorrect() {
    if this.ackedPlayer < 0 {
        // This shouldn't be possible, but paranoia is better than a segfault.
        fmt.Printf("Error: No currently acked player\n")
        return
    }

    // De-illuminated acked player.
    this.engine.SetMode(this.ackedPlayer, false, false)
    this.ackedPlayer = -1
    this.engine.DeregisterCmd(this.commandCorrect, 'y')
    this.engine.DeregisterCmd(this.commandIncorrect, 'n')

    // Check for any pending presses.
    if len(this.pendingPresses) > 0 {
        newPress := this.pendingPresses[0]
        this.pendingPresses = this.pendingPresses[1:]
        this.handlePress(newPress)
        return
    }

    // We need to wait for the next legal button press.
    this.printWaiting()
}


// Cancel the current question.
func (this *QuickFire) Cancel() {
    // Nothing special to do.
    this.finish()
}


// Quick fire controller.
type QuickFire struct {
    marks int
    ackedPlayer int  // <0 for none.
    haveTeamsBuzzed []bool
    pendingPresses []int
    scoreboard *Scoreboard
    engine *Engine
}


// Internals.

// Button press handler.
func (this *QuickFire) button(id int) {
    team, _ := BuzzerIdToTeam(id)

    if this.haveTeamsBuzzed[team] {
        // This team has already buzzed, ignore press.
        return
    }

    // This is the first press for this team.
    this.haveTeamsBuzzed[team] = true
    this.handlePress(id)
}


// Handle the given button press, which may have been pended.
func (this *QuickFire) handlePress(id int) {
    if this.ackedPlayer >= 0 {
        // A previous button press is currently being handled, pend this one.
        this.pendingPresses = append(this.pendingPresses, id)
        return
    }

    // Indicate pressed buzzer and await instruction from the user.
    this.engine.SetMode(id, true, true)
    this.ackedPlayer = id
    this.engine.RegisterCmd(this.commandCorrect, "Player answered correctly", 'y')
    this.engine.RegisterCmd(this.commandIncorrect, "Player answered incorrectly", 'n')
    fmt.Printf("Player %s pressed their button\n", BuzzerIdToString(id))
}


// Command handler for starting a new question.
func (this *QuickFire) commandNewQuestion(values []int) {
    this.NewQuestion(values[0])
}


// Command handler for the last acknowledge player gave the correct answer.
func (this *QuickFire) commandCorrect([]int) {
    this.Correct()
}


// Command handler for the last acknowledge player gave the incorrect answer.
func (this *QuickFire) commandIncorrect([]int) {
    this.Incorrect()
}


// Command handler for cancelling the current question.
func (this *QuickFire) commandCancel(values []int) {
    this.Cancel()
}


// Print a message stating the teams we're waiting for an answer from.
func (this *QuickFire) printWaiting() {
    s := ""

    for team, haveBuzzed := range this.haveTeamsBuzzed {
        if !haveBuzzed {
            s += " " + TeamIdToString(team)
        }
    }

    fmt.Printf("Waiting for button press from:%s\n", s)
}


// Finish the current question.
func (this *QuickFire) finish() {
    // Unregister everything we temporarily registered.
    this.engine.DeregisterCmd(this.commandCancel, 'q')
    this.engine.DeregisterButtons(this.button)

    if this.ackedPlayer >= 0 {
        this.engine.DeregisterCmd(this.commandCorrect, 'y')
        this.engine.DeregisterCmd(this.commandIncorrect, 'n')
    }

    // De-illuminate all buzzers.
    this.engine.SetModeAll(false, false)
}
