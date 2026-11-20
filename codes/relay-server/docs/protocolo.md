# Protocolo de comunicação

Há dois enlaces: **dispositivo ⇄ relay** por MQTT e **cena ⇄ relay** por
WebSocket. Ambos usam o mesmo formato de estado.

## Estado do braço (`ArmState`)

```json
{"joints": [90, 45, 120], "gripper": 30}
```

| Campo     | Tipo         | Descrição                                               |
|-----------|--------------|---------------------------------------------------------|
| `joints`  | array de 3 números | Ângulos em graus de `[base, ombro, cotovelo]`, 0–180 |
| `gripper` | número       | Ângulo da garra em graus, 0–180                          |
| `updatedAt` | string RFC 3339 | Apenas nas mensagens do relay para a cena: instante em que o relay recebeu o estado |

A ordem das juntas é fixa e deve ser a mesma no firmware (`JointIndex`), no
relay (`JointCount`) e na cena. Estados fora dessas regras são descartados
pelo relay.

## MQTT (dispositivo ⇄ relay)

Todos os tópicos ficam sob um prefixo configurável (`MQTT_TOPIC_PREFIX` no
relay e no firmware; padrão `robotarm`).

| Tópico              | Direção        | QoS | Retido | Payload                       |
|---------------------|----------------|-----|--------|-------------------------------|
| `<prefixo>/state`   | ESP → relay    | 0   | sim    | `ArmState` (sem `updatedAt`)  |
| `<prefixo>/cmd`     | relay → ESP    | 0   | não    | `ArmState` (sem `updatedAt`)  |
| `<prefixo>/status`  | ESP → relay    | 1   | sim    | `online` ou `offline`         |

- **`state`** é publicado sempre que o ângulo real de algum servo muda (no
  máximo a cada `TELEMETRY_MIN_INTERVAL_MS`, 100 ms por padrão) e logo após
  cada (re)conexão. Por ser retido, o relay recebe a pose atual assim que se
  inscreve, mesmo que tenha sido reiniciado.
- **`cmd`** contém os ângulos-alvo. O firmware limita os valores a 0–180 e
  move os servos gradualmente. Não é retido, para que um comando antigo nunca
  mova o braço depois de um reinício.
- **`status`** é publicado como `online` na conexão e registrado como
  *last will* `offline`: se o ESP perder energia ou rede, o broker publica
  `offline` sozinho.

Teste manual com os clientes do Mosquitto:

```bash
mosquitto_sub -h localhost -t 'robotarm/#' -v
mosquitto_pub -h localhost -t robotarm/cmd -m '{"joints":[45,120,60],"gripper":10}'
```

## WebSocket (cena ⇄ relay)

Endpoint: `ws://<host>:8080/ws/scene`, com `?token=<SCENE_TOKEN>` quando o
token estiver configurado (sem ele a conexão recebe HTTP 401).

Todas as mensagens são um envelope JSON:

```json
{"type": "...", "id": "...", "state": {...}, "deviceOnline": true, "message": "..."}
```

Somente os campos relevantes para cada tipo são enviados.

### Do relay para a cena

| `type`      | Campos               | Quando                                                       |
|-------------|----------------------|--------------------------------------------------------------|
| `status`    | `deviceOnline`       | Na conexão e sempre que o dispositivo fica online/offline     |
| `telemetry` | `state`              | Na conexão (se já houver estado) e a cada estado do dispositivo |
| `ack`       | `id`                 | Resposta a um `command` aceito e publicado                     |
| `error`     | `id`, `message`      | Resposta a uma mensagem rejeitada                              |

Mensagens de erro possíveis: `device offline`, `command without state`,
`expected 3 joints, got N`, `joint N angle X out of range [0, 180]`,
`gripper angle X out of range [0, 180]`, `invalid JSON: ...`,
`unsupported message type "..."`, `failed to forward command`.

### Da cena para o relay

| `type`    | Campos                      | Efeito                                        |
|-----------|-----------------------------|-----------------------------------------------|
| `command` | `state`, `id` (opcional)    | Publica `state` em `<prefixo>/cmd`             |

O `id` é livre e é devolvido no `ack`/`error`, permitindo correlacionar
respostas. Um `ack` significa que o comando foi entregue ao broker, não que o
braço já chegou à posição: a confirmação real é a `telemetry` subsequente.

### Exemplo de sessão

```
<- {"type":"status","deviceOnline":true}
<- {"type":"telemetry","state":{"joints":[90,90,90],"gripper":90,"updatedAt":"..."}}
-> {"type":"command","id":"1","state":{"joints":[45,120,60],"gripper":10}}
<- {"type":"ack","id":"1"}
<- {"type":"telemetry","state":{"joints":[88,92,88],"gripper":88,"updatedAt":"..."}}
...
<- {"type":"telemetry","state":{"joints":[45,120,60],"gripper":10,"updatedAt":"..."}}
```

## HTTP

| Rota              | Resposta                                                        |
|-------------------|-----------------------------------------------------------------|
| `GET /healthz`    | `{"ok":true,"deviceOnline":<bool>}`                              |
| `GET /api/state`  | Último `ArmState` recebido, ou `204 No Content` se ainda não houver |

As rotas HTTP aceitam CORS de qualquer origem e não exigem token, pois são
somente leitura.
