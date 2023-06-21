/* Functions to handle test mode.

A test mode controller lives for arbitrarily many occurrences of test mode.

Operation is as follows:
1. When we enter test mode all buzzers are de-illuminated.
2. Each press of a buzzer toggles whether it is illuminated and buzzing.
3. On exit from test mode all buzzers are de-illuminated.

All test mode functions and methods must be called only in the main thread, unless otherwise stated.

*/

package main

import "fmt"


// Create a test mode controller.
func CreateTestMode(engine *Engine) *TestMode {
    var p TestMode
    p.engine = engine

    engine.RegisterCmd(p.commandEnterTestMode, "Enter test mode", 't')

    return &p
}


// Enter test mode.
func (this *TestMode) EnterTestMode() {
    // De-illuminate all buzzers.
    this.buzzersOn = make(map[int]bool)
    this.engine.SetModeAll(false, false)

    // Register for needed inputs for duration of question.
    this.engine.RegisterCmd(this.commandExit, "Exit test mode", 'q')
    this.engine.RegisterButtons(this.button)

    fmt.Printf("Entering test mode\n")
}


// Test mode controller.
type TestMode struct {
    buzzersOn map[int]bool  // Indexed by buzzer ID.
    engine *Engine
}


// Internals.

// Button press handler.
func (this *TestMode) button(id int) {
    // Check is buzzer is currently on.
    on, ok := this.buzzersOn[id]

    if ok && on {
        // Buzzer is currently on, turn it off.
        this.engine.SetMode(id, false, false)
        this.buzzersOn[id] = false
    } else {
        // Buzzer is not currently on, turn it on.
        this.engine.SetMode(id, true, true)
        this.buzzersOn[id] = true
    }
}


// Command handler for starting a new question.
func (this *TestMode) commandEnterTestMode([]int) {
    this.EnterTestMode()
}


// Command handler for exiting test mode.
func (this *TestMode) commandExit(values []int) {
    // Unregister everything we temporarily registered.
    this.engine.DeregisterCmd(this.commandExit, 'q')
    this.engine.DeregisterButtons(this.button)

    // De-illuminate all buzzers.
    this.engine.SetModeAll(false, false)
}
