# split-model

Divide o `model2.glb` ("4-Axis Arm Assembly") em um `.glb` por parte móvel,
cada um com a origem no pivô da sua junta. O Decentraland só transforma um
modelo glTF como um bloco único, então a cena precisa recriar a cadeia
cinemática com entidades aninhadas (`Transform.parent`), uma por parte.

```bash
npm install
npm run split                                  # ../model2.glb -> ../parts/
npm run split -- --max-triangles 8000          # outro orçamento de triângulos
```

| Opção             | Padrão          | Descrição                                     |
|-------------------|-----------------|-----------------------------------------------|
| `--input`         | `../model2.glb` | Modelo de origem                               |
| `--out`           | `../parts`      | Pasta de saída                                 |
| `--max-triangles` | `10000`         | Total máximo de triângulos (10 mil = 1 parcela) |

## Partes geradas

| Arquivo          | Pai         | Campo do relay | Eixo     | Conteúdo                                         |
|------------------|-------------|----------------|----------|--------------------------------------------------|
| `base.glb`       | —           | —              | —        | Caixa fixa, rolamentos e eixos                    |
| `turntable.glb`  | `base`      | `joints[0]`    | `+Y`     | Prato giratório, suporte e servo do ombro          |
| `upper_arm.glb`  | `turntable` | `joints[1]`    | `-X`     | Braço e servo do cotovelo                          |
| `forearm.glb`    | `upper_arm` | `joints[2]`    | `-X`     | Antebraço, punho (rígido), corpo da garra e servo da garra |
| `jaw_driven.glb` | `forearm`   | `gripper`      | `+Y`     | Engrenagem/garra acoplada ao servo                 |
| `jaw_idler.glb`  | `forearm`   | `gripper` (espelhado) | `+Y` | Engrenagem/garra movida pela outra              |

O punho do modelo não tem servo correspondente no braço físico e por isso
foi mantido rígido dentro de `forearm`.

## Sistema de coordenadas

- Metros; `y` para cima; a base fica centrada na origem, apoiada em `y = 0`.
- O braço aponta para `+z`; ombro e cotovelo giram em torno de `x`.
- A pose de repouso é a do modelo original (altura total de 17,2 cm).

## `manifest.json`

Para cada parte: `file`, `parent`, `relayField`, `axis` (vetor unitário),
`pivot` (posição absoluta na pose de repouso) e `offsetFromParent` (posição
local da entidade em relação ao pai). Montagem na cena:

```
entidade da parte:
  parent   = entidade do pai
  position = offsetFromParent
  rotation = rotação de (ângulo - ângulo de repouso) em torno de axis
  GltfContainer = file
```

Os ângulos são relativos à pose do modelo (regra da mão direita). A relação
entre o ângulo do servo (0–180°) e o ângulo do modelo ainda precisa ser
calibrada com o braço físico. Na garra, `jaw_driven` com `+θ` e `jaw_idler`
com `−θ` abrem as pinças; fechar além de cerca de −15° faz as pinças se
sobreporem.

## Simplificação

Os 20.374 triângulos originais excedem o orçamento de uma parcela do
Decentraland (10 mil). O script solda vértices, remove duplicatas e aplica o
simplificador do meshoptimizer com a mesma razão em todas as partes até caber
no limite (resultado atual: 9.498 triângulos).

## Licença do modelo

"4-Axis Arm Assembly", de samoubala, licenciado sob
[CC-BY-4.0](http://creativecommons.org/licenses/by/4.0/)
([fonte](https://sketchfab.com/3d-models/4-axis-arm-assembly-c0a748faf37644288c1271e9a6f51057)).
A atribuição deve constar na cena e no relatório.
