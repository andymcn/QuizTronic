/* Functions to control central finite state machine.

The current state controls which inputs and outputs are in use.
The state is changed due to external conditions, such as messages from the host or losing contact with the host.

*/

#ifndef STATE_H
#define STATE_H

// Initialise state machine.
// Must be called before any other state_* functions.
void state_init(void);

// Tell state to indicate we are trying to connect to server.
void state_connect(void);

// Tell state to indicate we are connected to server.
void state_connected(void);

// Specify whether LED and buzzer are enabled.
void state_enable(bool led, bool audio);

// Tick.
// Should be called every 125ms.
void IRAM_ATTR state_tick(void);

#endif
