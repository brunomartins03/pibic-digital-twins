# Simulação no Wokwi

O [Wokwi](https://wokwi.com) não simula o ESP8266. Como o firmware usa apenas
APIs disponíveis nas duas plataformas (Wi-Fi, servos, PubSubClient,
ArduinoJson), ele é compilado também para ESP32, que o Wokwi suporta. As
únicas diferenças estão em diretivas `#if defined(ESP32)`:

| Recurso   | ESP8266 (placa real) | ESP32 (simulação) |
|-----------|----------------------|-------------------|
| Wi-Fi     | `ESP8266WiFi.h`      | `WiFi.h`          |
| Servos    | `Servo.h` (do *core*) | `ESP32Servo.h`, 50 Hz |

Os pinos simulados têm os mesmos números de GPIO da placa real (5, 4, 14 e 12).

## Arquivos

| Arquivo                      | Descrição                                                   |
|------------------------------|-------------------------------------------------------------|
| `simulation/diagram.json`    | ESP32 DevKit C v4 com quatro servos ligados aos GPIOs 5, 4, 14 e 12 |
| `simulation/wokwi.toml`      | Aponta para o firmware em `build/esp32/`                     |
| `config.wokwi.h`             | Configuração usada quando compilado com `-DWOKWI_SIM`        |

`config.wokwi.h` usa a rede `Wokwi-GUEST` e o broker público `broker.emqx.io`
com o prefixo aleatório `dt-sim-7f3a9c`. Um broker público é necessário porque
o `wokwi-cli` executa a simulação na nuvem, sem acesso ao `localhost` da sua
máquina.

## Execução

1. Gere um token em <https://wokwi.com/dashboard/ci> e exporte-o:
   ```bash
   export WOKWI_CLI_TOKEN=...
   ```
2. Compile e simule (as teclas digitadas vão para a Serial simulada):
   ```bash
   make sim                    # SIM_TIMEOUT=60000 por padrão (ms)
   ```
3. Alternativamente, abra a pasta `simulation/` com a extensão Wokwi do VS
   Code para ver os servos girando.

Saída esperada:

```
Robotic arm - digital twin firmware
[wifi] connecting to Wokwi-GUEST......
[wifi] connected, IP: 10.13.37.2
[mqtt] connecting to broker.emqx.io:1883 as robotarm-sim-e9d
[mqtt] connected, listening on dt-sim-7f3a9c/cmd
```

No plano gratuito do Wokwi, a simulação pelo CLI é encerrada após 5 minutos
(`API Error: Connection timed out after 5 minutes`), mesmo com um
`SIM_TIMEOUT` maior; rode uma simulação por teste.

Para o teste completo com relay e cena simulada, veja
[`relay-server/docs/testes.md`](../../relay-server/docs/testes.md#3-teste-ponta-a-ponta-wokwi).

## Resultados da análise

Validado em 20/11/2026:

| Verificação                                   | Resultado |
|-----------------------------------------------|-----------|
| Compilação ESP8266 (NodeMCU)                  | OK — 29,6 KB de RAM (36%) |
| Compilação ESP32                              | OK — 994 KB de flash (75%) |
| Boot, Wi-Fi e conexão MQTT no Wokwi           | OK        |
| Comando remoto move os servos suavemente      | OK — trajetória de 90° até `[45,120,60]`, garra 10 |
| Teclas na Serial movem o braço e publicam estado | OK — `qqqq`: base de 90° a 110° |
| Reconexão MQTT automática com *last will*      | OK        |

A simulação roda mais devagar que o tempo real e pode provocar quedas de
*keepalive* MQTT com o braço parado; ver
[particularidades da simulação](../../relay-server/docs/testes.md#particularidades-da-simulação).
