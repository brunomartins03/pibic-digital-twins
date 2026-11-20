# Arquitetura do relay server

## Origem

Este módulo resulta da fusão de duas versões anteriores:

| Versão anterior   | Pontos aproveitados                                  | Problemas                                            |
|-------------------|------------------------------------------------------|------------------------------------------------------|
| `relay-server` (MQTT + WebSocket) | Uso de MQTT com estado retido para o dispositivo | Protocolo incompatível com o firmware; escritas concorrentes no mesmo WebSocket; sem autenticação nem desligamento gracioso |
| `relay-server2` (somente WebSocket) | Hub de cenas, token, `/healthz`, `/api/state`, *keepalive*, desligamento gracioso | O dispositivo usava WebSocket, abandonando a estratégia MQTT |

A versão atual mantém a estrutura do `relay-server2` e substitui a conexão
WebSocket do dispositivo pelo enlace MQTT, acrescentando validação de
comandos e status do dispositivo por *last will*.

## Organização do código

```
cmd/relay/main.go          configuração por variáveis de ambiente e ciclo de vida
cmd/scene-sim/main.go      cliente de teste que se comporta como uma cena
internal/relay/
├── protocol.go            ArmState, Envelope, constantes e validação
├── hub.go                 estado em memória e difusão para as cenas
├── server.go              rotas HTTP e WebSocket (/ws/scene, /api/state, /healthz)
└── mqtt.go                MQTTLink: inscrição em state/status e publicação de cmd
```

As dependências seguem uma única direção:

```
MQTTLink ──UpdateState/SetDeviceOnline──► Hub ◄──register/sendTo── Server
   ▲                                                                  │
   └──────────────────── PublishCommand (CommandPublisher) ───────────┘
```

`Server` depende apenas da interface `CommandPublisher`, o que permite testá-lo
com um publicador falso, sem broker.

## Concorrência

- O `Hub` protege estado, status e conjunto de clientes com um único mutex.
- Cada cena tem uma fila (`send`, 32 mensagens) e **uma única goroutine de
  escrita** (`writePump`), exigência da biblioteca `gorilla/websocket`, que não
  permite escritas concorrentes. Respostas (`ack`/`error`), telemetria e
  *pings* passam todos por ela.
- A goroutine de leitura (`readPump`) só lê e enfileira respostas.
- Se a fila de uma cena encher (cliente lento), ela é desconectada em vez de
  atrasar as demais.
- O estado inicial (`status` + `telemetry`) é enfileirado sob o mesmo mutex do
  registro, garantindo que nenhuma atualização chegue antes dele.

## Robustez

- O cliente MQTT reconecta automaticamente e refaz as inscrições a cada
  conexão; a tentativa inicial também é repetida, então o relay pode subir
  antes do broker.
- *Ping/pong* a cada 54 s detecta cenas que caíram sem fechar a conexão.
- Mensagens maiores que 4 KiB são recusadas.
- `SIGINT`/`SIGTERM` encerram o servidor HTTP de forma graciosa.

## Limitações conhecidas

- O estado vive apenas em memória; após reiniciar, o relay depende da
  mensagem retida no broker para recuperá-lo.
- `CheckOrigin` aceita qualquer origem; o controle de acesso é feito pelo
  `SCENE_TOKEN`. Em produção, use `wss://` por trás de um proxy reverso com TLS,
  já que o token trafega na URL.
- Não há arbitragem entre cenas: o último comando prevalece.
