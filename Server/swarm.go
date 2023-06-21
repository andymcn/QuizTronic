/* Functions for managing a swarm of physical buzzers.

For each known buzzer we record timing stats, to spot any latency issues.

We record for both the current connection session and the total duration of this program. This is intended to allow
checking whether a power cycle fixes a buzzer that's having problems. To enable this, we do not delete our record for
a buzzer when it disconnects.

*/

package main

import "fmt"
import "os"
import "sort"
import "time"


// External interface.

// Create a Swarm object, which will track our buzzers.
func CreateSwarm(engine *Engine) *Swarm {
    var p Swarm
    p.buzzers = make(map[int]*buzzerRecord)
    p.engine = engine
    p.requests = make(chan func(), 1000)

    // Open log file.
    logFile, err := os.Create(BuzzersLogFile)
    if err == nil {
        fmt.Printf("Writing buzzer connections to %s\n", BuzzersLogFile)
        p.logFile = logFile
    } else {
        fmt.Printf("Could not open %s for writing: %v\n", BuzzersLogFile, err)
        p.logFile = os.Stdout
    }

    engine.RegisterCmd(p.printStats, "Print buzzer stats", 'Z')
    engine.RegisterCmd(p.commandOn, "Enable outputs on 1 buzzer", 'N', ARG_BUZ_ID)
    engine.RegisterCmd(p.commandOff, "Disable outputs on 1 buzzer", 'F', ARG_BUZ_ID)
    engine.RegisterCmd(p.commandOffAll, "Disable outputs on all buzzers", 'G')
    engine.RegisterCmd(p.commandTraceToggle, "Toggle button trace logging", 'T')
    engine.RegisterCmd(p.commandMute, "Mute 1 buzzer", 'M', ARG_BUZ_ID)
    engine.RegisterCmd(p.commandUnmute, "Unmute 1 buzzer", 'U', ARG_BUZ_ID)
    engine.RegisterCmd(p.commandUnmuteAll, "Unmute all buzzers", 'V')

    go p.run()
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

            this.Trace("Buzzer %s connected\n", BuzzerIdToString(id))
        } else {
            this.Trace("Buzzer %s reconnected\n", BuzzerIdToString(id))
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
        this.Trace("Buzzer %s disconnected\n", BuzzerIdToString(id))
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
            this.Log("Slow message %v\n", gap)
        }
    }
}


// Handle the given button press event.
func (this *Swarm) ButtonPress(buzzerId int) {
    // Just log this and pass it on to our engine.
    this.Trace("Buzzer %s pressed\n", BuzzerIdToString(buzzerId))
    this.engine.ButtonPress(buzzerId)
}


// Send a mode message to the specified buzzer.
// Returns false if the specified buzzer cannot be found.
func (this *Swarm) SetMode(buzzerId int, ledOn bool, buzzerOn bool) bool {
    // Create channel to get response.
    response := make(chan bool, 1)

    this.requests <- func() {
        // Lookup buzzer.
        rec, ok := this.buzzers[buzzerId]
        if !ok || (rec.buzzer == nil) {
            // Buzzer not found.
            response <- false
            return
        }

        // Check if the buzzer is muted.
        if rec.muted { buzzerOn = false }

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
                // Check if the buzzer is muted.
                b := buzzerOn
                if buzzer.muted { b = false }

                buzzer.buzzer.SetMode(ledOn, b)
            }
        }
    }
}


// Mute or unmute specified buzzer.
func (this *Swarm) Mute(buzzerId int, mute bool) {
    this.requests <- func() {
        un := ""
        if !mute { un = "un" }

        // Lookup buzzer.
        rec, ok := this.buzzers[buzzerId]
        if !ok {
            // Buzzer not found.
            fmt.Printf("Cannot %smute buzzer %s, not found\n", un, BuzzerIdToString(buzzerId))
            return
        }

        if rec.muted == mute {
            this.Trace("Buzzer %s already %smuted\n", BuzzerIdToString(buzzerId), un)
        } else {
            this.Trace("Buzzer %s %smuted\n", BuzzerIdToString(buzzerId), un)
        }

        rec.muted = mute
    }
}


// Unmute all buzzers.
func (this *Swarm) UnmuteAll() {
    this.requests <- func() {
        // Run through all known buzzers.
        for id, rec := range this.buzzers {
            if rec.muted {
                this.Trace("Buzzer %s unmuted\n", BuzzerIdToString(id))
            }

            rec.muted = false
        }
    }
}


// Log to the buzzers log.
func (this *Swarm) Log(format string, args ...interface{}) {
    fmt.Fprintf(this.logFile, format, args...)
}


// Log to the buzzers trace log.
func (this *Swarm) Trace(format string, args ...interface{}) {
    if this.trace {
        fmt.Fprintf(this.logFile, format, args...)
    }
}


// Object to represent a physical buzzer with which we're communicating.
type Swarm struct {
    buzzers map[int]*buzzerRecord  // Indexed by ID.
    engine *Engine
    trace bool
    logFile *os.File
    requests chan func()  // All requests are handling in the central Go routine.
}


// Internals.

// Info we need to store per buzzer.
type buzzerRecord struct {
    buzzer *Buzzer  // nil if disconnected.
    id int
    muted bool
    lastMsgTime time.Time
    slow2sCountSession int
    slow3sCountSession int
    slow2sCountTotal int
    slow3sCountTotal int
}

const (BuzzersLogFile string = "buzzer.log")


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
                this.Log("Buzzer %s quiet for >5s, disconnecting\n", BuzzerIdToString(id))

                // We don't need to adjust our records now, since the buzzer will tell us it's disconnected.
                buzzer.buzzer.Disconnect()
            }
        }
    }
}


// Command handler for turning on outputs on a specified buzzer.
func (this *Swarm) commandOn(values []int) {
    this.SetMode(values[0], true, true)
}


// Command handler for turning off outputs on a specified buzzer.
func (this *Swarm) commandOff(values []int) {
    this.SetMode(values[0], false, false)
}


// Command handler for turning off outputs on all buzzers.
func (this *Swarm) commandOffAll([]int) {
    this.SetModeAll(false, false)
}


// Command handler for muting a specified buzzer.
func (this *Swarm) commandMute(values []int) {
    this.Mute(values[0], true)
}


// Command handler for unmuting a specified buzzer.
func (this *Swarm) commandUnmute(values []int) {
    this.Mute(values[0], false)
}


// Command handler for unmuting all buzzers.
func (this *Swarm) commandUnmuteAll(values []int) {
    this.UnmuteAll()
}


// Command handler for toggling trace logging.
func (this *Swarm) commandTraceToggle([]int) {
    this.requests <- func() {
        this.trace = !this.trace

        if this.trace {
            this.Log("Trace logging on\n")
        } else {
            this.Log("Trace logging off\n")
        }
    }
}


// Print out stats for all known buzzers.
func (this *Swarm) printStats([]int) {
    this.requests <- func() {
        // Run through all buzzers.
        sumSlow2sCountSession := 0
        sumSlow3sCountSession := 0
        sumSlow2sCountTotal := 0
        sumSlow3sCountTotal := 0
        okCount := 0
        mutedCount := 0

        this.Log("             >2s >3s (>2s >3s)\n")

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

            muted := ""
            if buzzer.muted {
                muted = " muted"
                mutedCount++
            }

            this.Log("%3s: %s %3d %3d (%3d %3d)%s\n", BuzzerIdToString(buzzer.id), status,
                buzzer.slow2sCountSession, buzzer.slow3sCountSession,
                buzzer.slow2sCountTotal, buzzer.slow3sCountTotal, muted)

            sumSlow2sCountSession += buzzer.slow2sCountSession
            sumSlow3sCountSession += buzzer.slow3sCountSession
            sumSlow2sCountTotal += buzzer.slow2sCountTotal
            sumSlow3sCountTotal += buzzer.slow3sCountTotal
        }

        this.Log("Sum: %2d OK   %3d %3d (%3d %3d)  %d muted\n", okCount,
            sumSlow2sCountSession, sumSlow3sCountSession,
            sumSlow2sCountTotal, sumSlow3sCountTotal, mutedCount)
    }
}
