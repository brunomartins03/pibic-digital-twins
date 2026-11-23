# Cena do Decentraland

Cena SDK7 do gêmeo digital: monta o braço robótico a partir das partes
móveis extraídas do modelo 3D ([processo](docs/modelo.md)), segue a
telemetria do [relay](../relay-server/README.md) e envia comandos pelo painel
da interface.

```bash
npm install
npm run build      # compila e verifica os tipos
npm start          # preview local em http://localhost:8000
```

## Documentação

| Documento                         | Conteúdo                                              |
|-----------------------------------|-------------------------------------------------------|
| [Modelo 3D](docs/modelo.md)       | Divisão do modelo em partes, remontagem e ligação com o relay |
| [Cena](docs/cena.md)              | Funcionamento, configuração e calibração              |
| [Testes e publicação](docs/testes.md) | Preview com o relay e publicação no Catalyst      |

## Estrutura

```
scene/
├── assets/models/arm/   # partes do braço (.glb) e manifest.json usados pela cena
├── model2.glb           # modelo original ("4-Axis Arm Assembly")
├── parts/               # saída do script de divisão (cópia de assets/models/arm/)
├── src/
│   ├── index.ts         # ponto de entrada
│   ├── config.ts        # URL do relay, posição, escala e calibração
│   ├── arm.ts           # cadeia cinemática e animação das juntas
│   ├── relay.ts         # cliente HTTP/WebSocket do relay
│   └── ui.tsx           # painel de status e comandos
├── scene.json           # parcela 30,30
└── docs/
```

O modelo é "4-Axis Arm Assembly", de samoubala,
[CC-BY-4.0](http://creativecommons.org/licenses/by/4.0/); a atribuição aparece
na cena ao lado do braço.
