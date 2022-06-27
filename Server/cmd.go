/* Functions to handle console commands.

At any time the user can enter commands at the console. These are parsed according to preset schema and cause
functions to be executed

Each command is made up of a fixed string followed by simple lexical tokens, some of which produce a value. The
tokens are:
  * Unsigned integer. Integer value.
  * Single digit unsigned integer. Digit value.
  * Optional single character (flag). 1 if present, 0 otherwise.
  * Team identifier. Team ID.
  * Buzzer identifier. Buzzer ID.
  * Got. No value.

Whitespace between tokens is permitted and ignored, but not required.

The initial fixed string of each command must be unique, if it matches the user input, then the whole command must
match.

Commands are checked in the defined order and the first match is used. The initial string of a command maybe a
superstring of the initial string of an earlier command.

*/

package main

import "bufio"
import "fmt"
import "os"
import "strings"


// Create a command processor.
func CreateCommandProcessor() *CommandProcessor {
    fmt.Printf("? to show usage\n")

    var p CommandProcessor
    p.commands = make([]*cmdInfo, 0)

    p.AddCommand(p.Usage, "Show usage", "?")

    return &p

}


// Add a command to the thingy.
// The given function will be called when the given lexical tokens match an input line.
func (this *CommandProcessor) AddCommand(matchFn CmdFn, helpText string, initialString string, lex ...LexType) {
    var p cmdInfo
    p.matchFn = matchFn
    p.helpText = helpText
    p.initialString = initialString
    p.lexTokens = lex

    this.commands = append(this.commands, &p)
}

// Lexical token types.
const (
    LEX_UINT LexType = iota
    LEX_DIGIT
    LEX_TEAM
    LEX_BUZ_ID
    LEX_DOT
)

type LexType int

// Type of functions to call when a command matches.
// The values are from the lexical tokens that produce values, in order.
type CmdFn func(values ...int)


// Print a usage message for our commands.
func (this *CommandProcessor) Usage(values ...int) {
    fmt.Printf("Usage:\n")

    for _, cmd := range this.commands {
        s := cmd.initialString

        for _, token := range cmd.lexTokens {
            switch token {
            case LEX_UINT:    s += " {int}"
            case LEX_DIGIT:   s += " {digit}"
            case LEX_TEAM:    s += " {team_id}"
            case LEX_BUZ_ID:  s += " {buzzer_id}"
            case LEX_DOT:     s += "."
            }
        }

        fmt.Printf("  %s %s\n", s, cmd.helpText)
    }
}


// Read stdin and process all resulting commands.
// Never returns.
func (this *CommandProcessor) ProcessStdin() {
    stdin := bufio.NewReader(os.Stdin)

    for {
        text, _ := stdin.ReadString('\n')
        text = strings.TrimSpace(text)

        // Ignore blank lines.
        if text != "" {
            this.process(text)
        }
    }
}


// Command processor object.
type CommandProcessor struct {
    commands []*cmdInfo
}


// Internals.

// Info needed for a single command.
type cmdInfo struct {
    matchFn CmdFn
    helpText string
    initialString string
    lexTokens []LexType
}


// Parse the given command line and call the specified function.
func (this *CommandProcessor) process(cmdLine string) {
    // Check each of our commands in turn.
    for _, command := range this.commands {
        values, match := command.match(cmdLine)

        if match {
            // We've found the command.
            if values == nil {
                // Parse error, already reported. Nothing for us to do.
                return
            }

            // Call defined function.
            command.matchFn(values...)
            return
        }
    }

    fmt.Printf("Unrecognised command\n")
}


// Attempt to parse the given command line and extract variable values, according to this command.
// If there are no values an empty struct will be returned for values.
// If the command matches, but fails to parse correctly, then nil, true will be returned.
func (this *cmdInfo) match(cmdLine string) (values []int, match bool) {
    if !strings.HasPrefix(cmdLine, this.initialString) { return nil, false }  // No match.

    // Initial string matches, ditch it.
    cmdLine = cmdLine[len(this.initialString):]

    // Parse each token in turn and get their values.
    values = make([]int, 0, len(this.lexTokens))

    for _, token := range this.lexTokens {
        cmdLine = strings.TrimLeft(cmdLine, " ")
        if cmdLine == "" {
            fmt.Printf("Bad command, unexpected termination\n")
            return nil, true
        }

        switch token {
        case LEX_UINT:
            // Take as many digits as are available.
            value, ok := expectUint(&cmdLine, "integer")
            if !ok { return nil, true }  // Error already reported.
            values = append(values, value)

        case LEX_DIGIT:
            // Just take the next character, which must be a decimal digit.
            digit, ok := expectChar(&cmdLine, "dot", 0x30, 0x39)
            if !ok { return nil, true }  // Error already reported.

            digitValue := int(digit) - 0x30
            values = append(values, digitValue)

        case LEX_TEAM:
            // Just take the next character, which must be a team identifier, ie B, G, R or Y.
            team, ok := expectTeam(&cmdLine, "team")
            if !ok { return nil, true }  // Error already reported.
            values = append(values, team)

        case LEX_BUZ_ID:
            // First we need a team identifier.
            team, ok := expectTeam(&cmdLine, "buzzer")
            if !ok { return nil, true }  // Error already reported.

            // Now we need an integer.
            value, ok := expectUint(&cmdLine, "buzzer")
            if !ok { return nil, true }  // Error already reported.

            buzzer := (16 * team) + value
            values = append(values, buzzer)

        case LEX_DOT:
            // Just take the next character, which must be a dot.
            _, ok := expectChar(&cmdLine, "dot", 0x2E, 0x2E)
            if !ok { return nil, true }  // Error already reported.
        }
    }

    return values, true
}


// Extract a number from the start of the given string.
// The number will be removed from the given string.
// The expected argument is used for reporting errors and should be "value" or similar.
func expectUint(cmdLine *string, expected string) (value int, ok bool) {
    // Process each leading character until we find one that isn't a decimal digit.
    digitCount := 0
    value = 0
    for i := 0; i < len(*cmdLine); i++ {
        digit := (*cmdLine)[i]
        if (digit < 0x30) || (digit > 0x39) { break }  // Not a decimal digit.

        value = (10 * value) + int(digit - 0x30)
        digitCount++
    }

    // Check we found any valid digits.
    if digitCount == 0 {
        fmt.Printf("Bad command, expected %s, got \"%s\"\n", expected, (*cmdLine)[0:1])
        return 0, false
    }

    // Remove digits found from string.
    *cmdLine = (*cmdLine)[digitCount:]

    return value, true
}


// Extract a single character from the start of the given string, which must be in the specified range (inclusive).
// The character will be removed from the given string.
// The expected argument is used for reporting errors and should be "dot" or similar.
func expectChar(cmdLine *string, expected string, min byte, max byte) (char byte, ok bool) {
    char = (*cmdLine)[0]

    if (char < min) || (char > max) {
        fmt.Printf("Bad command, expected %s, got \"%s\"\n", expected, (*cmdLine)[0:1])
        return 0, false
    }

    *cmdLine = (*cmdLine)[1:]
    return char, true
}


// Extract a team number from the start of the given string and decode it.
// The team ID will be removed from the given string.
// The expected argument is used for reporting errors and should be "team" or similar.
func expectTeam(cmdLine *string, expected string) (team int, ok bool) {
    id := (*cmdLine)[0:1]
    team, ok = decodeTeam(id)

    if !ok {
        fmt.Printf("Bad command, expected %s, got \"%s\"\n", expected, id)
        return 0, false
    }

    *cmdLine = (*cmdLine)[1:]
    return team, true
}


// Decode the given string into a team number.
func decodeTeam(id string) (team int, ok bool) {
    switch id {
    case "b", "B":  return 0, true  // Blue.
    case "g", "G":  return 1, true  // Green.
    case "r", "R":  return 2, true  // Red.
    case "y", "Y":  return 3, true  // Yellow.

    default:
        // Unrecognised team ID.
        return 0, false
    }
}

