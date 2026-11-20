# Relay server

Servidor intermediário em Go que conecta o braço robótico (via **MQTT**) às
cenas do Decentraland (via **WebSocket**). Mantém em memória o último estado do
braço e o status do dispositivo, repassa a telemetria para todas as cenas e
encaminha os comandos das cenas para o braço.

```bash
go test ./...                                  # testes
MQTT_BROKER=tcp://localhost:1883 go run ./cmd/relay
go run ./cmd/scene-sim                         # cena do Decentraland simulada
```

## Documentação

| Documento                                   | Conteúdo                                                  |
|---------------------------------------------|-----------------------------------------------------------|
| [Arquitetura](docs/arquitetura.md)          | Organização do código, concorrência e decisões de projeto  |
| [Protocolo](docs/protocolo.md)              | Tópicos MQTT, mensagens WebSocket e endpoints HTTP         |
| [Operação](docs/operacao.md)                | Instalação do broker, configuração e execução             |
| [Testes](docs/testes.md)                    | Testes unitários, de integração e ponta a ponta (Wokwi)    |

## Estrutura

```
relay-server/
├── cmd/
│   ├── relay/        # executável do servidor
│   └── scene-sim/    # cliente que simula uma cena do Decentraland
├── internal/relay/   # protocolo, hub, servidor WebSocket/HTTP e link MQTT
└── docs/
```
