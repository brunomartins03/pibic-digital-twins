# Ambiente completo em Docker

Sobe, em qualquer máquina com Docker, o mesmo ambiente do servidor do
laboratório: um Catalyst do Decentraland (realm `test`) com a cena
`campusFGA` (parcelas 0,0–15,16) e a cena do gêmeo digital na parcela 30,30,
o relay do braço e, opcionalmente, os serviços de comunicação entre avatares.

Reconstruído em 23/11/2026 a partir dos contêineres em execução no servidor
(`/opt/catalyst/catalyst-owner`, `/opt/decentraland-comms` e
`codes/relay-server`). As imagens estão fixadas nos *digests* que rodavam lá,
e as de comunicação são compiladas dos mesmos *commits* do GitHub.

```
navegador ──HTTP(S)──▶ nginx :80
                        ├── /content, /lambdas, /about ─▶ content-server, lamb2 ─▶ postgres
                        ├── /robotarm/                 ─▶ relay ──MQTT──▶ broker ──▶ braço
                        ├── /archipelago/ws            ─▶ ws-connector ─┐
                        └── /rtc                       ─▶ livekit       │ NATS ◀─ pulse
                                                         gatekeeper ◀──┘ (perfil "comms")
deployer (executa uma vez): compila codes/scene e publica as duas cenas
```

| Serviço | Origem | Papel |
|---|---|---|
| `postgres`, `content-server`, `lamb2`, `nginx` | imagens do `catalyst-owner` | Catalyst: conteúdo, *lambdas* e `/about` do realm |
| `relay` | `../relay-server` | Ponte HTTP/WebSocket ⇄ MQTT do braço |
| `deployer` | `deployer/` | Compila a cena do braço e publica as cenas; termina em seguida |
| `nats`, `ws-connector`, `pulse`, `gatekeeper`, `postgres-gatekeeper`, `livekit` | GitHub `decentraland/*` | Comunicação entre avatares (perfil `comms`) |

## Requisitos

- Docker Engine com Compose 2.17 ou mais recente (`additional_contexts`).
- Acesso à internet: o Catalyst consulta os Catalysts públicos na primeira
  inicialização, e as imagens de comunicação são compiladas do GitHub.
- Cerca de 3 GB de disco para o núcleo e mais 3 GB para o perfil `comms`.

## Uso

```bash
cd codes/deploy
cp .env.example .env
# preencha POSTGRES_PASSWORD, POSTGRES_CONTENT_PASSWORD e MQTT_PASSWORD
docker compose up -d --build
docker compose logs -f deployer
```

Na primeira inicialização o Catalyst passa cerca de 6 minutos em
`Bootstrapping`; o `deployer` espera até o estado `Syncing` e então publica as
cenas:

```
waiting for the Catalyst (Bootstrapping)
Robotic arm digital twin: deployed bafkrei… to 1 parcel(s) from 30,30
campusFGA: deployed bafkrei… to 272 parcel(s) from 0,0
```

Verificação:

```bash
curl -s localhost/about | grep -o '"healthy":[a-z]*' | head -1   # "healthy":true
curl -s localhost/content/entities/active -H 'content-type: application/json' \
  -d '{"pointers":["30,30"]}' | grep -o '"title":"[^"]*"'         # Robotic arm digital twin
curl -s localhost/robotarm/healthz                                # {"deviceOnline":…,"ok":true}
```

Cliente web, com `<PUBLIC_URL>` codificado na URL:
`https://decentraland.zone/play/?realm=<PUBLIC_URL>&position=30%2C30`.

A cena do braço recebe na compilação a URL do relay (`${PUBLIC_URL}/robotarm`)
e o `SCENE_TOKEN`, que substituem os valores de `scene/src/config.ts`. Uma
cena publicada só abre URLs `https://` e `wss://`; por isso, com
`PUBLIC_URL=http://localhost`, o braço pode não receber telemetria no cliente.
O acesso pelo navegador não foi testado nesse modo.

### Servidor público

O cliente web exige um realm `https://`. Como no servidor do laboratório, o
TLS fica num proxy à frente do nginx, que repassa HTTP para `HTTP_PORT`. O
proxy precisa repassar o *upgrade* de WebSocket em `/archipelago/ws` e `/rtc`
e, para a cena usar `RELAY_TRANSPORT = 'websocket'`, também em `/robotarm/`.
No `.env`:

```bash
PUBLIC_URL=https://exemplo.org
LIVEKIT_PUBLIC_URL=wss://exemplo.org
LIVEKIT_NODE_IP=<IP público>      # liberar 3050/udp e 3060/tcp
```

### Comunicação entre avatares (perfil `comms`)

Descomente `COMPOSE_PROFILES=comms` e `ARCHIPELAGO_URL=${PUBLIC_URL}` no
`.env`, preencha as chaves do LiveKit e as senhas do *gatekeeper*
(`openssl rand -hex 32`) e rode `docker compose up -d --build`. A primeira
compilação leva vários minutos (a do Pulse baixa o SDK do .NET). Com o perfil
ativo, o `/about` anuncia
`archipelago:archipelago:<ws|wss>://…/archipelago/ws`.

Sem o perfil, deixe `ARCHIPELAGO_URL` vazio: definido e sem `ws-connector`, o
`/about` responde 503 e o cliente recusa o realm.

## Atualizar as cenas

Depois de alterar `codes/scene`:

```bash
docker compose up -d --build deployer
```

O `deployer` compara os arquivos e o `scene.json` com a entidade ativa em cada
parcela e só publica o que mudou. A assinatura usa uma carteira descartável
gerada a cada execução: o Catalyst roda com
`IGNORE_BLOCKCHAIN_ACCESS_CHECKS=true` e aceita qualquer endereço.

## Cópia da campusFGA

`scenes/campusFGA/` é uma cópia exata da entidade publicada no servidor em
21/11/2026 (`bafkreih4lznwhjnoksytqdsmmde4hob6b42rjtrisifpo6bx7f4zyupsti`).
Nenhuma cópia do código-fonte corresponde a ela: o `main.crdt` publicado
difere de todas as versões de `Thiago/CenasProntas/campusFGA` em
`pspd-unb/metaverso-decentraland`. Os metadados originais (com
`sdkVersion: 7.30.1`) estão em `.entity-metadata.json`. Ficou de fora apenas
`Claude outputs/Relatorio_TCC_Navegacao_Facilitada.docx`, um relatório
publicado por engano junto com a cena.

Para refazer a cópia a partir de um Catalyst:

```bash
python3 scripts/export-scene.py https://kiriland.unb.br/content 0,0 \
  scenes/campusFGA --exclude 'Claude outputs/'
```

## Diferenças em relação ao servidor do laboratório

| Servidor do laboratório | Este ambiente |
|---|---|
| Três projetos Compose (`catalyst-owner`, `dcl-comms-test`, `dcl-gatekeeper`/`dcl-livekit`) ligados por uma rede externa | Um projeto, uma rede |
| `CATALYST_URL=https://kiriland.unb.br`, acessado também por `kiriland-metaverso.duckdns.org` | Uma única `PUBLIC_URL` |
| Relay publicado em `172.20.0.1:8080` e alcançado pelo gateway da rede | Alcançado pelo nome `relay`, sem porta no host |
| Monitoramento (cAdvisor, *exporters*, rotas `/*_metrics`) e Certbot | Removidos |
| Postgres ajustado para o servidor (`shared_buffers` 2,5 GB) | Configuração padrão |
| Logs em `syslog` | Logs do Docker (`json-file`, 10 MB × 3) |
| Portas do Pulse, ws-connector, NATS e *gatekeeper* em `127.0.0.1` | Sem portas no host (só pelo nginx) |
| Cena do braço ausente em 30,30 desde a republicação da campusFGA | Publicada pelo `deployer` |
