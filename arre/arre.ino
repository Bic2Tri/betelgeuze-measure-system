#include <TroykaI2CHub.h>
#include "Adafruit_VL53L0X.h"

#define LOX2_ADDRESS 0x29

#define L0X_RANGING_TIME_MS 250
#define FILTER_UP 9
#define FILTER_DN 4
#define STATIC_OFFSET 30
#define DIST_MIN 30

#define SENSOR_VL53_ERROR 1520
#define SENSOR_VL53_COVERED 1320
#define SENSOR_VL53_OUT_RANGE 1120

#define BUTTON 8
#define RED_LED_PIN 9
#define GREEN_LED_PIN 6

#define NUM_SENSORS 4

TroykaI2CHub splitter;
const byte sensors_pins[NUM_SENSORS] = {0, 2, 4, 6}; // LEFT, RIGHT, TOP, BACK

int TOP_MAX = 100, WIDTH_MAX = 100, LENGTH_MAX = 100;
bool onlyWeight = false, start = false;

int widthBox = 0, heightBox = 0, lengthBox = 0;
int right = 0, prev_right = 0, left = 0, prev_left = 0, top = 0, prev_top = 0, back = 0, prev_back = 0;
long loop_counter = 0;

Adafruit_VL53L0X lox;
byte broken_sensors[NUM_SENSORS] = {1, 1, 1, 1};

unsigned long lastAutoPrintTime = 0;
const unsigned long AUTO_PRINT_INTERVAL = 10000; // 10 секунд

void getDistance() {
  for (int i = 0; i < NUM_SENSORS; i++) {
    int distInt = SENSOR_VL53_ERROR;
    if (!broken_sensors[i]) {
      splitter.setBusChannel(sensors_pins[i]);
      delay(10); // Добавляем задержку для переключения канала
      
      VL53L0X_RangingMeasurementData_t measure;
      lox.rangingTest(&measure, false);
      
      if (measure.RangeStatus != 4) { // 4 означает ошибку измерения
        distInt = measure.RangeMilliMeter;
        if (distInt == 65535 || distInt == 0xFFFF) {
          broken_sensors[i] = 1;
          distInt = SENSOR_VL53_ERROR;
        }
      } else {
        // Если ошибка измерения, но сенсор работает, используем предыдущее значение
        switch (i) {
          case 0: distInt = prev_left; break;
          case 1: distInt = prev_right; break;
          case 2: distInt = prev_top; break;
          case 3: distInt = prev_back; break;
        }
      }
    }
    
    // Корректируем offset только если измерение валидно
    if (distInt != SENSOR_VL53_ERROR && distInt > STATIC_OFFSET) {
      distInt -= STATIC_OFFSET;
    }
    
    int* prev, *cur;
    switch (i) {
      case 0: prev = &prev_left; cur = &left; break;
      case 1: prev = &prev_right; cur = &right; break;
      case 2: prev = &prev_top; cur = &top; break;
      case 3: prev = &prev_back; cur = &back; break;
    }
    
    *prev = *cur;
    *cur = distInt;
    
    // Применяем фильтр только если оба значения валидны
    if (distInt != SENSOR_VL53_ERROR && *prev != SENSOR_VL53_ERROR &&
        abs(*cur - *prev) < FILTER_UP &&
        abs(*prev - *cur) < FILTER_DN) {
      *cur = *prev;
    }
  }

  // Восстановление сломанных сенсоров
  int i = loop_counter % NUM_SENSORS;
  if (broken_sensors[i]) {
    splitter.setBusChannel(sensors_pins[i]);
    delay(20); // Увеличенная задержка для восстановления
    if (lox.begin()) {
      lox.startRangeContinuous(L0X_RANGING_TIME_MS);
      broken_sensors[i] = 0;
      Serial.print("Sensor ");
      Serial.print(i);
      Serial.println(" restored");
    }
  }
}

void Indication() {
  if (onlyWeight) {
    widthBox = heightBox = lengthBox = 0;
    right = left = top = back = 0;
    return;
  }

  getDistance();

  // Проверяем диапазоны и устанавливаем флаги
  if ((right != SENSOR_VL53_ERROR) && (right / 10 > WIDTH_MAX)) right = SENSOR_VL53_OUT_RANGE;
  if ((left  != SENSOR_VL53_ERROR) && (left  / 10 > WIDTH_MAX)) left = SENSOR_VL53_OUT_RANGE;
  if ((top   != SENSOR_VL53_ERROR) && (top   / 10 > TOP_MAX))   top = SENSOR_VL53_OUT_RANGE;
  if ((back  != SENSOR_VL53_ERROR) && (back  / 10 > LENGTH_MAX)) back = SENSOR_VL53_OUT_RANGE;

  if (right != SENSOR_VL53_ERROR && right < DIST_MIN) right = SENSOR_VL53_COVERED;
  if (left  != SENSOR_VL53_ERROR && left  < DIST_MIN) left = SENSOR_VL53_COVERED;
  if (top   != SENSOR_VL53_ERROR && top   < DIST_MIN) top = SENSOR_VL53_COVERED;
  if (back  != SENSOR_VL53_ERROR && back  < DIST_MIN) back = SENSOR_VL53_COVERED;

  // Вычисляем размеры только если сенсоры работают корректно
  if (right >= SENSOR_VL53_OUT_RANGE || left >= SENSOR_VL53_OUT_RANGE || 
      top >= SENSOR_VL53_OUT_RANGE || back >= SENSOR_VL53_OUT_RANGE ||
      right == SENSOR_VL53_ERROR || left == SENSOR_VL53_ERROR ||
      top == SENSOR_VL53_ERROR || back == SENSOR_VL53_ERROR) {
    widthBox = heightBox = lengthBox = 0;
  } else {
    // Вычисляем размеры коробки
    int tempWidth = WIDTH_MAX;
    int tempHeight = TOP_MAX;
    int tempLength = LENGTH_MAX;
    
    if (right < SENSOR_VL53_COVERED && left < SENSOR_VL53_COVERED) {
      tempWidth = WIDTH_MAX - (right + left) / 10;
    } else {
      tempWidth = 0;
    }
    
    if (top < SENSOR_VL53_COVERED) {
      tempHeight = TOP_MAX - top / 10;
    } else {
      tempHeight = 0;
    }
    
    if (back < SENSOR_VL53_COVERED) {
      tempLength = LENGTH_MAX - back / 10;
    } else {
      tempLength = 0;
    }
    
    // Проверяем на разумные значения
    widthBox = (tempWidth > 0 && tempWidth < WIDTH_MAX) ? tempWidth : 0;
    heightBox = (tempHeight > 0 && tempHeight < TOP_MAX) ? tempHeight : 0;
    lengthBox = (tempLength > 0 && tempLength < LENGTH_MAX) ? tempLength : 0;
  }
  
  // Debug output
  Serial.print("Sensors: L=");
  Serial.print(left);
  Serial.print(" R=");
  Serial.print(right);
  Serial.print(" T=");
  Serial.print(top);
  Serial.print(" B=");
  Serial.print(back);
  Serial.print(" | Box: W=");
  Serial.print(widthBox);
  Serial.print(" H=");
  Serial.print(heightBox);
  Serial.print(" L=");
  Serial.println(lengthBox);
}

void SerialExchange() {
  if (Serial.available()) {
    byte incoming = Serial.read();
    switch (incoming) {
      case 0x90:
        while (Serial.available() < 1);
        TOP_MAX = Serial.read();
        return;
      case 0x91:
        while (Serial.available() < 1);
        WIDTH_MAX = Serial.read();
        return;
      case 0x92:
        while (Serial.available() < 1);
        LENGTH_MAX = Serial.read();
        return;
      case 0x93:
        for (int i = 0; i < NUM_SENSORS; i++) broken_sensors[i] = 1;
        return;
      case 0x95:
        start = true;
        Serial.write((const uint8_t[]){0x7F, 0, 0, 0, 0}, 5);
        return;
      case 0x88: {
        Indication();
        delay(100); 
        byte buf[13] = {
          0x2D, 0x0B, byte(widthBox), 0x7B,
          0x2D, 0x16, byte(heightBox), 0x7B,
          0x2D, 0x21, byte(lengthBox), 0x7B,
          byte(onlyWeight)
        };
        Serial.write(buf, sizeof(buf));
        Serial.flush(); // Принудительная отправка
        return;
      }
      case 0x89: {
        Indication();
        delay(100); 
        byte buf[41] = {
          0x2D, 0x0B, byte(left / 10), 0x7B,
          0x2D, 0xBB, byte(right / 10), 0x7B,
          0x2D, 0x16, byte(top / 10), 0x7B,
          0x2D, 0x21, byte(back / 10), 0x7B,
          0x2D, 0x0B, byte(WIDTH_MAX), 0x7B,
          0x2D, 0x16, byte(TOP_MAX), 0x7B,
          0x2D, 0x21, byte(LENGTH_MAX), 0x7B,
          0x2D, 0x0B, byte(widthBox), 0x7B,
          0x2D, 0x16, byte(heightBox), 0x7B,
          0x2D, 0x21, byte(lengthBox), 0x7B,
          byte(onlyWeight)
        };
        Serial.write(buf, sizeof(buf));
        Serial.flush(); // Принудительная отправка
        return;
      }
      case 0x66:
        digitalWrite(GREEN_LED_PIN, onlyWeight);
        digitalWrite(RED_LED_PIN, !onlyWeight);
        return;
      case 0x55:
        digitalWrite(RED_LED_PIN, LOW);
        digitalWrite(GREEN_LED_PIN, LOW);
        return;
      case 0x77:
        Serial.write("OK", 2);
        return;
    }
  }
  while (Serial.available()) Serial.read();
}

void setup() {
  pinMode(RED_LED_PIN, OUTPUT);
  pinMode(GREEN_LED_PIN, OUTPUT);
  pinMode(BUTTON, INPUT_PULLUP); // Используем встроенный подтягивающий резистор
  Serial.begin(115200);
  Serial.println("Arduino started!");
  Serial.setTimeout(250);
  while (!Serial) {}

  splitter.begin();
  delay(100); // Даем время для инициализации I2C hub

  // Инициализация всех сенсоров с увеличенными задержками
  for (int i = 0; i < NUM_SENSORS; i++) {
    splitter.setBusChannel(sensors_pins[i]);
    delay(50); 
    if (lox.begin()) {
      lox.startRangeContinuous(L0X_RANGING_TIME_MS);
      broken_sensors[i] = 0;
      Serial.print("Sensor ");
      Serial.print(i);
      Serial.println(" initialized");
    } else {
      Serial.print("Sensor ");
      Serial.print(i);
      Serial.println(" initialization failed");
      broken_sensors[i] = 1;
    }
  }
  
  // Инициализируем начальные значения
  right = left = top = back = SENSOR_VL53_ERROR;
  prev_right = prev_left = prev_top = prev_back = SENSOR_VL53_ERROR;
}

void loop() {
  if (loop_counter % (L0X_RANGING_TIME_MS / 10) == 0 && start)
    Indication();

  SerialExchange();

  if (digitalRead(BUTTON) == LOW) {
    onlyWeight = !onlyWeight;
    digitalWrite(RED_LED_PIN, !onlyWeight);
    digitalWrite(GREEN_LED_PIN, onlyWeight);
    delay(400);
  }

  loop_counter++;
  delay(10);
}