# Firmware do braço robótico

Firmware Arduino para o ESP8266 (NodeMCU/Wemos D1 mini) que aciona os quatro
servos SG90 do braço (base, ombro, cotovelo e garra) e o mantém sincronizado
com o gêmeo digital por MQTT, via [relay server](../relay-server/README.md).
O mesmo código compila para ESP32, usado na simulação do Wokwi.

```bash
cp config.example.h config.h   # Wi-Fi, broker e pinos
make build                     # compila para ESP8266
make upload PORT=/dev/ttyUSB0  # grava na placa
make monitor                   # Serial a 115200 baud (teclas Q/A W/S E/D R/F)
make sim                       # compila para ESP32 e simula no Wokwi
```

## Documentação

| Documento                         | Conteúdo                                                   |
|-----------------------------------|------------------------------------------------------------|
| [Hardware](docs/hardware.md)      | Componentes, ligações, pinos e alimentação                  |
| [Firmware](docs/firmware.md)      | Ambiente, compilação, configuração e funcionamento interno  |
| [Simulação](docs/simulacao.md)    | Simulação no Wokwi e resultados da análise do código        |

## Estrutura

```
robot/
├── robot.ino          # setup/loop: Wi-Fi, controle pela Serial e publicação do estado
├── ArmController.*    # servos, movimento suave e teclas de controle local
├── MqttLink.*         # conexão MQTT: estado, comandos e status (last will)
├── config.example.h   # modelo do config.h (não versionado) para a placa real
├── config.wokwi.h     # configuração usada na simulação (-DWOKWI_SIM)
├── simulation/        # projeto Wokwi (diagram.json, wokwi.toml)
├── Makefile
└── docs/
```
