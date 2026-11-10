# Auto Yuumi

Um assistente para partidas de Yuumi que acompanha o ADC e cuida de algumas ações repetitivas durante o jogo.

## Como usar

1. Baixe o pacote mais recente na [página de Releases](https://github.com/levyvix/auto-yuumi/releases/latest) e extraia os arquivos.
2. Abra `auto-yuumi.exe` e volte ao jogo.
3. Use **F8** para pausar ou retomar. A tecla Esc não interfere no app.
4. Ative **E automático** se quiser usar E a cada 25 segundos.

O app só envia comandos enquanto o jogo está em primeiro plano. Mantenha `config.json` e a pasta `assets` junto do executável. Configure **W em conjuração rápida** e **B para recall** no jogo.

## O que o app faz

- Usa W no ADC identificado pelo quarto retrato acima do minimapa.
- Quando o ADC morre, clica na loja verde aliada, espera 10 segundos, usa recall e aguarda o ADC renascer.
- Reavalia o ADC durante o retorno; se ele estiver vivo, cancela o movimento ou recall e retoma o W.
- Pausa W e E durante a rotina de retorno.
- Ao morrer ou detectar chegada à base, abre a loja com **X**, tenta comprar o item recomendado da direita com o botão direito e fecha a loja. Também tenta comprar quando os Recomendados são abertos manualmente.
- Salva registros na pasta `logs`.
- Evolui habilidades com **Ctrl + tecla**, na prioridade **R > Q > E > W**, quando a seta de evolução está disponível.
- Enquanto espera o ADC renascer, alterna entre dois pontos da fonte a cada 20 segundos quando detecta Guarda Territorial. A opção `walk_in_base` desativa os movimentos.

## Limitações

O app foi calibrado para **1920 × 1080** e para a configuração de interface usada durante o desenvolvimento. Outras resoluções ou escalas podem exigir ajustes em `config.json` e nas imagens da pasta `assets`. O app não detecta o alcance do W nem confirma se o recall foi interrompido.

A compra requer que a loja esteja no mesmo local e que a aba **Recomendados** esteja visível. É feita uma tentativa por abertura, sem repetir até gastar todo o ouro. O jogo pode recusar a compra se faltar ouro, espaço ou acesso à loja. Configure **X** para abrir e fechar a loja; `auto_buy` em `config.json` ativa ou desativa essa rotina.

A morte é identificada pelos dados locais do jogo. Sem dados válidos, não há tentativa de compra por morte ou chegada à base. A chegada à base é identificada pelo ícone de Guarda Territorial, que pode permanecer visível brevemente fora da fonte. Cada nova morte ou visita gera uma tentativa; permanecer no mesmo estado não repete a rotina.

Automação pode infringir as regras da Riot e resultar em suspensão da conta. Use por sua conta e risco.
