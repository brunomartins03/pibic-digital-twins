# Firmware

## Ambiente

O projeto é compilado com o [`arduino-cli`](https://arduino.github.io/arduino-cli/)
(o Arduino IDE 2 também funciona abrindo `robot.ino`).

```bash
arduino-cli config init
arduino-cli config set board_manager.additional_urls \
  https://arduino.esp8266.com/stable/package_esp8266com_index.json \
  https://espressif.github.io/arduino-esp32/package_esp32_index.json
arduino-cli core update-index
arduino-cli core install esp8266:esp8266 esp32:esp32
arduino-cli lib install PubSubClient ArduinoJson ESP32Servo
```

Versões validadas:

| Componente     | Versão | Uso                                          |
|----------------|--------|----------------------------------------------|
| esp8266:esp8266| 3.1.2  | Placa real (inclui a biblioteca `Servo`)      |
| esp32:esp32    | 4.0.0  | Simulação no Wokwi                            |
| PubSubClient   | 2.8    | Cliente MQTT                                  |
| ArduinoJson    | 7.4.3  | JSON (API `JsonDocument` da v7)               |
| ESP32Servo     | 3.2.1  | Servos no ESP32 (apenas simulação)             |

## Configuração

Copie `config.example.h` para `config.h` (ignorado pelo Git) e preencha:

| Macro                       | Descrição                                               |
|-----------------------------|---------------------------------------------------------|
| `WIFI_SSID`, `WIFI_PASSWORD`| Rede Wi-Fi (2,4 GHz)                                     |
| `MQTT_HOST`, `MQTT_PORT`    | Broker MQTT; por padrão a implantação EMQX do projeto, porta 8883 |
| `MQTT_TLS`                  | `1` para conectar com TLS usando a CA de `mqtt_ca.h`; `0` para TCP simples (porta 1883) |
| `MQTT_USERNAME`, `MQTT_PASSWORD` | Credenciais do broker (`robotarm-device` no EMQX); vazio para acesso anônimo |
| `MQTT_CLIENT_ID`            | Base do ID do cliente (recebe um sufixo aleatório)       |
| `MQTT_TOPIC_PREFIX`         | Prefixo dos tópicos; igual ao do relay                   |
| `PIN_BASE` … `PIN_GRIPPER`  | GPIOs dos servos ([hardware](hardware.md))               |
| `STEP_DEG`                  | Graus por tecla na Serial (5)                            |
| `MOVE_STEP_DEG`, `MOVE_TICK_MS` | Velocidade: 2° a cada 20 ms = 100°/s                 |
| `TELEMETRY_MIN_INTERVAL_MS` | Intervalo mínimo entre publicações de estado (100 ms)    |

### TLS

O broker do projeto (EMQX Serverless) só aceita conexões com TLS. Com
`MQTT_TLS` em `1`, o `MqttLink` usa `WiFiClientSecure` (BearSSL no ESP8266,
mbedTLS no ESP32) e valida o certificado do broker com a CA de `mqtt_ca.h`
(DigiCert Global Root G2, válida até 2038, cópia de `codes/emqxsl-ca.crt`).

A validação depende da data atual, então o firmware sincroniza o relógio por
NTP (`pool.ntp.org`) e só tenta o *handshake* depois disso; até lá, a Serial
mostra `[mqtt] waiting for NTP time to validate the broker certificate`. O
*handshake* bloqueia o `loop()` por cerca de 1 a 2 s a cada (re)conexão.

A rede da placa precisa permitir a porta 8883 de saída. Se o broker mudar de
autoridade certificadora, substitua o conteúdo de `mqtt_ca.h`.

## Compilação e gravação

| Comando                         | Ação                                                   |
|---------------------------------|--------------------------------------------------------|
| `make build`                    | Compila para `esp8266:esp8266:nodemcuv2` em `build/esp8266` |
| `make upload PORT=/dev/ttyUSB0` | Compila e grava                                         |
| `make monitor`                  | Monitor serial a 115200 baud                            |
| `make build-sim` / `make sim`   | Compila para ESP32 com `-DWOKWI_SIM` / simula no Wokwi  |
| `make clean`                    | Remove `build/`                                        |

Para outra placa ESP8266, sobrescreva o FQBN: `make build FQBN=esp8266:esp8266:d1_mini`.

## Funcionamento

### Laço principal (`robot.ino`)

A cada iteração de `loop()`:

1. `MqttLink::loop()` mantém a conexão MQTT e reconecta sem bloquear (uma
   tentativa a cada 5 s, apenas com Wi-Fi conectado);
2. as teclas recebidas pela Serial viram novos alvos;
3. `ArmController::update()` avança os servos em direção aos alvos;
4. se o ângulo real mudou (ou houve reconexão), o estado é publicado em
   `<prefixo>/state`, respeitando o intervalo mínimo.

Nada no laço bloqueia: o controle local continua funcionando mesmo sem Wi-Fi
ou broker. No `setup()`, a espera pelo Wi-Fi é limitada a 15 s; depois disso a
reconexão continua em segundo plano.

### Controle local (Serial, 115200 baud)

| Junta    | Aumenta | Diminui |
|----------|---------|---------|
| Base     | `Q`     | `A`     |
| Ombro    | `W`     | `S`     |
| Cotovelo | `E`     | `D`     |
| Garra    | `R`     | `F`     |

Maiúsculas ou minúsculas; cada tecla altera o alvo em `STEP_DEG` graus.

### Movimento (`ArmController`)

Controle local e remoto apenas definem **alvos**. A cada `MOVE_TICK_MS`, cada
servo anda no máximo `MOVE_STEP_DEG` graus em direção ao seu alvo, produzindo
um movimento suave independentemente da origem do comando. O estado
publicado é o ângulo **atual** escrito nos servos, arredondado para graus
inteiros.

### Comunicação (`MqttLink`)

- Conecta com *last will* `offline` (retido) em `<prefixo>/status` e publica
  `online` logo após conectar.
- Inscreve-se em `<prefixo>/cmd`; comandos sem as 3 juntas e a garra são
  ignorados e registrados na Serial.
- Publica o estado em `<prefixo>/state` com a flag *retain*.

O formato das mensagens está em
[`relay-server/docs/protocolo.md`](../../relay-server/docs/protocolo.md).

## Correções em relação à versão anterior

| Problema                                                                 | Correção                                       |
|--------------------------------------------------------------------------|------------------------------------------------|
| Falava WebSocket diretamente com o servidor, contrariando a estratégia MQTT | Novo `MqttLink` (PubSubClient)              |
| GPIOs 26 e 27 inexistentes no ESP8266 (cotovelo e garra parados)          | Pinos D1, D2, D5, D6                           |
| `update()` avançava a cada iteração do `loop()` (milhares por segundo): os servos "saltavam" para o alvo e a telemetria inundava o canal | Movimento temporizado (`MOVE_TICK_MS`) e telemetria limitada |
| Telemetria com ângulos fracionários (ex.: `91.99999`)                    | Ângulos arredondados                           |
| Documentação e comentários referentes ao ESP32                           | Firmware portável ESP8266/ESP32 e documentação revisada |
