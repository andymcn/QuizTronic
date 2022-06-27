/* Functions to communicate with the host.

*/

#ifndef HOST_H
#define HOST_H

// Initialise host communication.
// Must be called before any other host_* functions.
void host_init(void);

// Open connection to the host.
// Returns true on success, false on failure.
bool host_connect(void);

// Listen for, and process, incoming messages from the host.
// Only returns when communication with the host is lost.
void host_process_messages(void);

// Send a button press message to our host.
// May be called from interrupts.
void host_send_press(void);

#endif
