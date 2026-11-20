# Testes

## 1. Testes unitários

```bash
go vet ./...
go test -race ./...
```

Cobrem, sem broker real (`internal/relay/server_test.go`):

- validação de `ArmState` (quantidade de juntas, limites 0–180);
- envio de `status` + `telemetry` ao conectar e difusão para várias cenas;
- encaminhamento de comandos: `ack`, `error` com dispositivo offline e
  rejeição de JSON inválido, estados inválidos e tipos desconhecidos;
- autenticação por `SCENE_TOKEN` (HTTP 401 sem token);
- `/api/state` (204 antes da primeira telemetria) e `/healthz`.

## 2. Teste de integração com MQTT

`TestMQTTLinkIntegration` usa um broker real e faz o papel do dispositivo:
publica `status`/`state` retidos, verifica que o relay os recebe, que um
comando chega a `<prefixo>/cmd` e que `offline` é refletido. Ele é ignorado
quando `MQTT_TEST_BROKER` não está definida e usa um prefixo único, limpando as
mensagens retidas ao final.

```bash
MQTT_TEST_BROKER=tcp://localhost:1883 go test -race -run Integration -v ./internal/relay
```

## 3. Teste ponta a ponta (Wokwi)

Valida o sistema completo sem hardware: firmware real rodando em um ESP32
simulado no Wokwi → broker MQTT → relay → `scene-sim` no papel do
Decentraland.

### Por que um broker público

O `wokwi-cli` executa a simulação nos servidores do Wokwi; o ESP32 simulado
tem acesso à internet, mas **não** alcança o `localhost` da sua máquina (isso
exige o *Private IoT Gateway*, pago). Por isso, o firmware de simulação
(`robot/config.wokwi.h`) usa o broker público `broker.emqx.io` com um prefixo
de tópicos aleatório, e o relay local se conecta ao mesmo broker. Trafegam
apenas ângulos fictícios — não use esse caminho com dados sensíveis.

Se a sua rede bloquear a porta 1883 de saída (comum em redes
institucionais), conecte o relay pela porta WebSocket com TLS do mesmo broker,
`wss://broker.emqx.io:8084/mqtt`. O ESP32 simulado continua em TCP 1883, pois
sua conexão sai da nuvem do Wokwi.

### Passo a passo

Requer `WOKWI_CLI_TOKEN` (gerado em <https://wokwi.com/dashboard/ci>) e o
firmware compilado (`make build-sim`, veja
[a simulação do robô](../../robot/docs/simulacao.md)). Use três terminais:

```bash
# 1. relay, no mesmo broker e prefixo do firmware simulado
cd codes/relay-server
MQTT_BROKER=wss://broker.emqx.io:8084/mqtt MQTT_TOPIC_PREFIX=dt-sim-7f3a9c \
  MQTT_CLIENT_ID=relay-sim-$RANDOM go run ./cmd/relay

# 2. braço simulado (teclas digitadas vão para a Serial: q/a w/s e/d r/f)
cd codes/robot
make sim

# 3. cena simulada, após o log "[mqtt] connected" do passo 2
cd codes/relay-server
go run ./cmd/scene-sim -send "45 120 60 10;home" -interval 10s -wait 30s
```

### Resultado obtido (20/11/2026)

- **Decentraland → braço:** os dois comandos receberam `ack`, o ESP32 registrou
  `[mqtt] command: base=45 shoulder=120 elbow=60 gripper=10` e a cena recebeu a
  telemetria da trajetória suave até `[45,120,60]`/`10` e de volta a `90`.
- **Braço → Decentraland:** as teclas `qqqq` na Serial levaram a base de 90° a
  110°, e cada passo chegou à cena como `telemetry`; `/api/state` retornou
  `{"joints":[110,90,90],"gripper":90,...}`.
- **Status:** `/healthz` indicou `deviceOnline: true`; o relay registrou
  `device status "online"` e, na queda descrita abaixo, `"offline"` (via
  *last will*) seguido de `"online"` após a reconexão automática.

### Particularidades da simulação

- O Wokwi roda mais devagar que o tempo real quando há Wi-Fi simulado (cerca
  de 1/8 nos testes): passos de 100 ms de tempo simulado chegaram com ~0,8 s de
  intervalo. Os movimentos parecem lentos, mas os valores estão corretos.
- Pelo mesmo motivo, o *keepalive* MQTT de 15 s (tempo simulado) pode exceder o
  limite do broker (tempo real) quando o braço fica parado; o broker então
  derruba a conexão, publica `offline` e o firmware reconecta sozinho em
  seguida. Isso não ocorre no hardware real.
