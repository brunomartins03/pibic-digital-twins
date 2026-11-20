# Gêmeo digital bidirecional de um braço robótico no Decentraland

Projeto de Iniciação Científica (PIBIC 2025/2026) do Laboratório de
Desenvolvimento de Sistemas (LDS) da FCTE/UnB.

O objetivo é prototipar um **gêmeo digital** de um equipamento do laboratório
na plataforma de metaverso **Decentraland**, com comunicação **bidirecional**
entre o objeto físico e sua representação virtual. Como prova de conceito, foi
escolhido um pequeno braço robótico com quatro servomotores SG90 controlado por
um **ESP8266**:

- um movimento do braço físico aparece em tempo quase real no Decentraland;
- um comando dado no Decentraland movimenta o braço físico.

## Arquitetura

```
┌──────────────┐  MQTT   ┌────────────┐  MQTT   ┌──────────────┐ WebSocket ┌──────────────┐
│ ESP8266 +    │◄───────►│  Broker    │◄───────►│ Relay server │◄─────────►│ Cena         │
│ 4 servos SG90│         │ (Mosquitto)│         │ (Go)         │           │ Decentraland │
└──────────────┘         └────────────┘         └──────────────┘           └──────────────┘
```

O microcontrolador fala **MQTT** (leve e adequado a IoT). As cenas do
Decentraland só conseguem abrir **WebSockets**. O *relay server* faz a ponte
entre os dois mundos e guarda o último estado conhecido do braço.

## Estrutura do repositório

| Caminho                                    | Conteúdo                                                   |
|--------------------------------------------|------------------------------------------------------------|
| [`codes/`](codes/README.md)                | Código-fonte e visão técnica geral do sistema               |
| [`codes/robot/`](codes/robot/README.md)    | Firmware do braço (Arduino/ESP8266) e simulação no Wokwi     |
| [`codes/relay-server/`](codes/relay-server/README.md) | Servidor intermediário MQTT ⇄ WebSocket em Go     |
| [`research/`](research/)                   | Relatório de pesquisa em LaTeX (`main.tex`) e figuras        |

## Início rápido

Consulte o [guia geral em `codes/README.md`](codes/README.md) para executar o
sistema completo (broker, relay, firmware) e para reproduzir o teste ponta a
ponta com o braço simulado no Wokwi e uma cena do Decentraland simulada.

## Status

| Componente                | Estado                                                      |
|---------------------------|-------------------------------------------------------------|
| Firmware do braço         | Compila para ESP8266 e ESP32; validado em simulação (Wokwi) |
| Relay server              | Testes unitários e de integração com MQTT passando           |
| Integração ponta a ponta  | Validada: Wokwi ⇄ broker MQTT ⇄ relay ⇄ cena simulada        |
| Cena do Decentraland      | Fora deste repositório; protocolo documentado                |
