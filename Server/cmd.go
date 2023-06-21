/* Functions to handle console commands.

At any time the user can enter commands at the console. These are parsed according to preset schema and cause
registered functions to be executed.

In the interests of minimising typing while running a quiz, commands are very terse and dense.

Each command is made up of:
 1. A single lead character, which specifies which command to run. This character must be unique, if it matches the
    user input, then the whole command must match.
 2. Some number of arguments. The number and type of arguments is specified by the command. Each argument is a fixed
    length of either 1 or 2 characters, depending on the argument type.

The argument types are:
  * Marks. Single character 0..9.
  * Team identifier. Single character B, G, R or Y, case insensitive.
  * Multiple choice answer. Single character A..E, case insensitive.
  * Buzzer identifier. Double character, team identifier followed by unsigned integer.

Only ASCII characters are permitted. Whitespace and extra leading/trailing characters are not permitted.

*/

package main

import "fmt"


// Extract the leading command character from the given user input.
func ParseUserCmd(userInput string) byte {
    if len(userInput) == 0 {
        // Not valid, return unusable rune.
        return 0
    }

    // Since we only allow ASCII we can just grab the first byte.
    return userInput[0]
}


// Argument types.
const (
    ARG_MARKS ArgType = iota
    ARG_TEAM
    ARG_MULTIPLE_CHOICE
    ARG_BUZ_ID
    // TODO: How to handle half marks?
)

type ArgType int


// Parse the given user input string, expecting the specified list of arguments.
// The leading command character will already have been processed before this call, but should still be present in the
// given input.
func ParseUserArgs(userInput string, argTypes []ArgType) (argValues []int, ok bool) {
    argValues = []int{}

    // Ditch the lead character from the given input.
    userInput = userInput[1:]

    // Run through the defined argument types.
    for _, argType := range argTypes {
        switch argType {
        case ARG_MARKS:
            value, ok := expectChar(&userInput, "marks", '0', '9', false)
            if !ok { return argValues, false }

            argValues = append(argValues, int(value))

        case ARG_TEAM:
            value, ok := expectTeam(&userInput, "team")
            if !ok { return argValues, false }

            argValues = append(argValues, int(value))

        case ARG_MULTIPLE_CHOICE:
            value, ok := expectChar(&userInput, "multiple choice", 'A', 'E', true)
            if !ok { return argValues, false }

            argValues = append(argValues, int(value))

        case ARG_BUZ_ID:
            team, ok := expectTeam(&userInput, "button")
            if !ok { return argValues, false }

            index, ok := expectChar(&userInput, "button", '0', '9', false)
            if !ok { return argValues, false }

            value := TeamToBuzzerId(team, int(index))
            argValues = append(argValues, int(value))
        }
    }

    // Check there's no extra input.
    if len(userInput) != 0 {
        fmt.Printf("Unexpected input found: %s\n", userInput)
        return argValues, false
    }

    return argValues, true
}


// Return usage info for the given argument type list.
func ArgUsage(argTypes []ArgType) string {
    s := ""

    for _, argType := range argTypes {
        switch argType {
        case ARG_MARKS:             s += "<marks>"
        case ARG_TEAM:              s += "<team>"
        case ARG_MULTIPLE_CHOICE:   s += "<answer>"
        case ARG_BUZ_ID:            s += "<button>"
        }
    }

    return s
}


// Internals.

// Extract a single character from the start of the given string, which must be in the specified range (inclusive).
// The character will be removed from the given string.
// The expected argument is used for reporting errors and should be "value" or similar.
// If caseInsensitive is set to true, the character found will be forced to upper case before being compared to the
// given range.
// The value returned is the index into the given range.
func expectChar(cmdLine *string, expected string, min byte, max byte, caseInsensitive bool) (index byte, ok bool) {
    char, ok := extractChar(cmdLine, expected)
    if !ok { return 0, false }

    charOrig := char
    if caseInsensitive { char &= 0xEF }

    if (char < min) || (char > max) {
        fmt.Printf("Bad command, expected %s, got \"%c\"\n", expected, charOrig)
        return 0, false
    }

    return char - min, true
}


// Extract a team number from the start of the given string and decode it.
// The team ID will be removed from the given string.
// The expected argument is used for reporting errors and should be "team" or similar.
func expectTeam(cmdLine *string, expected string) (team int, ok bool) {
    id, ok := extractChar(cmdLine, expected)
    if !ok { return 0, false }

    team, ok = decodeTeam(id)

    if !ok {
        fmt.Printf("Bad command, expected %s, got \"%c\"\n", expected, id)
        return 0, false
    }

    return team, true
}


// Decode the given character into a team number.
func decodeTeam(id byte) (team int, ok bool) {
    switch id {
    case 'b', 'B':  return 0, true  // Blue.
    case 'g', 'G':  return 1, true  // Green.
    case 'r', 'R':  return 2, true  // Red.
    case 'y', 'Y':  return 3, true  // Yellow.

    default:
        // Unrecognised team ID.
        return 0, false
    }
}


// Extract the next character from the given command line.
// The character will be removed from the given string.
// The expected argument is used for reporting errors and should be "value" or similar.
// The value returned is the index into the given range.
func extractChar(cmdLine *string, expected string) (char byte, ok bool) {
    if len(*cmdLine) == 0 {
        fmt.Printf("Bad command, expected %s not found\n", expected)
        return 0, false
    }

    char = (*cmdLine)[0]
    *cmdLine = (*cmdLine)[1:]
    return char, true
}
