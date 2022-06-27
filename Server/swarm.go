/* Functions for managing a swarm of physical buzzers.

For each known buzzer we record timing stats, to spot any latency issues.

We record for both the current connection session and the total duration of this program. This is intended to allow
checking whether a power cycle fixes a buzzer that's having problems. To enable this, we do not delete our record for
a buzzer when it disconnects.



*/

package main

import "fmt"
import "sort"
import "time"


// External interface.

// Create a Swarm object, which will track our buzzers.
func CreateSwarm(cmdProc *CommandProcessor, controller *Controller) *Swarm {
    var p Swarm
    p.controller = controller
    p.buzzers = make(map[int]*buzzerRecord)
    p.requests = make(chan func(), 1000)

    go p.run()

    cmdProc.AddCommand(p.PrintStats, "Print stats", "stats")
    cmdProc.AddCommand(p.commandOn, "Enable outputs on 1 buzzer", "on", LEX_BUZ_ID)
    cmdProc.AddCommand(p.commandOffAll, "Disable outputs on all buzzers", "offall")
    cmdProc.AddCommand(p.commandOff, "Disable outputs on 1 buzzer", "off", LEX_BUZ_ID)

    return &p
}


// Report discovery of a new buzzer.
func (this *Swarm) NewBuzzer(id int, buzzer *Buzzer) {
    this.requests <- func() {
        // Lookup buzzer.
        p, ok := this.buzzers[id]

        if !ok {
            // Record not found for new buzzer, create one.
            var rec buzzerRecord
            rec.id = id
            p = &rec
            this.buzzers[id] = p
        }

        p.buzzer = buzzer

        // Clear sessions stats.
        p.lastMsgTime = time.Now()
        p.slow2sCountSession = 0
        p.slow3sCountSession = 0
    }
}


// Report disconnection from a buzzer.
func (this *Swarm) Disconnected(id int, buzzer *Buzzer) {
    this.requests <- func() {
        // Lookup buzzer.
        rec, ok := this.buzzers[id]
        if !ok { return }  // Buzzer not found, nothing to do.

        // Buzzer ID found. We have to check it's the same buzzer, because concurrency.
        if rec.buzzer != buzzer { return }  // Specified buzzer has already been replaced, nothing to do.

        // We've found the specified buzzer. Ditch it.
        // We keep the record for stats purposes.
        rec.buzzer = nil
    }
}


// Report that a message has been received from a buzzer.
func (this *Swarm) Received(id int) {
    this.requests <- func() {
        // Lookup buzzer.
        rec, ok := this.buzzers[id]
        if !ok { return }  // Buzzer not found, nothing to do.

        now := time.Now()
        gap := now.Sub(rec.lastMsgTime)
        rec.lastMsgTime = now
        slow := false

        if gap > (3 * time.Second) {
            rec.slow3sCountSession++
            rec.slow3sCountTotal++
            slow = true
        } else if gap > (2 * time.Second) {
            rec.slow2sCountSession++
            rec.slow2sCountTotal++
            slow = true
        }

        if slow {
            fmt.Printf("Slow message %v\n", gap)
        }
    }
}


// Send a mode message to the specified buzzer.
// Returns false if the specified buzzer cannot be found.
func (this *Swarm) SetMode(buzzerId int, ledOn bool, buzzerOn bool) bool {
    // Create channel to get response.
    response := make(chan bool, 1)

    this.requests <- func() {
        // Lookup buzzer.
        rec, ok := this.buzzers[buzzerId]
        if !ok {
            // Buzzer not found.
            response <- false
            return
        }

        // Sending can be slow, so use a fresh Go routine.
        rec.buzzer.SetMode(ledOn, buzzerOn)
        response <- true
    }

    // Wait for response.
    return <-response
}


// Send a mode message to all connected buzzers.
func (this *Swarm) SetModeAll(ledOn bool, buzzerOn bool) {
    this.requests <- func() {
        // Run through each buzzer in turn.
        for _, buzzer := range this.buzzers {
            if buzzer.buzzer != nil {
                buzzer.buzzer.SetMode(ledOn, buzzerOn)
            }
        }
    }

    // No need to wait for a response.
}


// Print out stats for all known buzzers.
func (this *Swarm) PrintStats(value ...int) {
    this.requests <- func() {
        // Run through all buzzers.
        sumSlow2sCountSession := 0
        sumSlow3sCountSession := 0
        sumSlow2sCountTotal := 0
        sumSlow3sCountTotal := 0
        okCount := 0

        fmt.Printf("             >2s >3s (>2s >3s)\n")

        // First get and sort the buzzer IDs.
        ids := make([]int, 0, len(this.buzzers))
        for id := range this.buzzers {
            ids = append(ids, id)
        }
        sort.Ints(ids)

        // Now run through the buzzers in ID order.
        for _, id := range ids {
            buzzer, _ := this.buzzers[id]
            status := "Missing"
            if buzzer.buzzer != nil {
                status = "OK     "
                okCount++
            }

            fmt.Printf("%3s: %s %3d %3d (%3d %3d)\n", BuzzerIdToString(buzzer.id), status,
                buzzer.slow2sCountSession, buzzer.slow3sCountSession,
                buzzer.slow2sCountTotal, buzzer.slow3sCountTotal)

            sumSlow2sCountSession += buzzer.slow2sCountSession
            sumSlow3sCountSession += buzzer.slow3sCountSession
            sumSlow2sCountTotal += buzzer.slow2sCountTotal
            sumSlow3sCountTotal += buzzer.slow3sCountTotal
        }

        fmt.Printf("Sum: %2d OK   %3d %3d (%3d %3d)\n", okCount,
            sumSlow2sCountSession, sumSlow3sCountSession,
            sumSlow2sCountTotal, sumSlow3sCountTotal)
    }
}


// Object to represent a physical buzzer with which we're communicating.
type Swarm struct {
    controller *Controller
    buzzers map[int]*buzzerRecord  // Indexed by ID.
    requests chan func()  // All requests are handling in the central Go routine.
}


// Internals.

// Info we need to store per buzzer.
type buzzerRecord struct {
    buzzer *Buzzer  // nil if disconnected.
    id int
    lastMsgTime time.Time
    slow2sCountSession int
    slow3sCountSession int
    slow2sCountTotal int
    slow3sCountTotal int
}


// Handles requests in a single thread.
// Never returns. Should be called as a Go routine.
func (this *Swarm) run() {
    // Setup a tick for checking for dead connections.
    ticker := time.NewTicker(time.Second)

    // Process incoming messages forever.
    for {
        select {
        case request := <-this.requests:
            request()

        case <-ticker.C:
            this.checkDisconnects()
        }
    }
}


// Check if any buzzers have disappeared.
func (this *Swarm) checkDisconnects() {
    now := time.Now()

    // Check each buzzer in turn.
    for id, buzzer := range this.buzzers {
        if buzzer.buzzer != nil {

            age := now.Sub(buzzer.lastMsgTime)

            if age > (5 * time.Second) {
                // We've not heard from this buzzer for too long, disconnect it.
                fmt.Printf("Buzzer %s quiet for >10s, disconnecting\n", BuzzerIdToString(id))

                // We don't need to adjust our records now, since the buzzer will tell us it's disconnected.
                buzzer.buzzer.Disconnect()
            }
        }
    }
}


// Command handler for turning on outputs on a specified buzzer.
func (this *Swarm) commandOn(value ...int) {
    this.SetMode(value[0], true, true)
}


// Command handler for turning off outputs on a specified buzzer.
func (this *Swarm) commandOff(value ...int) {
    this.SetMode(value[0], false, false)
}


// Command handler for turning off outputs on all buzzers.
func (this *Swarm) commandOffAll(value ...int) {
    this.SetModeAll(false, false)
}
