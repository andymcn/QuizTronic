/* Functions to track quiz scores.

*/

package main

import "fmt"
import "math"
import "os"


// Create a scoreboard.
func CreateScoreboard(engine *Engine) *Scoreboard {
    var p Scoreboard
    p.scores = make([]int, 4)  // TODO: Remove embedded 4.

    // Open log file.
    logFile, err := os.Create(ScoreLogFile)
    if err == nil {
        fmt.Printf("Writing scores to %s\n", ScoreLogFile)
        p.logFile = logFile
    } else {
        fmt.Printf("Could not open %s for writing: %v\n", ScoreLogFile, err)
        p.logFile = os.Stdout
    }

    engine.RegisterCmd(p.commandAdd, "Give points to a team", '+', ARG_TEAM, ARG_MARKS)
    engine.RegisterCmd(p.commandSub, "Deduct points from a team", '-', ARG_TEAM, ARG_MARKS)

    return &p
}


// Add points to the specified team.
func (this *Scoreboard) Add(team int, points int) {
    this.scores[team] += points
}


// Print out the current scores.
func (this *Scoreboard) Print() {
    // We want to find 1st, 2nd, etc places, allowing for ties.
    // Create a copy of the scores that we can destroy.
    scores := make([]int, len(this.scores))
    copy(scores, this.scores)

    places := make([]int, len(this.scores))
    ties := make([]string, len(this.scores))
    for i := range ties { ties[i] = " " }

    // Find the team in each place in turn.
    lastScore := math.MaxInt
    lastTeam := -1
    for place := range scores {
        // Find the team in next highest place.
        team := this.highestIntIndex(scores)
        places[team] = place + 1  // Places are reported 1 based.
        score := scores[team]
        scores[team] = math.MinInt

        // Check for a tie.
        if score == lastScore {
            // This team ties with the previous.
            ties[team] = "="
            ties[lastTeam] = "="
            places[team] = places[lastTeam]
        }

        lastScore = score
        lastTeam = team
    }

    // Stringify all teams' scores, so we can print ona  single line.
    s := ""
    for i := 0; i < 4; i++ {
        s += fmt.Sprintf("   %s%s%d:%3d.", TeamIdToString(i), ties[i], places[i], this.scores[i])
        // s += fmt.Sprintf("   %s%d %s %3d.", ties[i], places[i], TeamIdToString(i), this.scores[i])
    }

    // Finally we can print the scores.
    fmt.Fprintf(this.logFile, "Scores:%s\n", s)
}


// Scoreboard object.
type Scoreboard struct {
    scores []int
    logFile *os.File
}


// Internals.

const (ScoreLogFile string = "score.log")

// Command handler for adding points to the specified team.
func (this *Scoreboard) commandAdd(values []int) {
    this.Add(values[0], values[1])
    this.Print()
}


// Command handler for subtracting points from the specified team.
func (this *Scoreboard) commandSub(values []int) {
    this.Add(values[0], -values[1])
    this.Print()
}


// Find the index of the highest value in the given list.
func (this *Scoreboard) highestIntIndex(values []int) int {
    maxValue := math.MinInt
    maxIndex := -1

    for i, v := range values {
        if v > maxValue {
            maxValue = v
            maxIndex = i
        }
    }

    return maxIndex
}
