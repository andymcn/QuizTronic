/* Functions to control central finite state machine.

The current state controls which inputs and outputs are in use.
The state is changed due to external conditions, such as messages from the host or losing contact with the host.

A regular tick is required, which presumably is called in an interrupt context. Communication between the interrupt
and main thread is via global bools, which can be written and read atomically.

*/

#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "driver/gpio.h"
#include "global.h"
#include "audio.h"
#include "gpio.h"
#include "host.h"
#include "state.h"

// States.
// #define STATE_CONNECT   0  // Connecting to host.
// #define STATE_IDLE      1  // Doing nothing, awaiting instructions from host.
// #define STATE_READY     2  // Ready for use, listening for button presses.
// #define STATE_ON        3  // Button pressed, LED on, buzzer sounding initially.

// static volatile uint8_t _state;
static volatile bool _connected;
static volatile bool _led_on;
static volatile bool _button_pressed = false;
static volatile bool _status_flashing;
static volatile int _flash_phase;


// Check the state of the button and act accordingly.
static void check_button(void)
{
    // The button is wired active low.
    int pin = gpio_get_level(PIN_BUTTON);
    bool new_state = (pin == 0);

    if(new_state && !_button_pressed && _connected)
    {
        // The button is newly pressed and we should be reporting presses.
        host_send_press();
        // audio_start();  // Temp test.
    }

    _button_pressed = new_state;

    // Set PCB LED to button state to aid debugging.
    if(new_state)
    {
        gpio_set_level(PIN_LED_PCB, 1);
    } else {
        gpio_set_level(PIN_LED_PCB, 0);
    }
}


// Task to poll button.
static void button_poll_task(void *param)
{
    while(1)
    {
        check_button();
        vTaskDelay(10 / portTICK_PERIOD_MS);  // Check roughly every 10 milliseconds.
    }
}


// Initialise state machine.
// Must be called before any other state_* functions.
void state_init(void)
{
    _flash_phase = 0;
    state_connect();

    // Start our button pool task.
    xTaskCreate(button_poll_task, "ButtonPool", 2048, NULL, 5, NULL);
}


// Tell state to indicate we are trying to connect to server.
void state_connect(void)
{
    _connected = false;
    _led_on = false;
    _status_flashing = true;
    gpio_set_level(PIN_LED_BUTTON, 0);
    audio_stop();
}


// Tell state to indicate we are connected to server.
void state_connected(void)
{
    _connected = true;
    _led_on = false;
    _status_flashing = false;
    gpio_set_level(PIN_LED_BUTTON, 0);
    audio_stop();
}


// Specify whether LED and buzzer are enabled.
void state_enable(bool led, bool audio)
{
    _led_on = led;
    gpio_set_level(PIN_LED_BUTTON, led ? 1 : 0);

    if(audio) {
        audio_start();
    } else {
        audio_stop();
    }

    _status_flashing = false;
}


// Tick.
// Should be called every 125ms.
void IRAM_ATTR state_tick(void)
{
    if(_status_flashing)
    {
        // Toggle status LED.
        _flash_phase = 1 - _flash_phase;
    } else {
        // Force status LED on.
        _flash_phase = 1;
    }

    gpio_set_level(PIN_LED_STATUS, _flash_phase);
}
