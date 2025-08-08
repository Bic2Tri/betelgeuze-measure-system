package config

const (
    CMD_START         = 0x95
    CMD_GET_DIMENSIONS = 0x89
    CMD_SET_TOP_MAX   = 0x90
    CMD_SET_WIDTH_MAX = 0x91
    CMD_SET_LENGTH_MAX = 0x92
    CMD_RESET_SENSORS = 0x93
    CMD_LED_ON        = 0x66
    CMD_LED_OFF       = 0x55
    CMD_PING          = 0x77
    
    SERVER_PORT = ":8080"
    WEIGHT_THRESHOLD = 1.0
)