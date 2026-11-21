# Implantação no servidor do projeto

Este documento registra como o sistema foi publicado no servidor do
laboratório e as mudanças feitas fora do repositório para isso: a rota no
nginx do Catalyst, o contorno do proxy da universidade e a publicação da cena.
Validado em 20/11/2026.

## Visão geral

```
navegador (cliente web do Decentraland)
   │  HTTPS  https://kiriland-metaverso.duckdns.org
   ▼
proxy TLS da universidade (192.168.122.1, fora do nosso controle)
   │  HTTP
   ▼
nginx do Catalyst (contêiner "nginx", porta 80 da VM 192.168.122.84)
   ├── /content, /lambdas, ...   → Catalyst (cenas publicadas)
   └── /robotarm/                → relay (172.20.0.1:8080)
                                      │  MQTT
                                      ▼
                                   broker (broker.emqx.io no teste com Wokwi)
                                      │  MQTT
                                      ▼
                                   braço (ESP8266 ou ESP32 no Wokwi)
```

| Item                   | Valor                                                        |
|------------------------|--------------------------------------------------------------|
| Catalyst               | `/opt/catalyst/catalyst-owner`, realm `test`                 |
| URL pública            | <https://kiriland-metaverso.duckdns.org>                     |
| Cena                   | parcela 30,30 — <https://decentraland.zone/play/?realm=https%3A%2F%2Fkiriland-metaverso.duckdns.org&position=30%2C30> |
| Relay                  | `https://kiriland-metaverso.duckdns.org/robotarm/`           |
| Rede Docker do Catalyst | `catalyst-owner_default`, gateway `172.20.0.1`               |

## 1. Relay em Docker

O relay roda no contêiner `robotarm-relay` (ver `Dockerfile` e
`compose.yaml`). A porta é publicada apenas no gateway da rede do Catalyst,
`172.20.0.1:8080`: o contêiner do nginx o alcança por esse endereço e ele não
fica exposto nas interfaces públicas da VM.

```bash
cd codes/relay-server
cp .env.example .env        # broker, prefixo e token
docker compose up -d --build
docker compose logs -f      # "mqtt: connected", "device status ..."
curl 172.20.0.1:8080/healthz
```

O contêiner reinicia sozinho (`restart: unless-stopped`) e tem
*healthcheck* em `/healthz`. Para parar: `docker compose down`.

O `.env.example` usa o broker público e o prefixo da simulação no Wokwi. Para
o braço físico com o Mosquitto da VM, aponte `MQTT_BROKER` para um endereço da
VM visível pelo contêiner (por exemplo `tcp://172.20.0.1:1883`) e configure o
Mosquitto para ouvir nele: hoje ele ouve só em `127.0.0.1`.

## 2. Rota no nginx do Catalyst

O nginx do Catalyst lê suas rotas de
`/opt/catalyst/catalyst-owner/local/nginx/include/`. Foram feitas duas
alterações (como `root`):

1. Novo arquivo `include/robotarm.conf`:
   ```nginx
   # Robotic arm digital twin: relay server (pibic-digital-twins/codes/relay-server)
   # running on the host, reachable through the catalyst-owner_default gateway.
   location /robotarm/ {
       proxy_pass http://172.20.0.1:8080/;
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
2. Linha acrescentada ao fim de `include/routes.conf`:
   ```nginx
   include /etc/nginx/include/robotarm.conf;
   ```

Aplicação, com validação antes de recarregar:

```bash
docker exec nginx nginx -t && docker exec nginx nginx -s reload
```

A barra final em `proxy_pass` remove o prefixo: `/robotarm/healthz` chega ao
relay como `/healthz`. O endereço é um IP, e não o nome de um contêiner, para
que o nginx inicie mesmo com o relay parado; nesse caso a rota responde 502.

Para desfazer: apague `robotarm.conf` e a linha do `routes.conf` e recarregue
o nginx.

## 3. Problema: o proxy da universidade bloqueia WebSocket

A VM fica atrás de um proxy TLS da universidade (`192.168.122.1`, origem das
requisições no log do nginx), que termina o HTTPS e repassa HTTP ao nginx
(`CATALYST_TLS_AT_PROXY=1` no `.env` do Catalyst). Esse proxy só repassa o
*upgrade* de WebSocket para as rotas que já conhece:

| Teste de *handshake* WebSocket                        | Resultado |
|-------------------------------------------------------|-----------|
| `https://kiriland-metaverso.duckdns.org/archipelago/ws` | `101 Switching Protocols` |
| `https://kiriland-metaverso.duckdns.org/robotarm/ws/scene` | `400 Bad Request` (do relay: o pedido chegou sem `Upgrade`) |
| `http://localhost/robotarm/ws/scene` (nginx, sem o proxy) | `101 Switching Protocols` |
| `http://172.20.0.1:8080/ws/scene` (relay direto)       | `101 Switching Protocols` |

Requisições HTTP comuns (`GET`, `POST` e o *preflight* CORS) atravessam o
proxy normalmente.

### Solução adotada: consulta periódica por HTTP

Em vez de manter um WebSocket, a cena consulta `/healthz` e `/api/state` a
cada 0,2 s e envia comandos por `POST /api/command` (ver o
[protocolo](protocolo.md#http) e a [cena](../../scene/docs/cena.md#comunicação-relayts)).
O modo WebSocket continua disponível (`RELAY_TRANSPORT = 'websocket'` em
`scene/src/config.ts`).

### Alternativas avaliadas

| Alternativa                                   | Situação |
|-----------------------------------------------|----------|
| Liberar o *upgrade* em `/robotarm/` no proxy da universidade | Solução ideal; depende da equipe que administra o proxy. Basta replicar a configuração de `/archipelago/ws`. Depois, mude a cena para `websocket` |
| Túnel da Cloudflare (`cloudflared`)            | Inviável: a porta de saída 7844 está bloqueada pelo firewall |
| Outro túnel de terceiros (ngrok etc.)          | Possível pela porta 443, mas depende de serviço externo e a URL muda a cada execução |
| Cena falando MQTT direto com o broker          | Descartada: contorna a validação do relay e o runtime das cenas não suporta bem clientes MQTT |

## 4. Publicação da cena

O Catalyst está configurado com `IGNORE_BLOCKCHAIN_ACCESS_CHECKS=true`, então
qualquer carteira publica em qualquer parcela. A cena fica em 30,30, fora da
cena `campusFGA` (x 0–15, y 0–16). Antes de publicar em outra parcela,
verifique se ela está livre, pois a publicação substitui a cena existente:

```bash
curl -s https://kiriland-metaverso.duckdns.org/content/entities/active \
  -H 'content-type: application/json' -d '{"pointers":["30,30"]}'
```

A publicação foi assinada com chaves descartáveis, nunca gravadas no
repositório (comandos em [testes da cena](../../scene/docs/testes.md#publicação-no-catalyst-do-projeto)).

## 5. Teste ponta a ponta

Com o relay publicado e o ESP32 simulado no Wokwi
([simulação](../../robot/docs/simulacao.md)):

| Verificação                                                | Resultado |
|------------------------------------------------------------|-----------|
| `/healthz` e `/api/state` pela URL pública                 | OK        |
| `POST /api/command` pela URL pública                        | OK — `ack`; o braço simulado foi de 90° a `[30,60,150]`, garra 20, em ~6 s |
| Painel da cena no cliente web (`+`/`-`, `Home`)             | OK — cada clique chegou ao firmware (`[mqtt] command: ...`) e o braço virtual acompanhou a telemetria |
| WebSocket pela URL pública                                  | Falha, ver seção 3 |
| Relay em Docker atrás da rota `/robotarm/`                  | OK — contêiner saudável, `/healthz` e `/api/command` respondem pela URL pública |

O teste revelou um defeito na cena, já corrigido: cada comando partia da
telemetria, e um clique em outra junta interrompia o movimento em curso.

## Pendências

- **Autenticação.** Sem `SCENE_TOKEN`, qualquer pessoa com a URL move o braço;
  mesmo com token, ele fica legível no código da cena publicada. Defina um
  controle de acesso antes de ligar o braço físico.
- **Broker do braço físico.** Configurar o Mosquitto da VM para o contêiner
  (seção 1) e o `MQTT_HOST` do firmware.
- **Calibração** dos ângulos dos servos com o modelo 3D
  ([cena](../../scene/docs/cena.md#calibração)).
