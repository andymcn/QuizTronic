/* Functions to control audio.

*/

#ifndef AUDIO_H
#define AUDIO_H

// Initialise audio.
// Must be called before any other audio_* functions.
void audio_init(void);

// Start audio playback.
void audio_start(void);

// Stop audio playback.
void audio_stop(void);

// Tick.
// Should be called every millisecond.
void IRAM_ATTR audio_tick(void);

#endif
