# Testes e publicação

## Compilação

```bash
npm run build
```

Compila `src/` para `bin/index.js` e executa o verificador de tipos.

## Preview local com o relay

O preview serve a cena em `http://localhost:8000` (realm `LocalPreview`).
Para usar um relay local, troque `RELAY_URL` em `src/config.ts` por
`http://localhost:8080`; o ciclo completo funciona com o relay rodando na
mesma máquina que o cliente do Decentraland.

1. Broker e relay (ver [operação do relay](../../relay-server/docs/operacao.md)):
   ```bash
   cd ../relay-server
   MQTT_BROKER=tcp://localhost:1883 go run ./cmd/relay
   ```
2. Braço físico, simulação no Wokwi ou, sem nenhum dos dois, um dispositivo
   falso pelo Mosquitto:
   ```bash
   mosquitto_pub -t robotarm/status -r -m online
   mosquitto_pub -t robotarm/state -r -m '{"joints":[90,60,120],"gripper":70}'
   mosquitto_sub -t robotarm/cmd -v     # comandos enviados pelo painel
   ```
3. Preview:
   ```bash
   npm start                 # abre o Decentraland Explorer (desktop)
   npm start -- --no-client  # apenas o servidor
   ```

Se o preview e o relay estiverem em um servidor remoto, encaminhe as duas
portas e abra o Explorer, ou o cliente web
(`https://decentraland.org/bevy-web/?preview=true&realm=http://127.0.0.1:8000&position=12,13`),
na sua máquina:

```bash
ssh -L 8000:localhost:8000 -L 8080:localhost:8080 <usuario>@<servidor>
```

Verificações esperadas:

| Ação                                  | Resultado                                         |
|---------------------------------------|---------------------------------------------------|
| Entrar na cena                        | Braço em pé no centro da parcela, com a atribuição |
| Publicar um novo `state`               | O braço virtual se move suavemente até a pose      |
| Clicar `+`/`-` ou `Home` no painel     | Comando aparece em `robotarm/cmd`                  |
| Publicar `offline` em `status`         | Painel mostra `Arm: offline`; comandos recebem erro |

## Publicação no Catalyst do projeto

O Catalyst local (realm `test`, <https://kiriland.unb.br>) está
configurado com `IGNORE_BLOCKCHAIN_ACCESS_CHECKS=true`: qualquer carteira
pode publicar em qualquer parcela. A cena ocupa as parcelas da `campusFGA`
(0,0–15,16) e a publicação substitui a versão ativa do campus.

A assinatura pode ser feita com uma chave descartável, guardada fora do
repositório:

```bash
DCL_PRIVATE_KEY=0x<chave> npm run deploy -- \
  --target https://kiriland.unb.br --no-browser --skip-validations
```

Verificação:

```bash
curl -s https://kiriland.unb.br/content/entities/active \
  -H 'content-type: application/json' -d '{"pointers":["0,0"]}'
```

Para visitar o braço, abra
<https://decentraland.zone/play/?realm=https%3A%2F%2Fkiriland.unb.br&position=12%2C13>.
A cena publicada usa o relay em `https://kiriland.unb.br/robotarm`,
publicado pelo nginx do Catalyst; a configuração e o problema do proxy da
universidade com WebSocket estão em
[implantação](../../relay-server/docs/implantacao.md).

### Teste ponta a ponta com o Wokwi

Feito com a cena ainda separada, na parcela 30,30.

Validado em 20/11/2026, com o relay em `RELAY_ADDR=172.20.0.1:8080` e o
broker e o prefixo do Wokwi
([simulação](../../robot/docs/simulacao.md)):

| Verificação                                                    | Resultado |
|----------------------------------------------------------------|-----------|
| `/healthz` e `/api/state` pela URL pública                     | OK — `deviceOnline: true` com o ESP32 simulado |
| *Preflight* CORS de `POST /api/command`                         | OK        |
| Comando `[30,60,150]`, garra 20, por `POST` na URL pública      | OK — `ack`; o braço simulado chegou à pose em ~6 s e cada passo apareceu em `/api/state` |
| WebSocket pela URL pública                                      | Falha — o proxy TLS da universidade responde 400 ao *upgrade* |

O plano gratuito do Wokwi encerra a simulação pelo CLI após 5 minutos.
