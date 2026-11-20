# Testes e publicação

## Compilação

```bash
npm run build
```

Compila `src/` para `bin/index.js` e executa o verificador de tipos.

## Preview local com o relay

O preview serve a cena em `http://localhost:8000` (realm `LocalPreview`).
Nele, a cena pode abrir `ws://localhost:8080`, então o ciclo completo funciona
com o relay rodando na mesma máquina que o cliente do Decentraland.

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
portas e abra o Explorer na sua máquina:

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

O Catalyst local (realm `test`, <https://kiriland-metaverso.duckdns.org>) está
configurado com `IGNORE_BLOCKCHAIN_ACCESS_CHECKS=true`: qualquer carteira
pode publicar em qualquer parcela. A parcela 30,30 fica fora da cena
`campusFGA` (x 0–15, y 0–16); verifique antes de publicar em outra parcela,
pois a publicação substitui a cena existente.

A assinatura pode ser feita com uma chave descartável, guardada fora do
repositório:

```bash
DCL_PRIVATE_KEY=0x<chave> npm run deploy -- \
  --target https://kiriland-metaverso.duckdns.org --no-browser --skip-validations
```

Verificação:

```bash
curl -s https://kiriland-metaverso.duckdns.org/content/entities/active \
  -H 'content-type: application/json' -d '{"pointers":["30,30"]}'
```

Para visitar, conecte o Explorer ao realm
`https://kiriland-metaverso.duckdns.org` e vá até 30,30. Como o relay não está
exposto por `wss://`, a cena publicada mostra o braço na pose de repouso com o
painel em `Relay: disconnected`; o ciclo com o relay é testado no preview.
