# Funcionamento da cena

## Montagem do braço (`arm.ts`)

O Decentraland só transforma um `.glb` como um bloco único, por isso o braço
é dividido em seis partes (ver
[`split-model`](../../Robotik_arm/split-model/README.md)). A cena lê
`assets/models/arm/manifest.json` e cria uma entidade por parte, aninhada na
parte em que está montada:

```
raiz (ARM_POSITION, escala ARM_SCALE)
└── base
    └── turntable      joints[0]  base
        └── upper_arm  joints[1]  ombro
            └── forearm   joints[2]  cotovelo (punho rígido)
                ├── jaw_driven   gripper
                └── jaw_idler    gripper, sentido oposto
```

Cada entidade fica em `offsetFromParent` e gira em torno de `axis`. Como o
glTF é destro e o Decentraland é canhoto (os modelos importados são
espelhados em `x`), as posições do manifesto têm `x` invertido e os eixos têm
`y` e `z` invertidos; assim, os sinais dos ângulos do manifesto continuam
valendo.

A pose exibida segue a última telemetria a no máximo `FOLLOW_SPEED_DEG_S`
(180°/s), suavizando as atualizações de 10 Hz do firmware.

## Comunicação (`relay.ts`)

A cena abre um WebSocket em `RELAY_URL` (com `?token=` quando `SCENE_TOKEN`
estiver definido) e reconecta a cada `RECONNECT_DELAY_S` segundos se a
conexão cair. Mensagens tratadas, conforme o
[protocolo](../../relay-server/docs/protocolo.md#websocket-cena--relay):

| Mensagem    | Efeito na cena                                       |
|-------------|------------------------------------------------------|
| `telemetry` | Atualiza a pose-alvo do braço virtual                |
| `status`    | Mostra se o braço físico está online                 |
| `error`     | Mostra o motivo da rejeição no painel                |

## Painel (`ui.tsx`)

Mostra o estado da conexão, os ângulos atuais e botões `-`/`+` por junta e
para a garra (passo `COMMAND_STEP_DEG`, 10°), além de `Home` (todas as
juntas em 90°). Cada comando parte da última telemetria, isto é, da pose real
do braço, e é limitado a 0–180°.

## Configuração (`config.ts`)

| Constante             | Padrão                         | Descrição                                     |
|-----------------------|--------------------------------|-----------------------------------------------|
| `RELAY_URL`           | `ws://localhost:8080/ws/scene` | Endpoint do relay                              |
| `SCENE_TOKEN`         | *(vazio)*                      | Igual ao `SCENE_TOKEN` do relay                 |
| `ARM_POSITION`        | `(8, 0, 8)`                    | Centro da parcela                               |
| `ARM_SCALE`           | `8`                            | O modelo tem 17 cm; escalado fica com ~1,4 m    |
| `COMMAND_STEP_DEG`    | `10`                           | Graus por clique                                |
| `FOLLOW_SPEED_DEG_S`  | `180`                          | Velocidade de acompanhamento da telemetria      |
| `JOINT_CALIBRATION`   | `rest: 90, sign: 1`            | Conversão servo → modelo de cada junta          |
| `GRIPPER_CALIBRATION` | `rest: 90, sign: 1`            | Conversão servo → modelo da garra               |
| `GRIPPER_LIMITS`      | `-15` a `45`                   | Faixa de rotação das pinças (abaixo de −15° elas se sobrepõem) |

Uma cena publicada só pode abrir conexões `wss://`. Para usar o relay fora do
preview, publique-o atrás de um proxy com TLS e ajuste `RELAY_URL`.

## Calibração

O ângulo do modelo é `sign × (servo − rest)`, em graus, relativo à pose do
modelo original. Os valores atuais são provisórios. Para calibrar cada junta
com o braço físico:

1. Leve a junta à posição que corresponde à pose do modelo e anote o ângulo
   do servo: esse é o `rest`.
2. Aumente o ângulo do servo; se o braço virtual girar no sentido contrário
   ao físico, use `sign: -1`.
