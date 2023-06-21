/* Functions to handle multiple choice questions.

A multiple choice controller lives for arbitrarily many questions.

Operation is as follows:
1. When each question starts all of the multiple choice answer buzzers are illuminated.
2. When each team presses one of their buttons, that is recorded. The pressed button stays illuminated and all of that
   team's others are de-illuminated.
3. If a team presses a different multiple choice button, that is recorded and the illuminations are updated
   accordingly.
4. When the user tells the controller to continue, any team with the correct answer gets a mark. All buttons are
   de-illuminated.

All multiple choice functions and methods must be called only in the main thread, unless otherwise stated.

*/

package main

import "fmt"


// Create a multiple choice controller.
func CreateMultipleChoice(engine *Engine, scoreboard *Scoreboard) *MultipleChoice {
    var p MultipleChoice
    p.engine = engine
    p.scoreboard = scoreboard

    engine.RegisterCmd(p.commandNewQuestion, "Start a multiple choice question", 'm', ARG_MULTIPLE_CHOICE, ARG_MARKS)

    return &p
}


// Start a new multiple choice question.
func (this *MultipleChoice) NewQuestion(answer int, marks int) {
    this.correctAnswer = answer
    this.marks = marks
    // TODO: Remove embedded team count.
    this.teamChoices = make([]int, 4)
    for i := range this.teamChoices { this.teamChoices[i] = -1 }

    // Illuminate all multiple choice buzzers.
    this.engine.SetModeAll(false, false)

    for team := 0; team < 4; team++ {
        // TODO: Remove embedded multiple choice answer count.
        for i := 0; i < 5; i++ {
            buzzer := TeamToBuzzerId(team, i)
            this.engine.SetMode(buzzer, true, false)
        }
    }

    // Register for needed inputs for duration of question.
    this.engine.RegisterCmd(this.commandComplete, "Complete current question", 'y')
    this.engine.RegisterCmd(this.commandCancel, "Cancel current question", 'q')
    this.engine.RegisterButtons(this.button)
}


// Complete the current question.
func (this *MultipleChoice) Complete() {
    // Check if any team had the correct answer.
    correctTeams := ""

    for team, choice := range this.teamChoices {
        if choice == this.correctAnswer {
            this.scoreboard.Add(team, this.marks)
            correctTeams += " " + TeamIdToString(team)
        }
    }

    if correctTeams != "" {
        fmt.Printf("Teams who got it right:%s\n", correctTeams)
        this.scoreboard.Print()
    } else {
        fmt.Printf("No teams got it right\n")
    }

    this.finish()
}


// Cancel the current question.
func (this *MultipleChoice) Cancel() {
    // Nothing special to do.
    this.finish()
}


// Multiple choice controller.
type MultipleChoice struct {
    correctAnswer int
    marks int
    teamChoices []int
    scoreboard *Scoreboard
    engine *Engine
}


// Internals.

// Button press handler.
func (this *MultipleChoice) button(id int) {
    team, choice := BuzzerIdToTeam(id)

    if choice > 4 {
        // Not a valid multiple choice button, ignore press.
        return
    }

    if this.teamChoices[team] == choice {
        // Reiteration of existing choice. Nothing to do.
        return
    }

    // Report choice, then record it.
    if this.teamChoices[team] < 0 {
        // TODO: Add choiceToRune() function?
        fmt.Printf("Team %s selected %c    ", TeamIdToString(team), 'A' + rune(choice))
    } else {
        fmt.Printf("Team %s changed to %c  ", TeamIdToString(team), 'A' + rune(choice))
    }

    this.teamChoices[team] = choice
    this.printChoices()

    // Adjust illuminated buzzers accordingly.
    for i := 0; i < 5; i++ {
        ledOn := (i == choice)
        this.engine.SetMode(TeamToBuzzerId(team, i), ledOn, false)
    }
}


// Command handler for starting a new question.
func (this *MultipleChoice) commandNewQuestion(values []int) {
    this.NewQuestion(values[0], values[1])
}


// Command handler for completing the current question.
func (this *MultipleChoice) commandComplete(values []int) {
    this.Complete()
}


// Command handler for cancelling the current question.
func (this *MultipleChoice) commandCancel(values []int) {
    this.Cancel()
}


// Print current choices.
func (this *MultipleChoice) printChoices() {
    s := ""

    for team, choice := range this.teamChoices {
        letter := '-'
        if choice >= 0 { letter = 'A' + rune(choice) }

        s += fmt.Sprintf(" %s:%c", TeamIdToString(team), letter)
    }

    fmt.Printf("Choices:%s\n", s)
}


// Finish the current question.
func (this *MultipleChoice) finish() {
    // Unregister everything we temporarily registered.
    this.engine.DeregisterCmd(this.commandComplete, 'y')
    this.engine.DeregisterCmd(this.commandCancel, 'q')
    this.engine.DeregisterButtons(this.button)

    // De-illuminate all multiple choice buzzers.
    this.engine.SetModeAll(false, false)
}
