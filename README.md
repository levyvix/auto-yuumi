# Auto Yuumi 1.0

Aplicativo para Windows, feito em Go, para acompanhar o ADC com a Yuumi.

## Como usar

1. Baixe e extraia o ZIP da versao 1.0.
2. Abra `auto-yuumi.exe` e volte ao jogo.
3. Use **F8** para pausar ou retomar. Esc nao interfere no app.
4. Marque a opcao de **E automatico** se quiser usar E a cada 25 segundos.

O app so envia comandos com o jogo em primeiro plano. Mantenha `config.json` e a pasta `assets` junto do executavel. Configure **W em conjuracao rapida** e **B para recall**.

## O que faz

- Usa W no ADC pelo quarto retrato acima do minimapa.
- Quando o ADC morre, clica na loja verde aliada, espera 10 segundos, usa recall e aguarda o ADC renascer.
- Reavalia o ADC durante todo o retorno: se ele estiver vivo, cancela o movimento ou recall e retoma o W.
- Bloqueia W e E durante a rotina de retorno.
- Salva os registros na pasta `logs`.

## Limitacoes

Calibrado para **1920 x 1080**, com a interface usada no desenvolvimento. Mudancas na escala da interface exigem ajustar `config.json` e os recortes em `assets`. Nao detecta alcance do W nem confirma se o recall foi interrompido. Automacao pode infringir as regras da Riot e causar suspensao.

## Compilar

O CI segue o modelo do `league-auto-accept`: testes, `go vet`, build Windows e artefatos. O Release Please prepara as proximas versoes a partir dos commits e publica o executavel e o ZIP.

Requer Windows 64 bits e Go 1.26 ou superior.

```powershell
go test ./...
go build -ldflags '-H=windowsgui' -o auto-yuumi.exe .
```
