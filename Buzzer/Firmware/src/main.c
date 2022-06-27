#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "driver/timer.h"
#include "nvs_flash.h"

#include "global.h"
#include "audio.h"
#include "gpio.h"
#include "host.h"
#include "state.h"
#include "wifi.h"

// #include "freertos/FreeRTOS.h"
// #include "freertos/task.h"
#include "driver/gpio.h"
#include "driver/timer.h"
#include "global.h"
#include "audio.h"
#include "gpio.h"
#include "state.h"

static int _timer_state_divide;  // Divide counter for state tick.


// Interrupt tick handler.
static void IRAM_ATTR audio_isr(void *param)
{
    // Timer admin.
    TIMERG0.int_clr_timers.t0 = 1;
    TIMERG0.hw_timer[0].config.alarm_en = 1;

    // Call the audio tick every millisecond.
    audio_tick();

    // Call the state tick every 125ms.
    _timer_state_divide++;
    if(_timer_state_divide >= 125)
    {
        state_tick();
        _timer_state_divide = 0;
    }
}


// Setup the tick timer.
static void setup_timer(void)
{
    _timer_state_divide = 0;

    timer_config_t config;
    config.divider = 80;  // Set prescaler for 1 MHz clock.
    config.counter_dir = TIMER_COUNT_UP;
    config.alarm_en = 1;
    config.intr_type = TIMER_INTR_LEVEL;
    config.auto_reload = TIMER_AUTORELOAD_EN;
    config.counter_en = TIMER_PAUSE;
    timer_init(TIMER_GROUP_0, 0, &config);
    timer_set_counter_value(TIMER_GROUP_0, 0 ,0);
    timer_isr_register(TIMER_GROUP_0, 0, audio_isr, NULL, ESP_INTR_FLAG_IRAM, NULL);
    timer_set_alarm_value(TIMER_GROUP_0, 0, 1000);  // Period in ms.
    timer_enable_intr(TIMER_GROUP_0, 0);
    timer_start(TIMER_GROUP_0, 0);
}


// Connect to the host and run its commands.
// Only returns once we are not connected to the host.
// Returns true if we connect to the host and then disconnect, false if we never connect to it.
static bool run(void)
{
    // First we need WIFI, then we can connect to the host.
    state_connect();
    if(!wifi_connect()) return false;
    if(!host_connect()) return false;

    // We're connected to the host. Process any messages it sends us, until we're disconnected.
    state_connected();
    host_process_messages();
    return true;
}


void app_main(void)
{
    // Initialize NVS - needed for WIFI to work.
    esp_err_t ret = nvs_flash_init();
    if(ret == ESP_ERR_NVS_NO_FREE_PAGES || ret == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        ESP_ERROR_CHECK(nvs_flash_erase());
        ret = nvs_flash_init();
    }

    ESP_ERROR_CHECK(ret);

    // Initialise everything else.
    gpio_init();
    wifi_init();
    state_init();
    host_init();
    audio_init();
    setup_timer();

    // Main loop.
    while(1)
    {
        // Try talking to the host.
        if(!run())
        {
            // Couldn't connect to host. Wait before trying again.
            vTaskDelay(2000 / portTICK_PERIOD_MS);
        }

        // We aren't connected to the host, try again.
    }
}
