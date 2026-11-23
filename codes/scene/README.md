# Cena do Decentraland

Cena SDK7 do campus FGA (`campusFGA`, parcelas 0,0–15,16) com o gêmeo
digital do braço robótico. O campus vem da cena feita no Creator Hub
([origem e mudanças](docs/cena.md#campus-fga)); o braço é montado a partir
das partes móveis extraídas do modelo 3D ([processo](docs/modelo.md)), segue
a telemetria do [relay](../relay-server/README.md) e recebe comandos pelo
painel da interface.

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
├── assets/scene/        # main.composite: o campus montado no Creator Hub
├── assets/Models/       # modelos do campus (prédio da FGA, símbolo e texto da UnB)
├── assets/asset-packs/  # itens do Creator Hub (bancos, botões...) e seus scripts
├── assets/models/arm/   # partes do braço (.glb) e manifest.json usados pela cena
├── model2.glb           # modelo original ("4-Axis Arm Assembly")
├── parts/               # saída do script de divisão (cópia de assets/models/arm/)
├── split-model/         # script que divide model2.glb em partes (glTF-Transform)
├── src/
│   ├── index.ts         # ponto de entrada
│   ├── config.ts        # URL do relay, posição, escala e calibração
│   ├── arm.ts           # cadeia cinemática e animação das juntas
│   ├── relay.ts         # cliente HTTP/WebSocket do relay
│   └── ui.tsx           # painel de status e comandos
├── scene.json           # parcelas 0,0–15,16 (272), ponto de chegada e permissões
└── docs/
```

O modelo é "4-Axis Arm Assembly", de samoubala,
[CC-BY-4.0](http://creativecommons.org/licenses/by/4.0/); a atribuição aparece
na cena ao lado do braço.
