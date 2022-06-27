/* Functions to connect to WIFI and get an IP address from DHCP.

*/

#ifndef WIFI_H
#define WIFI_H

// Setup WIFI structures and config.
// Must be called before any other wifi_* functions.
void wifi_init(void);

// Attempt to connect to WIFI and get an IP address from DHCP.
// Returns true on success, false on failure.
bool wifi_connect(void);

#endif
