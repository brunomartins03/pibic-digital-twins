# Do modelo 3D ao gêmeo digital na cena

Este documento descreve o processo usado para transformar o modelo 3D do braço
em um gêmeo digital animado no Decentraland: a divisão do modelo em partes
móveis, a remontagem na cena e a ligação das juntas com o relay. Realizado entre
20 e 21/11/2026.

```
model2.glb ──split-model──▶ parts/*.glb + manifest.json ──▶ assets/models/arm/
(1 bloco, 20.374 triângulos)  (6 partes, 9.498 triângulos)      │
                                                                ▼
                       cena (arm.ts): uma entidade por parte, aninhadas
                                                                │ ângulos
relay ──telemetria (HTTP/WebSocket)──▶ relay.ts ──▶ rotação de cada junta
      ◀──────── comandos do painel (ui.tsx) ────────
```

## 1. Por que dividir o modelo

O Decentraland (SDK7) só move um `.glb` como um bloco único:

| Recurso do SDK7         | O que permite                                       | Por que não serve para as juntas |
|-------------------------|-----------------------------------------------------|----------------------------------|
| `Transform` + `GltfContainer` | Posição, rotação e escala do modelo inteiro    | Não alcança os nós internos      |
| `GltfNodeModifiers`     | Trocar material e sombra de nós internos            | Não altera a transformação       |
| `Animator`              | Tocar animações embutidas (`playing`, `loop`, `speed`, `weight`) | Não posiciona a animação em um instante arbitrário, então não reproduz um ângulo vindo da telemetria |

A solução é exportar **um `.glb` por parte móvel**, com a origem no eixo da
junta, e recriar a cadeia cinemática na cena com entidades aninhadas
(`Transform.parent`). Girar uma entidade gira junto tudo o que está montado
nela.

Limites que orientaram o processo: 10.000 triângulos por parcela e
`log2(n+1) × 20` materiais para `n` parcelas. A cena ocupa uma parcela.

## 2. Escolha do modelo

Foram avaliados dois modelos:

| Modelo | Situação |
|--------|----------|
| `Robotik_arm.glb` (exportado do Blender; hoje usado na `campusFGA`) | Primeiro candidato; as juntas foram identificadas, mas ele foi substituído |
| `model2.glb` — "4-Axis Arm Assembly", de samoubala ([Sketchfab](https://sketchfab.com/3d-models/4-axis-arm-assembly-c0a748faf37644288c1271e9a6f51057), [CC-BY-4.0](http://creativecommons.org/licenses/by/4.0/)) | Escolhido: cada peça é uma malha separada (`Object_N`), o que permite agrupá-las por junta. Exige redução de triângulos (20.374 > 10.000) e atribuição do autor |

O `model2.glb` original fica em `scene/model2.glb`. A atribuição aparece na
cena, ao lado do braço, e no `manifest.json`.

## 3. Identificação das partes

Cada `Object_N` do modelo foi atribuído a uma das seis partes rígidas. Os
pivôs foram medidos nos eixos de saída dos servos. Ombro, cotovelo e punho
giram em torno do mesmo eixo horizontal, `[0,659, 0, −0,752]` nas coordenadas
do modelo.

| Parte        | Pai          | Campo do relay | Malhas | Conteúdo | Pivô |
|--------------|--------------|----------------|--------|----------|------|
| `base`       | —            | —              | `Object_30`, `15`, `24`, `39`, `48`, `21`, `51`, `57`, `60` | Caixa fixa, rolamentos e eixos | Centro da base, no chão |
| `turntable`  | `base`       | `joints[0]`    | `Object_54`, `6`, `9` | Prato giratório, suporte e servo do ombro | Centro do prato, na face inferior |
| `upper_arm`  | `turntable`  | `joints[1]`    | `Object_63`, `18` | Braço e servo do cotovelo | Eixo do servo do ombro |
| `forearm`    | `upper_arm`  | `joints[2]`    | `Object_12`, `33`, `36`, `42`, `27` | Antebraço, punho, corpo da garra e servo da garra | Eixo do servo do cotovelo |
| `jaw_driven` | `forearm`    | `gripper`      | `Object_3` | Engrenagem acoplada ao servo da garra | Eixo do servo da garra |
| `jaw_idler`  | `forearm`    | `gripper` (espelhado) | `Object_45` | Engrenagem movida pela outra | Centro do pino `Object_42` |

O modelo tem quatro eixos, mas o braço físico tem três juntas e uma garra. Por
isso, o **punho foi mantido rígido** dentro de `forearm`.

## 4. Divisão (`split-model`)

A divisão é feita por código, com
[glTF-Transform](https://gltf-transform.dev) 4.5 e o simplificador do
[meshoptimizer](https://meshoptimizer.org), sem passar pelo Blender. O script
`split-model.mjs`:

1. **Define um referencial canônico:** base centrada na origem e apoiada em
   `y = 0`, prato giratório no eixo `y` e braço apontando para `+z`. O giro em
   torno de `y` é calculado a partir do eixo comum dos servos, para que ombro e
   cotovelo girem exatamente em torno de `x`. O punho só define o sentido.
2. **Extrai cada parte** em um documento próprio: as malhas recebem a matriz
   `T(−pivô) × referencial × matriz global`, de modo que a origem do arquivo
   fique no pivô e a posição relativa entre as peças seja preservada.
3. **Valida a atribuição:** cada `Object_N` deve pertencer a exatamente uma
   parte; caso contrário, o script falha.
4. **Reduz os triângulos:** solda vértices e remove duplicatas, depois
   simplifica todas as partes com a mesma razão, sempre a partir das malhas
   originais, até o total caber no orçamento (padrão 10.000). Resultado:
   20.374 → 9.498 triângulos, razão 0,467.
5. **Grava** `<parte>.glb` e um `manifest.json` com, para cada parte, `file`,
   `parent`, `relayField`, `axis` (vetor unitário), `pivot` (posição absoluta
   na pose de repouso), `offsetFromParent` e `triangles`.

| Parte        | Eixo     | `offsetFromParent` (m)       | Triângulos |
|--------------|----------|------------------------------|-----------:|
| `base`       | —        | `[0, 0, 0]`                  | 1.568 |
| `turntable`  | `+Y`     | `[0, 0,038, 0]`              | 708 |
| `upper_arm`  | `−X`     | `[−0,0197, 0,0346, 0,0039]`  | 640 |
| `forearm`    | `−X`     | `[0,0136, 0,0564, 0,0458]`   | 1.878 |
| `jaw_driven` | `+Y`     | `[−0,0171, 0,0425, 0,1293]`  | 3.988 |
| `jaw_idler`  | `+Y`     | `[−0,0301, 0,0115, 0,1303]`  | 716 |

A `jaw_driven` concentra 42% dos triângulos por causa dos dentes da
engrenagem.

### Reproduzir a divisão

O script não está mais na árvore de trabalho (removido em `aae0fe3`), mas
continua no histórico, no commit `269ac47`. Para regenerar as partes:

```bash
git archive 269ac47 codes/Robotik_arm/split-model | tar -x -C /tmp
cd /tmp/codes/Robotik_arm/split-model
npm ci
node split-model.mjs --input <repo>/codes/scene/model2.glb \
  --out <repo>/codes/scene/assets/models/arm --max-triangles 10000
```

Verificado em 23/11/2026: o resultado é idêntico, byte a byte, aos arquivos de
`scene/assets/models/arm/` e de `scene/parts/`.

### Verificação da divisão

As partes foram remontadas fora do Decentraland, a partir apenas do
`manifest.json`, e renderizadas:

| Verificação | Resultado |
|-------------|-----------|
| Pose de repouso | Igual ao original: 17,2 cm de altura, apoiada em `y = 0`, alcance de 22,9 cm em `+z` |
| Pose de teste (base 30°, ombro −25°, cotovelo 30°, pinças ±25°) | Os elos continuam unidos nos pinos |
| Garra | Cada engrenagem gira em torno do próprio centro; `jaw_driven` `+θ` com `jaw_idler` `−θ` abre as pinças de forma simétrica, e elas se sobrepõem abaixo de cerca de −15° |

## 5. Remontagem na cena (`arm.ts`)

As partes ficam em `scene/assets/models/arm/` (`scene/parts/` guarda uma cópia
idêntica da saída do script). A cena importa o `manifest.json` e cria uma
entidade por parte:

```
raiz (ARM_POSITION = centro da parcela, ARM_SCALE = 8 → ~1,4 m)
└── base
    └── turntable      joints[0]
        └── upper_arm  joints[1]
            └── forearm   joints[2]
                ├── jaw_driven   gripper
                └── jaw_idler    gripper, sentido oposto
```

Cada entidade recebe `parent`, `position = offsetFromParent` e o
`GltfContainer` do arquivo da parte. Como a origem de cada arquivo está no
pivô, girar a entidade em torno de `axis` move a junta corretamente.

**Conversão de sistema de coordenadas.** O glTF é destro e o Decentraland é
canhoto: os modelos importados são espelhados em `x`. Para que o manifesto
continue valendo, as posições têm `x` invertido e os eixos de rotação têm `y`
e `z` invertidos. Com isso, os ângulos do manifesto mantêm o sinal (por
exemplo, o eixo de arfagem `−X` continua `−X`, e o eixo vertical vira `−Y`).

## 6. Ligação com o relay (`relay.ts`, `arm.ts`, `ui.tsx`)

**Telemetria → juntas.** O relay entrega o estado do braço físico
(`{"joints":[base, ombro, cotovelo], "gripper": g}`, em graus de servo, 0–180).
A cada quadro, a cena:

1. aproxima a pose exibida da última telemetria a no máximo 180°/s
   (`FOLLOW_SPEED_DEG_S`), suavizando as atualizações de 10 Hz;
2. converte cada ângulo de servo em ângulo do modelo:
   `sign × (servo − rest)` (`JOINT_CALIBRATION`, `GRIPPER_CALIBRATION`);
3. na garra, limita o ângulo a `GRIPPER_LIMITS` (−15° a 45°) e aplica `+θ` em
   `jaw_driven` e `−θ` em `jaw_idler`;
4. define `rotation = Quaternion.fromAngleAxis(ângulo, eixo)` em cada junta.

**Painel → comandos.** Os botões `−`/`+` e `Home` enviam o estado completo
desejado. Por `TARGET_HOLD_S` (10 s) após um comando, os seguintes partem do
último alvo enviado, e não da telemetria.

**Transporte.** A cena fala com o relay em `RELAY_URL`. No servidor do projeto,
o proxy TLS da universidade não repassa WebSocket; por isso o padrão é o modo
`http`: consulta a `/healthz` e `/api/state` a cada 0,2 s e envio de comandos
por `POST /api/command`. O modo `websocket` continua disponível. Detalhes em
[funcionamento da cena](cena.md#comunicação-relayts) e em
[implantação](../../relay-server/docs/implantacao.md).

## 7. Problemas encontrados e soluções

| Problema | Solução |
|----------|---------|
| O SDK7 não gira nós internos de um `.glb` | Um `.glb` por parte e entidades aninhadas |
| 20.374 triângulos, acima do limite de uma parcela | Simplificação com a mesma razão em todas as partes (9.498) |
| Primeira divisão com o eixo de arfagem 2,3° fora de `x` (o giro vinha da direção do punho) | Giro calculado pela perpendicular ao eixo comum dos servos |
| Simplificação aplicada sobre malhas já simplificadas a cada iteração | Cada tentativa parte de uma cópia das malhas originais soldadas |
| Diferença de lateralidade entre glTF e Decentraland | Conversão de posições e eixos ao ler o manifesto |
| Punho do modelo sem servo no braço físico | Mantido rígido em `forearm` |
| WebSocket bloqueado pelo proxy da universidade | Modo HTTP com consulta periódica e `POST /api/command` |
| Clicar em outra junta interrompia o movimento em curso | Comandos partem do último alvo por `TARGET_HOLD_S` |
| Cliente sem interface gráfica para testar a cena no servidor (a compilação nativa do Hammurabi não foi permitida) | Testes pelo cliente web e pelas APIs do Catalyst e do relay |

## 8. Pendências

- **Calibração:** os valores de `rest` (90°) e `sign` (1) são provisórios;
  ajustar com o braço físico ([calibração](cena.md#calibração)).
- **Qualidade visual:** conferir a simplificação, principalmente nos dentes
  das engrenagens.
