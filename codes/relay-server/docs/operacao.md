# Operação do relay server

## 1. Broker MQTT

O relay e o ESP8266 se encontram em um broker MQTT. Em uma máquina Linux
local, o Mosquitto é suficiente:

```bash
sudo apt-get install mosquitto mosquitto-clients
```

Por padrão o Mosquitto 2.x só aceita conexões de `localhost`. Para que o
ESP8266 se conecte pela rede local, crie `/etc/mosquitto/conf.d/robotarm.conf`:

```
listener 1883 0.0.0.0
allow_anonymous true
```

`allow_anonymous true` só é aceitável em uma rede local fechada. Caso
contrário, crie usuários com `mosquitto_passwd` e configure `password_file`,
preenchendo `MQTT_USERNAME`/`MQTT_PASSWORD` no relay e no firmware.

```bash
sudo systemctl restart mosquitto
mosquitto_sub -h localhost -t 'robotarm/#' -v   # verificação
```

Verifique antes se a porta 1883 está livre (`ss -ltn | grep 1883`): se já
houver outro broker ouvindo, reutilize-o ou escolha outra porta.

## 2. Configuração

Todas as opções são variáveis de ambiente:

| Variável            | Padrão                  | Descrição                                              |
|---------------------|-------------------------|--------------------------------------------------------|
| `RELAY_ADDR`        | `:8080`                 | Endereço HTTP/WebSocket                                 |
| `SCENE_TOKEN`       | *(vazio)*               | Token exigido em `/ws/scene?token=`; vazio desativa a autenticação |
| `MQTT_BROKER`       | `tcp://localhost:1883`  | URL do broker: `tcp://`, `ssl://`, `ws://` ou `wss://`  |
| `MQTT_CLIENT_ID`    | `robotarm-relay`        | Deve ser único no broker                                |
| `MQTT_USERNAME`     | *(vazio)*               | Usuário MQTT (opcional)                                 |
| `MQTT_PASSWORD`     | *(vazio)*               | Senha MQTT (opcional)                                   |
| `MQTT_TOPIC_PREFIX` | `robotarm`              | Prefixo dos tópicos; igual ao `MQTT_TOPIC_PREFIX` do firmware |

## 3. Execução

```bash
go run ./cmd/relay
# ou
go build -o relay ./cmd/relay && ./relay
```

Saída esperada:

```
mqtt: connecting to tcp://localhost:1883
http: listening on :8080 (topic prefix "robotarm")
mqtt: connected to tcp://localhost:1883
mqtt: device status "online"          # quando o ESP8266 se conectar
```

Verificação:

```bash
curl localhost:8080/healthz      # {"deviceOnline":true,"ok":true}
curl localhost:8080/api/state    # último estado do braço
```

## 4. Simulador de cena

`scene-sim` substitui a cena do Decentraland durante o desenvolvimento.

```bash
go run ./cmd/scene-sim                          # interativo: digite "45 120 60 10" ou "home"
go run ./cmd/scene-sim -send "45 120 60 10;home" -interval 5s -wait 10s   # roteirizado
```

| Flag        | Padrão                          | Descrição                                       |
|-------------|---------------------------------|-------------------------------------------------|
| `-url`      | `ws://localhost:8080/ws/scene`  | Endpoint do relay                                |
| `-token`    | `$SCENE_TOKEN`                  | Token da cena                                    |
| `-send`     | *(vazio = stdin)*               | Comandos separados por `;` (`base ombro cotovelo garra`) |
| `-interval` | `3s`                            | Intervalo entre comandos roteirizados             |
| `-wait`     | `5s`                            | Tempo escutando após o último comando             |

## 5. Conectando a cena do Decentraland

Na cena, abra um WebSocket para `ws://<ip-do-relay>:8080/ws/scene?token=...`,
atualize o modelo 3D a cada `telemetry` e envie `command` com o estado
completo desejado (veja o [protocolo](protocolo.md)). Se o WebSocket não for
possível, consulte `/healthz` e `/api/state` periodicamente e envie comandos
por `POST /api/command`. A cena em [`scene/`](../../scene/README.md) suporta
os dois modos.

Cenas publicadas fora do modo de pré-visualização exigem `https://`/`wss://`:
exponha o relay por trás de um proxy reverso com TLS.

### Publicação pelo nginx do Catalyst

No servidor do projeto, o relay é publicado em
`https://kiriland-metaverso.duckdns.org/robotarm/` por uma rota no nginx do
Catalyst, `/opt/catalyst/catalyst-owner/local/nginx/include/robotarm.conf`
(incluída em `routes.conf`):

```nginx
location /robotarm/ {
    proxy_pass http://172.20.0.1:8080/;   # gateway da rede catalyst-owner_default
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_read_timeout 3600s;
    proxy_send_timeout 3600s;
}
```

O relay deve ouvir no gateway da rede Docker para que o contêiner do nginx o
alcance:

```bash
RELAY_ADDR=172.20.0.1:8080 go run ./cmd/relay
```

Recarregue com `docker exec nginx nginx -t && docker exec nginx nginx -s reload`.
O proxy TLS da universidade, à frente do nginx, não repassa o *upgrade* de
WebSocket para essa rota; por isso a cena publicada usa o modo HTTP.
Sem `SCENE_TOKEN`, qualquer pessoa com a URL pode enviar comandos; e, mesmo
com token, ele fica visível no código da cena publicada.
