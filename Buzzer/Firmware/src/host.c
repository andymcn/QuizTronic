/* Functions to communicate with the host.

*/

#include "lwip/sockets.h"
#include "global.h"
#include "host.h"
#include "gpio.h"
#include "state.h"

// Hardcode host IP address.
#define HOST_IP "192.168.2.5"

static volatile int _host_socket;

// Message values.
#define MSG_VERSION     0x04
#define MSG_MODE_PREFIX 0x20
#define MSG_MODE_MASK   0xFC
#define MSG_MODE_LED    0x01
#define MSG_MODE_AUDIO  0x02
#define MSG_PRESS       0x30
#define MSG_HEARTBEAT   0x31
#define MSG_ERR_BAD_MSG 0x7F
#define MSG_ID_PREFIX   0x80


// Send the given message to our host.
// Returns true on success, false on failure.
static bool host_send(uint8_t message_byte)
{
    if(_host_socket == 0) return false;

    // Send message byte.
    char msg[] = {message_byte};
    if(send(_host_socket, msg, 1, 0) < 0)
    {
        // Error sending.
        _host_socket = 0;
        return false;
    }

    return true;
}


// Task to send heartbeats to our host.
static void heartbeat_task(void *param)
{
    while(1)
    {
        // We should only try to send if we have an open socket. host_send() handles that for us.
        host_send(MSG_HEARTBEAT);
        vTaskDelay(1000 / portTICK_PERIOD_MS);  // Send roughly every 1 second.
    }
}


// Initialise host communication.
// Must be called before any other host_* functions.
void host_init(void)
{
    _host_socket = 0;

    // Start our heartbeat task.
    xTaskCreate(heartbeat_task, "Heartbeat", 2048, NULL, 1, NULL);
}


// Open connection to the host.
// Returns true on success, false on failure.
bool host_connect(void)
{
    int sock = socket(AF_INET, SOCK_STREAM, IPPROTO_IP);
    if(sock < 0) {
        _host_socket = 0;
        return false;
    }

    struct sockaddr_in host_addr;
    host_addr.sin_addr.s_addr = inet_addr(HOST_IP);
    host_addr.sin_family = AF_INET;
    host_addr.sin_port = htons(9753);

    int err = connect(sock, (struct sockaddr *)&host_addr, sizeof(struct sockaddr_in));
    if(err != 0) {
        // Could not connect to host. Maybe it's not listening yet.
        shutdown(sock, 0);
        close(sock);
        _host_socket = 0;
        return false;
    }

    // We're connected to the host. Send initial messages.
    _host_socket = sock;
    uint8_t id = read_module_id();  // We need to know our ID.

    if(!host_send(MSG_VERSION)) return false;
    if(!host_send(MSG_ID_PREFIX | id)) return false;

    return true;
}


// Listen for, and process, incoming messages from the host.
// Only returns when communication with the host is lost.
void host_process_messages(void)
{
    while(_host_socket != 0)
    {
        uint8_t msg;
        recv(_host_socket, &msg, 1, 0);

        if((msg & MSG_MODE_MASK) == MSG_MODE_PREFIX) {
            // Mode message. Check bits to see which outputs should be on.
            bool led = ((msg & MSG_MODE_LED) != 0);
            bool audio = ((msg & MSG_MODE_AUDIO) != 0);
            state_enable(led, audio);
        } else {
            // Unrecognised message, error.
            host_send(MSG_ERR_BAD_MSG);
        }
    }
}


// Send a button press message to our host.
void host_send_press(void)
{
    host_send(MSG_PRESS);
}
