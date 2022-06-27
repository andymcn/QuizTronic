/* Functions to control audio.

Communication between the interrupt and other threads is via global bools, which can be written and read atomically.
These flags can be set by any thread to request an action and are cleared by the audio playback interrupt.

*/

// #include "freertos/FreeRTOS.h"
// #include "freertos/task.h"
#include "driver/gpio.h"
// #include "driver/timer.h"
#include "global.h"
#include "audio.h"
#include "gpio.h"

static volatile bool _audio_start;  // Signal to start playback.
static volatile bool _audio_stop;  // Signal to stop playback.
static volatile int _audio_count;  // Count through playback. 0 => not playing back.


// Interrupt tick handler.
void IRAM_ATTR audio_tick(void)
{
    // Check signals.
    // Stop takes priority, so check start first.
    if(_audio_start)
    {
        _audio_start = false;

        // Don't restart the audio if it's already playing, but do consume the start signal.
        if(_audio_count == 0)
        {
            _audio_count = 1000;
        }
    }

    if(_audio_stop)
    {
        _audio_stop = false;
        _audio_count = 1;  // 1 causes us to turn off the transducer and then stop playback.
    }

    // Now playback, if needed.
    if(_audio_count > 0)
    {
        _audio_count--;
        gpio_set_level(PIN_BUZZER, _audio_count & 1);
    }
}


// Initialise audio.
// Must be called before any other audio_* functions.
void audio_init(void)
{
    _audio_start = false;
    _audio_stop = false;
    _audio_count = 0;
}


// Start audio playback.
void audio_start(void)
{
    // Just set the flag.
    _audio_start = true;
}


// Stop audio playback.
void audio_stop(void)
{
    // Just set the flag.
    _audio_stop = true;
}
