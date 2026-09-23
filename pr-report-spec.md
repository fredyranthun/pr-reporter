# pr-report — Especificação do MVP

Versão: 1.2 • Data: 22/09/2026 • Linguagem: Go

## 1. Objetivo

Construir uma aplicação de linha de comando que receba uma lista de repositórios do GitHub e produza um relatório consolidado dos PRs abertos, incluindo repositório, número, título, comentários, discussões de código pendentes e estado do merge.

O usuário deve conseguir identificar rapidamente quais PRs precisam de atenção, sem abrir cada repositório no navegador. A ferramenta será local, somente de leitura, sem banco de dados ou serviço em execução.

Este documento define o comportamento esperado; não representa uma implementação já existente.

A versão 1.2 resolve T01 no [contrato de casos de borda](docs/contract-edge-cases.md), parte normativa desta especificação. As decisões detalham presença/nulidade, merge, duplicatas, observações e completude sem alterar o schema JSON 1.

## 2. Decisões de escopo

| Aspecto | Decisão para o MVP |
|---|---|
| Executável | `pr-report` |
| Pacote do comando | `cmd/pr-report` (`package main`) |
| Módulo Go | `github.com/fredyranthun/pr-reporter` |
| Implementação | Go, preferencialmente biblioteca padrão |
| Integração | Executar `gh api graphql` como subprocesso |
| Autenticação | Reutilizar a autenticação do GitHub CLI |
| Entrada | Arquivo TXT e/ou flags repetidas `--repo` |
| Host | Apenas `github.com` |
| Repositórios | Públicos e privados acessíveis pela identidade autenticada |
| PRs | Abertos, incluindo rascunhos |
| Saída | Tabela e JSON versionado |
| Paginação | Explícita, controlada pela aplicação |
| Concorrência | Sequencial por padrão; limite global configurável |
| Plataforma inicial | Linux; manter código portável |

Ficam fora do MVP: tratamento específico de rate limits, alterações em PRs, merge automático, notificações, agendamento, interface web/TUI, armazenamento histórico, clonagem de repositórios, GitHub Enterprise Server, avaliação completa de rulesets, merge queues e permissões de bypass. Não serão coletados corpos dos comentários ou reviews. O tratamento de rate limits fica para uma etapa posterior, descrita na seção 13.

## 3. Uso e contrato da CLI

Pré-requisito: `gh` instalado no PATH e autenticado, por exemplo com `gh auth login`. A ferramenta não deve iniciar login interativo automaticamente nem gerenciar tokens próprios.

```bash
# Consultar uma lista
pr-report --repos repos.txt

# Consultar repositórios diretamente
pr-report --repo minha-org/backend --repo minha-org/frontend

# Gerar JSON para outra ferramenta
pr-report --repos repos.txt --format json > prs.json

# Mostrar apenas PRs com discussões de código pendentes
pr-report --repos repos.txt --only-unresolved

# Consultar até três requisições simultaneamente
pr-report --repos repos.txt --concurrency 3 --timeout 45s
```

| Flag | Padrão | Comportamento |
|---|---|---|
| `--repos <arquivo>` | Ausente | Lê um arquivo de repositórios |
| `--repo <owner/name>` | Ausente | Pode ser repetida; combina com o arquivo |
| `--format table\|json` | `table` | Define a saída |
| `--fields <nomes>` | ausente | Seleciona campos dos PRs na tabela ou JSON, em ordem |
| `--only-unresolved` | `false` | Filtra por discussões pendentes confirmadas |
| `--concurrency <n>` | `1` | Inteiro entre 1 e 8; limite global de subprocessos `gh` |
| `--timeout <duração>` | `30s` | Limite por tentativa de requisição; duração positiva |
| `--help` | — | Exibe ajuda e encerra sem consultar a API |
| `--version` | — | Exibe versão e encerra sem consultar a API |

Flags devem anteceder qualquer argumento posicional; argumentos posicionais não são aceitos. Ao menos uma fonte de repositórios é obrigatória. Entradas e flags inválidas interrompem a execução antes de qualquer chamada de rede.

### Arquivo de entrada

```text
# Serviços da equipe
minha-org/backend
minha-org/frontend

minha-org/infra
```

Regras propostas:

- Aceitar UTF-8, LF e CRLF; remover BOM inicial e espaços nas extremidades.
- Ignorar linhas vazias e linhas cujo primeiro caractere após trim seja `#`.
- Aceitar somente dois segmentos não vazios separados por `/`. Owner aceita letras ASCII, números e hífen; nome aceita também ponto e underscore. Rejeitar segmentos `.` e `..`.
- URLs, caminhos locais, sufixo `.git` e comentários ao final da linha não são suportados no MVP. Informar linha e valor inválidos.
- Deduplicar sem diferenciar maiúsculas de minúsculas. Usar o nome canônico retornado pela API na saída.
- Uma lista efetivamente vazia é erro de entrada.

## 4. Dados por PR

Os nomes abaixo são parte do contrato JSON. Campos indisponíveis devem ser `null`, nunca preenchidos com zero ou falso por conveniência.

| Campo | Tipo | Uso no relatório |
|---|---|---|
| `repo` | string | Nome canônico `owner/name` |
| `number` | integer | Número do PR |
| `title` | string | Título integral |
| `url` | string | Link retornado pela API |
| `author` | string ou null | Login do autor |
| `is_draft` | boolean | Rascunho |
| `created_at`, `updated_at` | string | Datas em RFC 3339 |
| `observed_at` | string | Momento UTC da consulta dos dados principais |
| `conversation_comments_count` | integer ou null | Comentários gerais |
| `review_threads_count` | integer ou null | Discussões de código |
| `unresolved_threads_count` | integer ou null | Discussões ainda não resolvidas |
| `has_comments` | boolean ou null | Indicador definido na seção 5 |
| `review_decision` | string ou null | Valor original da API |
| `mergeable` | string ou null | Valor original da API |
| `merge_state_status` | string ou null | Valor original da API |
| `blocked` | boolean ou null | Classificação conservadora da seção 6 |
| `signals` | array de strings | Indicadores derivados, sem ordem de prioridade |
| `details_complete` | boolean | Todos os campos necessários foram consultados |

Números de PR só são únicos dentro de um repositório. Identificar registros por `(repo, number)`.

A seção 1 do [contrato de casos de borda](docs/contract-edge-cases.md) define quais campos ausentes invalidam a página e quais preservam o PR com detalhe incompleto. `author = null` e `reviewDecision = null` explícitos são ausências legítimas; omissão desses campos não é equivalente.

## 5. Comentários e discussões

Na API, `comments` representa comentários gerais; `reviewThreads` expõe discussões de revisão e `isResolved` informa sua resolução. [2]

Definições de produto:

- `has_comments = true` quando houver ao menos um comentário geral ou uma thread de revisão.
- Será `false` apenas se ambas as contagens forem conhecidas e zero.
- Será `null` se não houver evidência positiva e alguma contagem estiver indisponível.
- Uma mensagem publicada somente no corpo de uma review não entra nesse indicador. Essa limitação deve constar na ajuda e no README.
- Comentários de bots, do autor do PR e threads resolvidas entram em `has_comments`.
- `unresolved_threads_count` considera toda thread com `isResolved = false`, inclusive quando marcada como desatualizada.
- Threads pendentes são um sinal de atenção; sua existência não deve, isoladamente, estabelecer `blocked = true`.
- Quantidade de threads não é quantidade de comentários inline. A ferramenta não promete uma contagem total de todas as mensagens.

Conservar as contagens da primeira observação principal aceita, mesmo quando a enumeração posterior de threads divergir; não substituir nem limitar uma contagem pela outra. Total principal zero dispensa enumeração; total positivo ou desconhecido exige enumeração. Ver seção 4 do [contrato de casos de borda](docs/contract-edge-cases.md).

Consultar as contagens gerais via `totalCount`, evitando baixar textos. Se a enumeração de threads falhar antes do fim, a contagem de pendências será `null`; não publicar um subtotal como se fosse definitivo.

## 6. Estado do merge

Referência dos estados usados: `BLOCKED` indica bloqueio; `DIRTY`, conflito; `BEHIND`, branch desatualizada; `CLEAN`, merge possível com status passando; `UNSTABLE`, status não passando; `HAS_HOOKS`, presença de hooks; `UNKNOWN`, estado indeterminado. `mergeable` trata conflitos, não todas as regras de merge. [2]

O produto deve preservar os valores originais e adotar a seguinte política própria, em ordem:

| Condição | `blocked` |
|---|---|
| Campos necessários ao merge incompletos, contradição ou enum desconhecido | `null` |
| `is_draft = true` ou estado `DRAFT` | `true` |
| `mergeable = CONFLICTING` ou estado `DIRTY` | `true` |
| Estado `BLOCKED` | `true` |
| Estado `CLEAN`, `mergeable = MERGEABLE` e sem rascunho | `false` |
| Qualquer outra combinação | `null` |

Uma contradição inclui `CLEAN` junto de rascunho, conflito ou decisão `CHANGES_REQUESTED`. Não tentar resolver inconsistências por adivinhação.

`blocked = false` significa ausência de bloqueio identificado pelos campos consultados naquele instante. Não garante que a identidade autenticada possa executar o merge nem que regras adicionais estejam satisfeitas.

Sinais previstos: `draft`, `conflicts`, `merge_blocked`, `behind`, `nonpassing_status`, `hooks_present`, `changes_requested`, `review_required`, `unresolved_threads`, `merge_unknown`, `inconsistent_merge_data`. Usar somente quando os campos correspondentes sustentarem a indicação. Ordenar alfabeticamente.

Sinais não são uma lista exaustiva de causas do bloqueio. `BLOCKED` sem causa identificável deve continuar aparecendo como bloqueado.

Enum novo retornado pela API deve ser preservado, produzir interpretação desconhecida e um aviso de compatibilidade. Ausência legítima de decisão de revisão não é aprovação nem falha de consulta.

Os campos necessários são `isDraft`, `mergeable`, `mergeStateStatus` e a presença de `reviewDecision` (que aceita `null` explícito). Enum desconhecido em qualquer um dos três campos de enum prevalece inclusive sobre rascunho, conflito ou `BLOCKED`; `UNKNOWN` é conhecido e segue a tabela. Falhas em autor, contagens e threads não impedem classificar merge. Valores conhecidos, sinais e exemplos de precedência estão na seção 2 do [contrato de casos de borda](docs/contract-edge-cases.md).

## 7. Coleta e paginação

`gh api graphql` fornece o acesso autenticado à API. Embora o CLI ofereça `--paginate`, este projeto controlará os cursores em Go para distinguir falhas de páginas de PRs e falhas de detalhes. [1]

Fluxo proposto:

1. Validar entrada, localizar `gh` e inicializar o contexto cancelável.
2. Para cada repositório, consultar `repository.pullRequests` com estado `OPEN`, ordem de criação crescente e páginas de até 100 registros.
3. Buscar dados principais, `comments.totalCount` e `reviewThreads.totalCount`. Evitar conexões aninhadas volumosas nessa consulta.
4. Avançar pelo cursor de PRs até `hasNextPage = false`.
5. Para cada PR com threads, consultar suas threads em páginas de até 100, usando um cursor independente por PR, para contar pendências.
6. Normalizar, ordenar, aplicar o filtro e renderizar o relatório.

Usar queries constantes e variáveis GraphQL passadas como argumentos separados; não interpolar nomes de repositório no texto da query. Uma página deve ser uma invocação de `gh`.

Não usar um limite arbitrário alto como substituto da paginação. O MVP não expõe flag que limite o total de PRs. Cursor repetido, cursor ausente com próxima página ou resposta inválida são erros de coleta.

Para simplificar a recuperação, se uma resposta tiver `errors` GraphQL, considerar aquela página malsucedida, mesmo que contenha `data`. Preservar somente páginas anteriores válidas. Não inferir sucesso apenas pelo exit code do subprocesso.

A paginação de uma conexão é sequencial. A concorrência opcional pode distribuir repositórios e consultas independentes, respeitando um único limite global de processos; não multiplicar pools aninhados.

Deduplicar PRs e threads por seus IDs. A primeira ocorrência em página válida aceita vence integralmente, sem mesclar campos; para threads, conservar seu primeiro `isResolved`. Validar a página antes de deduplicar e rejeitar conflitos de identidade. Regras completas na seção 3 do [contrato de casos de borda](docs/contract-edge-cases.md). A API é observada ao longo de um intervalo, sem snapshot transacional: PRs podem abrir ou fechar durante a execução. Registrar horários de início e fim; completude significa que todas as páginas exigidas foram percorridas com sucesso, não que o resultado corresponda a um instante atômico.

## 8. Falhas e completude

Cada repositório terá dois marcadores: `listing_complete`, para a enumeração de PRs, e `details_complete`, para os detalhes dos PRs coletados. Repositório sem PRs, consultado com sucesso, tem ambos como `true`.

`details_complete` do repositório é a conjunção dos marcadores dos PRs únicos coletados antes do filtro. Sem PRs coletados, é `true` mesmo se a listagem falhar; nesse caso `listing_complete = false` mantém o relatório incompleto. Falha antes da primeira página válida também mantém `repo = null`. Isso não descreve o repositório como vazio. Ver seção 5 do [contrato de casos de borda](docs/contract-edge-cases.md).

O relatório terá `complete = true` somente quando todos os repositórios tiverem ambos os marcadores verdadeiros. Estados de negócio desconhecidos, como `merge_state_status = UNKNOWN`, não tornam a coleta incompleta.

Falha em um repositório não interrompe os demais. Falha de detalhe mantém o PR no relatório, marca o campo afetado como `null` e `details_complete = false`. Repositório inexistente ou inacessível não deve ser descrito como vazio; quando a API não distinguir as causas, usar uma mensagem que preserve essa ambiguidade.

Política de recuperação proposta:

- Até duas novas tentativas por requisição para timeout, falhas de rede e erros temporários do servidor; espera de 1s e 2s, com pequeno jitter.
- Não repetir automaticamente erro de autenticação, permissão, query inválida ou schema incompatível.
- O MVP não implementa detecção específica, espera ou recuperação de rate limits. Respostas desse tipo seguem o tratamento genérico de erro da API, preservando os dados já coletados e marcando a coleta afetada como incompleta. Erros da API sem classificação temporária não são repetidos automaticamente.
- Toda espera deve responder ao cancelamento.
- `Ctrl+C` cancela subprocessos e encerra com código 130; não há garantia de relatório nessa situação.

| Código de saída | Significado |
|---|---|
| `0` | Relatório completo, mesmo vazio ou com PRs bloqueados |
| `1` | Falha operacional que impede a emissão do relatório |
| `2` | Entrada ou flags inválidas |
| `3` | Relatório emitido, mas incompleto; inclui falha de todos os repositórios |
| `130` | Execução cancelada pelo usuário |

## 9. Formatos de saída

### Tabela

Colunas: `REPO`, `PR`, `TÍTULO`, `COMENT.`, `THREADS PEND.`, `REVISÃO`, `MERGE`, `BLOQUEADO`, `URL`.

`COMENT.` representa apenas comentários gerais. O campo booleano abrangente `has_comments` permanece disponível no JSON. `BLOQUEADO` usa `sim`, `não` ou `?`; campos ausentes usam `?`. `MERGE` preserva o estado original.

Ordenar por repositório, sem diferenciar caixa, e número crescente. Remover sequências de controle, tabs e quebras de linha de campos exibidos no terminal. Títulos podem ser truncados a 60 runes com reticências; preservar o texto integral no JSON. Usar `text/tabwriter`, sem cores no MVP.

Com `--fields`, a tabela mostra apenas as colunas escolhidas pelos nomes JSON dos campos de PR, na ordem informada. Sem a flag, mantém as nove colunas acima.

Na tabela, incluir resumo de repositórios completos/incompletos e quantidade de PRs coletados/exibidos. Mensagens operacionais e erros vão para stderr. Para saída vazia, distinguir “nenhum PR aberto”, “nenhum PR atende ao filtro” e “nenhum PR recuperado; consulta incompleta”.

### JSON

Emitir exatamente um objeto válido em stdout, com indentação de dois espaços. Logs e avisos não podem contaminar o JSON. Exemplo ilustrativo:

```json
{
  "schema_version": 1,
  "started_at": "2026-09-21T12:00:00Z",
  "finished_at": "2026-09-21T12:00:04Z",
  "complete": true,
  "filters": {"only_unresolved": false},
  "repositories": [
    {
      "requested_repo": "minha-org/backend",
      "repo": "minha-org/backend",
      "listing_complete": true,
      "details_complete": true,
      "prs_collected": 1
    }
  ],
  "prs_collected": 1,
  "prs_returned": 1,
  "pull_requests": [
    {
      "repo": "minha-org/backend",
      "number": 123,
      "title": "Corrige autenticação",
      "url": "https://github.com/minha-org/backend/pull/123",
      "author": "dev-example",
      "is_draft": false,
      "created_at": "2026-09-20T09:00:00Z",
      "updated_at": "2026-09-21T11:00:00Z",
      "observed_at": "2026-09-21T12:00:01Z",
      "conversation_comments_count": 3,
      "review_threads_count": 2,
      "unresolved_threads_count": 1,
      "has_comments": true,
      "review_decision": "CHANGES_REQUESTED",
      "mergeable": "MERGEABLE",
      "merge_state_status": "BLOCKED",
      "blocked": true,
      "signals": ["changes_requested", "merge_blocked", "unresolved_threads"],
      "details_complete": true
    }
  ],
  "errors": [],
  "warnings": []
}
```

Cada erro contém `repo` (ou null), `pr_number` (ou null), `stage`, `code` e `message`. Estágios: `list_prs` ou `review_threads`. Códigos previstos no MVP: `auth`, `not_found_or_forbidden`, `timeout`, `network`, `api`, `invalid_response`. O código específico `rate_limit` fica para a etapa posterior. Avisos usam o mesmo contexto, com `code` e `message`, sem afetar automaticamente `complete`.

A projeção opcional `--fields repo,number,title` limita apenas as propriedades de cada objeto em `pull_requests` e preserva sua ordem informada. O envelope de `schema_version: 1`, horários, filtros, repositórios, contadores, completude, erros e avisos continua presente. Sem `--fields`, cada PR mantém todos os campos do contrato acima. Com a flag, consumidores devem interpretar os objetos de PR como projeções explícitas do schema 1. Nomes válidos são exatamente os campos de PR listados neste contrato; campos vazios, repetidos ou desconhecidos são inválidos. A projeção não altera a consulta nem a classificação, apenas a apresentação.

Em repositório não resolvido, `repo` será null e `requested_repo` preservará a entrada. Arrays vazios serão `[]`, não null. Campos opcionais serão explicitamente null. Mudanças incompatíveis exigem incremento de `schema_version`.

`--only-unresolved` atua depois da coleta: inclui apenas contagens conhecidas maiores que zero. Registros com contagem desconhecida não entram, mas seus erros e a incompletude permanecem no relatório. `prs_collected` conta antes do filtro; `prs_returned`, depois. A flag não altera os marcadores de completude.

## 10. Implementação em Go

O pacote do comando será `cmd/pr-report`, produzindo o executável `pr-report`. Seus arquivos Go usam `package main`, necessário para um executável; `pr-report` é o nome de distribuição, pois identificadores de pacote Go não aceitam hífen. O módulo será `github.com/fredyranthun/pr-reporter`, correspondente ao repositório de publicação `https://github.com/fredyranthun/pr-reporter.git`.

Estrutura inicial sugerida, com um único pacote Go:

```text
pr-reporter/
  go.mod
  README.md
  cmd/
    pr-report/
      main.go
      config.go
      github.go
      queries.go
      models.go
      normalize.go
      output.go
      *_test.go
      testdata/
```

Compilar localmente com `go build -o pr-report ./cmd/pr-report`. Após publicar uma versão, instalar com `go install github.com/fredyranthun/pr-reporter/cmd/pr-report@latest`; ambos produzem o executável `pr-report`.

Responsabilidades: `main.go` coordena execução e códigos de saída; `config.go` valida entrada; `github.go` executa consultas e paginação; `queries.go` mantém as queries; `models.go` separa DTOs da API e modelo público; `normalize.go` aplica regras; `output.go` renderiza os formatos.

Usar `flag`, `bufio`, `encoding/json`, `context`, `os/exec`, `sync`, `time` e `text/tabwriter`. Não adicionar framework de CLI no MVP.

Executar `gh` com `exec.CommandContext`, sem shell, passando argumentos separadamente. O pacote `os/exec` não interpreta comandos como um shell. [3] Fixar o host como `github.com`, capturar stdout e stderr separadamente e manter o ambiente de autenticação disponível ao subprocesso. Não ler tokens via `gh auth token` nem registrá-los.

Criar uma pequena interface de executor para substituir `gh` por respostas controladas nos testes. Usar tipos anuláveis no modelo público para preservar a diferença entre zero/falso e desconhecido. Não usar saída textual do terminal como fonte de dados.

Fixar a versão mínima do Go em `go.mod` e documentar a versão de `gh` validada no primeiro release. Medir duração por repositório durante desenvolvimento; não estabelecer promessa de latência independente da rede e da quantidade de threads.

## 11. Critérios de aceitação e testes

| Cenário | Resultado exigido |
|---|---|
| TXT com CRLF, comentários, vazios e duplicatas | Lista única e válida |
| Entrada inválida ou lista vazia | Código 2; nenhuma chamada à API |
| Repositório com 205 PRs | Três páginas percorridas; 205 registros únicos |
| PR com 130 threads | Duas páginas; contagem correta de pendências |
| Thread desatualizada e não resolvida | Entra na contagem de pendências |
| Só há texto no corpo de uma review | Não altera `has_comments`, conforme contrato |
| `BLOCKED`, `DIRTY`, rascunho, `CLEAN`, `UNKNOWN` | Classificação conforme seção 6 |
| Dados de merge contraditórios | `blocked = null` e sinal de inconsistência |
| Resposta com enum novo | Preservar valor e indicar interpretação desconhecida |
| Um repo inacessível e outro acessível | Preservar resultados; `complete = false`; código 3 |
| Falha na segunda página de PRs | Preservar primeira página; `listing_complete = false` |
| Falha na paginação das threads | Manter PR; pendências null; detalhe incompleto |
| JSON com `data` e `errors` | Página tratada como falha |
| Repositório vazio com sucesso | Arrays vazios; completo; código 0 |
| Filtro elimina todos os registros | Relatório válido; contadores coerentes |
| Timeout | Tentativas limitadas e canceláveis |
| Concorrência configurada como 3 | Nunca exceder três subprocessos ativos |
| Título com escape de terminal e quebra de linha | Tabela sanitizada; JSON corretamente escapado |

Os casos de borda de T01 e as tarefas responsáveis por implementar suas fixtures estão na seção 6 do [contrato de casos de borda](docs/contract-edge-cases.md). A matriz é uma obrigação de cobertura futura, não uma suíte executável já existente.

Priorizar testes de parser, normalização, paginação e recuperação com fixtures e executor falso. Um teste de integração autenticado será opcional e manual; a suíte padrão deve rodar sem rede nem token. Validar `go test ./...` e, ao introduzir concorrência, `go test -race ./...` em ambiente compatível.

## 12. Sequência de implementação

1. Entrada, executor `gh`, consulta paginada de um repositório e JSON básico.
2. Múltiplos repositórios, falhas parciais e contrato completo do relatório.
3. Threads paginadas, normalização e classificação de merge.
4. Tabela, filtro, timeouts e tentativas limitadas.
5. Concorrência opcional, testes dos critérios, README e binário de release.

O MVP estará concluído quando todos os critérios forem atendidos, a consulta não tiver cortes silenciosos e o README permitir instalar, autenticar e gerar um relatório sem conhecimento do código.

## 13. Próximos passos após o MVP

O tratamento específico de rate limits será implementado em uma etapa posterior:

- Capturar status e cabeçalhos HTTP separadamente do corpo JSON.
- Identificar limites primários e secundários e definir o código `rate_limit` no relatório.
- Definir a política de espera a partir de `Retry-After` e do horário de reset, incluindo ausência desses dados e um limite de espera.
- Coordenar a pausa entre todas as consultas concorrentes e permitir cancelamento durante a espera.
- Definir um orçamento de novas tentativas e testar recuperação, esgotamento e preservação dos resultados parciais.

Esses itens não fazem parte dos critérios de conclusão do MVP.

## 14. Referências oficiais

Consultadas em 21/09/2026. As regras de produto e escolhas de implementação deste documento são propostas para esta ferramenta, não garantias da API.

1. [GitHub CLI — gh api](https://cli.github.com/manual/gh_api): acesso GraphQL, parâmetros e paginação.
2. [GitHub GraphQL — Pull requests](https://docs.github.com/en/graphql/reference/pulls): campos, threads e estados de merge/revisão.
3. [Go — os/exec](https://pkg.go.dev/os/exec): execução de subprocessos e contexto.
