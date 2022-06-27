/* Functions to control GPIOs.

*/

#include "driver/gpio.h"
#include "global.h"
#include "gpio.h"

// Define the pins that make up the module ID, in order, big endian.
#define ID_SIZE 7
static int _id_pins[ID_SIZE] = {25, 26, 27, 9, 10, 13, 5};


// Configure the specified pin as an input.
static void configure_input_pin(int pin)
{
    gpio_config_t io_conf = {};  // 0 all fields.
    io_conf.intr_type = GPIO_INTR_DISABLE;
    io_conf.mode = GPIO_MODE_INPUT;
    io_conf.pin_bit_mask = 1 << (uint64_t)pin;
    io_conf.pull_down_en = 0;
    io_conf.pull_up_en = 1;  // Inputs have pull ups.
    gpio_config(&io_conf);
}


// Configure the specified pin as an output.
static void configure_output_pin(int pin)
{
    gpio_config_t io_conf = {};  // 0 all fields.
    io_conf.intr_type = GPIO_INTR_DISABLE;
    io_conf.mode = GPIO_MODE_OUTPUT;
    io_conf.pin_bit_mask = 1 << (uint64_t)pin;
    io_conf.pull_down_en = 0;
    io_conf.pull_up_en = 0;
    gpio_config(&io_conf);
}


// Configure all required pins as inputs/outputs.
void gpio_init(void)
{
    configure_output_pin(PIN_LED_PCB);
    configure_output_pin(PIN_LED_STATUS);
    configure_output_pin(PIN_LED_BUTTON);
    configure_output_pin(PIN_BUZZER);
    configure_input_pin(PIN_BUTTON);

    for(int i = 0; i < ID_SIZE; i++)
    {
        configure_input_pin(_id_pins[i]);
    }
}


// Read module ID from GPIOs.
uint8_t read_module_id(void)
{
    uint8_t id = 0;

    for(int i = 0; i < ID_SIZE; i++)
    {
        int pin = gpio_get_level(_id_pins[i]);
        id = (id << 1) | pin;
    }

    return id;
}
