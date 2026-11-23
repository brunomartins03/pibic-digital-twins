# Código-fonte do gêmeo digital

Esta pasta contém os módulos do sistema:

| Módulo                                   | Linguagem      | Papel                                                         |
|------------------------------------------|----------------|---------------------------------------------------------------|
| [`robot/`](robot/README.md)              | C++ (Arduino)  | Firmware do ESP8266: aciona os servos e publica/recebe estados |
| [`relay-server/`](relay-server/README.md)| Go             | Ponte MQTT ⇄ WebSocket entre o braço e as cenas do Decentraland |
| [`scene/`](scene/README.md)              | TypeScript (SDK7) | Cena do Decentraland: braço virtual, telemetria e comandos |
| [`deploy/`](deploy/README.md)            | Docker Compose | Ambiente completo: Catalyst, relay, cenas e comunicação        |

## Visão geral

```
            controle local (Serial: Q/A W/S E/D R/F)
                         │
                         ▼
┌─────────────────────────────────┐
│ ESP8266 (robot/)                │
│  ArmController ── MqttLink      │
└──────────────┬──────────────────┘
               │ MQTT  <prefixo>/state   (retido)  ESP → relay
               │       <prefixo>/cmd               relay → ESP
               │       <prefixo>/status  (retido, last will)
┌──────────────▼──────────────────┐
│ Broker MQTT (Mosquitto)         │
└──────────────┬──────────────────┘
               │ MQTT
┌──────────────▼──────────────────┐
│ relay-server                    │  GET /healthz, GET /api/state, POST /api/command
│  MQTTLink ── Hub ── Server      │
└──────────────┬──────────────────┘
               │ WebSocket /ws/scene ou HTTP (consulta + POST /api/command)
┌──────────────▼──────────────────┐
│ Cena do Decentraland            │  (ou cmd/scene-sim para testes)
└─────────────────────────────────┘
```

### Fluxos

1. **Físico → virtual.** Uma tecla na Serial (ou um comando) altera o alvo de
   uma junta; o `ArmController` move o servo gradualmente; a cada mudança
   (no máximo a cada 100 ms) o firmware publica o estado em `<prefixo>/state`;
   o relay recebe e envia `telemetry` para todas as cenas conectadas.
2. **Virtual → físico.** A cena envia `command` pelo WebSocket; o relay valida,
   publica em `<prefixo>/cmd` e responde `ack` (ou `error`); o ESP move o braço
   e a telemetria resultante fecha o ciclo.

O estado publicado é sempre o **real** (ângulo escrito nos servos), não o
alvo. Assim, o gêmeo virtual converge para o braço físico qualquer que seja a
origem do movimento.

### Decisões de projeto

- **MQTT no dispositivo, WebSocket na cena.** O ESP8266 tem pouca memória e o
  MQTT oferece mensagens retidas e *last will*; o Decentraland só suporta
  WebSocket. O relay isola essa diferença.
- **Estado completo em cada mensagem.** Comandos e telemetria sempre levam as
  três juntas e a garra, em graus (0–180). Mensagens são idempotentes e
  perdas não deixam o gêmeo inconsistente.
- **Último comando vence.** Não há fila nem trava entre cenas: o braço segue o
  comando mais recente, venha ele da Serial ou de qualquer cena.
- **Comandos não são retidos.** Um comando antigo nunca move o braço após um
  reinício; o estado, por outro lado, é retido para que o relay saiba a pose
  atual assim que se conecta.

O protocolo completo está em
[`relay-server/docs/protocolo.md`](relay-server/docs/protocolo.md).

### Implantação

No servidor do projeto, a cena está publicada no Catalyst (parcela 30,30) e o
relay roda em Docker atrás do nginx do Catalyst, em
`https://kiriland-metaverso.duckdns.org/robotarm/`. Como o proxy TLS da
universidade não repassa WebSocket, a cena publicada usa o modo HTTP. Detalhes
em [`relay-server/docs/implantacao.md`](relay-server/docs/implantacao.md).

## Execução do sistema completo

Pré-requisitos: Go ≥ 1.22, Mosquitto, `arduino-cli` (com os *cores* ESP8266 e
ESP32) — detalhes em [`robot/docs/firmware.md`](robot/docs/firmware.md) e
[`relay-server/docs/operacao.md`](relay-server/docs/operacao.md).

1. **Broker:** instale o Mosquitto e libere conexões da rede local
   (veja [operação do relay](relay-server/docs/operacao.md#1-broker-mqtt)).
2. **Relay:**
   ```bash
   cd relay-server
   MQTT_BROKER=tcp://localhost:1883 SCENE_TOKEN=segredo go run ./cmd/relay
   ```
3. **Firmware:**
   ```bash
   cd robot
   cp config.example.h config.h   # preencha WiFi e MQTT_HOST (IP do broker)
   make upload PORT=/dev/ttyUSB0
   ```
4. **Cena:** `cd scene && npm install && npm start` (ajuste `RELAY_URL` e
   `SCENE_TOKEN` em `scene/src/config.ts`; ver [testes da cena](scene/docs/testes.md)),
   ou use o simulador: `go run ./cmd/scene-sim -token segredo`.

## Teste ponta a ponta sem hardware

O braço pode ser simulado no Wokwi (ESP32) e a cena no `scene-sim`. O passo a
passo, incluindo a configuração de broker público necessária porque o Wokwi
roda na nuvem, está em
[`relay-server/docs/testes.md`](relay-server/docs/testes.md#3-teste-ponta-a-ponta-wokwi).
