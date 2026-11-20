# Hardware

## Componentes

| Item                       | Qtd. | Observação                                    |
|----------------------------|------|-----------------------------------------------|
| ESP8266 NodeMCU ou Wemos D1 mini | 1 | Placa alvo do firmware                       |
| Servomotor SG90            | 4    | Base, ombro, cotovelo e garra                  |
| Fonte 5 V, ≥ 2 A           | 1    | Alimentação exclusiva dos servos               |
| Estrutura do braço         | 1    | Kit acrílico/MDF para 4 servos SG90            |

## Ligações

| Servo     | Sinal (fio laranja) | Pino da placa | GPIO |
|-----------|---------------------|---------------|------|
| Base      | PWM                 | D1            | 5    |
| Ombro     | PWM                 | D2            | 4    |
| Cotovelo  | PWM                 | D5            | 14   |
| Garra     | PWM                 | D6            | 12   |

- Fio **vermelho** dos servos: +5 V da fonte externa.
- Fio **marrom/preto** dos servos: GND da fonte **e** GND do ESP8266 (terra
  comum, obrigatório para o sinal PWM ter referência).

Os pinos evitam os GPIOs de *boot* do ESP8266 (0, 2 e 15), cujo nível no
reset decide o modo de inicialização, e o GPIO 16, que não gera PWM. Para usar
outros pinos, altere `PIN_*` no `config.h`.

> A versão anterior do firmware usava os GPIOs 13, 14, 27 e 26, herdados de um
> ESP32. Os GPIOs 26 e 27 **não existem** no ESP8266, então cotovelo e garra
> nunca se moveriam.

## Alimentação

Quatro SG90 podem consumir mais de 1 A em conjunto quando partem ou ficam
travados. Não os alimente pelo regulador de 3,3 V nem pelo pino 5 V/VIN da
placa ligado ao USB: quedas de tensão reiniciam o ESP8266. Use uma fonte de
5 V separada e una os terras.

## Ângulos

Todos os servos usam 0–180°, com pulsos de 500–2400 µs. A posição inicial
(*home*) é 90° em todas as juntas. Se algum servo vibrar ou forçar nos
extremos, reduza a faixa em `PULSE_MIN_US`/`PULSE_MAX_US`
(`ArmController.cpp`) ou limite os ângulos usados pela cena.

**Não mova o braço com a mão enquanto estiver energizado.** Os servos mantêm
a posição ativamente; forçá-los pode danificar as engrenagens e, como não há
sensor de posição, o gêmeo digital não percebe esse movimento.
