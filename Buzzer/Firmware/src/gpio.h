/* Functions to control GPIOs.

*/

#ifndef GPIO_H
#define GPIO_H

#define PIN_LED_PCB 2
#define PIN_LED_STATUS 3
#define PIN_LED_BUTTON 16
#define PIN_BUZZER 12
#define PIN_BUTTON 17

// Configure all required pins as inputs/outputs.
void gpio_init(void);

// Read module ID from GPIOs.
uint8_t read_module_id(void);

#endif
