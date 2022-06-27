/* Functions to connect to WIFI and get an IP address from DHCP.

*/

#include "freertos/FreeRTOS.h"
#include "freertos/event_groups.h"
#include "esp_wifi.h"
#include "global.h"
#include "wifi.h"

// Hardcode WIFI settings.
#define WIFI_SSID "BeastQuiz"
#define WIFI_PASSWORD "SassThatHoopyFordPrefect"
#define WIFI_MAX_RETRIES 6  // After this we wait for a while before trying again.

/* We need to signal from our callback function to our main thread when connection to WIFI has succeeded or failed. To
do this we use an event group. Each bit is a separate event, we only care about 2 success and failure.
*/
static EventGroupHandle_t _wifi_event_signal;  // Event group to signal result back to main thread.
static volatile int _wifi_connect_retries;  // Number of retries so far.
#define WIFI_CONNECTED 1  // Connected to WIFI and DHCP IP address received.
#define WIFI_FAILED 2  // Failed to connect to WIFI WIFI_MAX_RETRIES times.


// Event handler for WIFI conenction.
static void event_handler(void* arg, esp_event_base_t event_base, int32_t event_id, void* event_data)
{
    if(event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_START) {
        // Start connection to WIFI.
        _wifi_connect_retries = 0;
        esp_wifi_connect();
        return;
    }

    if(event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_DISCONNECTED) {
        // Failed to connect to WIFI.
        if(_wifi_connect_retries >= WIFI_MAX_RETRIES) {
            // Too many attempts, give up for now.
            xEventGroupSetBits(_wifi_event_signal, WIFI_FAILED);
            return;
        }

        // Retry.
        _wifi_connect_retries++;
        esp_wifi_connect();
        return;
    }

    if(event_base == IP_EVENT && event_id == IP_EVENT_STA_GOT_IP) {
        // Got IP address from DHCP, just signal main thread.
        xEventGroupSetBits(_wifi_event_signal, WIFI_CONNECTED);
    }
}


// Setup WIFI structures and config.
// Must be called before any other wifi_* functions.
void wifi_init(void)
{
    _wifi_event_signal = xEventGroupCreate();

    esp_netif_init();
    esp_event_loop_create_default();
    esp_netif_create_default_wifi_sta();

    wifi_init_config_t cfg = WIFI_INIT_CONFIG_DEFAULT();
    esp_wifi_init(&cfg);

    wifi_config_t wifi_config = {
        .sta = {
            .ssid = WIFI_SSID,
            .password = WIFI_PASSWORD,
            .threshold.authmode = WIFI_AUTH_WPA2_PSK,
            .pmf_cfg = {
                .capable = true,
                .required = false
            }
        }
    };
    esp_wifi_set_mode(WIFI_MODE_STA);
    esp_wifi_set_config(WIFI_IF_STA, &wifi_config);
}


// Attempt to connect to WIFI and get an IP address from DHCP.
// Returns true on success, false on failure.
bool wifi_connect(void)
{
    // Register for events.
    esp_event_handler_instance_t wifi_events;
    esp_event_handler_instance_t got_ip_event;
    esp_event_handler_instance_register(WIFI_EVENT, ESP_EVENT_ANY_ID, event_handler, NULL, &wifi_events);
    esp_event_handler_instance_register(IP_EVENT, IP_EVENT_STA_GOT_IP, event_handler, NULL, &got_ip_event);

    // Attempt to connect.
    esp_wifi_start();

    // Wait for response.
    EventBits_t signal = xEventGroupWaitBits(_wifi_event_signal, WIFI_CONNECTED | WIFI_FAILED, pdFALSE, pdFALSE,
        portMAX_DELAY);

    // We don't want any more events to be processed, so unregister handler.
    esp_event_handler_instance_unregister(IP_EVENT, IP_EVENT_STA_GOT_IP, got_ip_event);
    esp_event_handler_instance_unregister(WIFI_EVENT, ESP_EVENT_ANY_ID, wifi_events);

    // Check which signal we got back.
    if((signal & WIFI_FAILED) != 0) {
        // Fail.
        return false;
    }

    // Success.
    return true;
}
