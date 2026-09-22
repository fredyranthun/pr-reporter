# pr-report

Relatório local de pull requests abertos no GitHub, incluindo rascunhos, comentários gerais, discussões de revisão pendentes e estado do merge. A ferramenta só lê dados de `github.com`; não altera PRs e não mantém banco de dados.

## Instalação

Requer Linux (primeiro alvo), Go **1.25.0 ou mais recente** para compilar e [GitHub CLI (`gh`)](https://cli.github.com/) no `PATH`. O desenvolvimento foi validado com Go 1.25.1 em Linux/amd64 e `gh` **2.100.0**. Autentique-se manualmente antes de consultar repositórios:

```sh
gh auth login --hostname github.com
gh auth status --hostname github.com
```

Clone o projeto e compile ou instale localmente:

```sh
git clone https://github.com/fredyranthun/pr-reporter.git
cd pr-reporter
go build -o pr-report ./cmd/pr-report
./pr-report --version
# Alternativa: go install ./cmd/pr-report
```

Após o primeiro release, a instalação por versão publicada será:

```sh
go install github.com/fredyranthun/pr-reporter/cmd/pr-report@latest
```

`go install` coloca o executável em `GOBIN` ou, se não configurado, em `GOPATH/bin`; inclua esse diretório no `PATH`. O binário de release Linux/amd64 e as instruções de verificação ficarão na página de releases.

## Primeiro relatório

```sh
./pr-report --repo minha-org/backend
./pr-report --repo minha-org/backend --repo minha-org/frontend
./pr-report --repos repos.txt --format json > prs.json
./pr-report --repos repos.txt --only-unresolved
./pr-report --repos repos.txt --concurrency 3 --timeout 45s
```

`repos.txt` é UTF-8, aceita BOM apenas no início, LF ou CRLF, um `owner/name` por linha, linhas vazias e comentários de linha inteira iniciados por `#` após espaços. Valores também podem vir de flags `--repo` repetidas; arquivo e flags são combinados e duplicatas são removidas sem diferenciar maiúsculas de minúsculas. URLs, caminhos locais, sufixos `.git` e comentários ao final da linha são inválidos. Toda a entrada é validada antes da primeira chamada à API.

| Flag | Padrão | Uso |
|---|---|---|
| `--repos <arquivo>` | ausente | Lê uma lista TXT |
| `--repo <owner/name>` | ausente | Adiciona repositório; pode repetir |
| `--format table\|json` | `table` | Seleciona tabela ou JSON |
| `--only-unresolved` | `false` | Exibe apenas PRs com pendências confirmadas |
| `--concurrency <n>` | `1` | Limite global de processos `gh`, de 1 a 8 |
| `--timeout <duração>` | `30s` | Prazo por tentativa de consulta |
| `--help`, `-h` | — | Exibe ajuda sem consultar a API |
| `--version` | — | Exibe versão sem consultar a API |

A tabela mostra repositório, número, título, comentários gerais, threads pendentes, decisão de revisão, estado de merge, bloqueio e URL. `?` indica dado indisponível; `sim`, `não` e `?` são os estados de bloqueio. Títulos acima de 60 caracteres Unicode exibidos recebem reticências; a saída JSON mantém o título integral. O resumo mostra repositórios completos/incompletos e PRs coletados/exibidos.

O JSON em stdout contém um objeto indentado de `schema_version: 1`. Os campos e regras completas estão em [pr-report-spec.md](pr-report-spec.md), §§4 e 9. `repositories` mantém `requested_repo` e o nome canônico `repo` retornado pela API, ou `null` se a consulta falhar antes da primeira página válida. `pull_requests` contém somente PRs únicos. `prs_collected` é anterior ao filtro; `prs_returned`, posterior. Campos indisponíveis são `null` e arrays vazios são `[]`. Erros e avisos contextuais também aparecem no JSON; mensagens de diagnóstico vão para stderr, sem contaminar stdout.

`--only-unresolved` inclui somente contagens `unresolved_threads_count` conhecidas e maiores que zero. Contagens desconhecidas ficam fora do conjunto exibido, mas os erros, marcadores de completude e total coletado permanecem no relatório. Um relatório parcial preserva PRs de páginas válidas anteriores e nunca afirma que um repositório inacessível está vazio.

| Código de saída | Significado |
|---|---|
| `0` | Relatório completo, inclusive vazio ou com PRs bloqueados |
| `1` | Falha operacional que impediu a emissão do relatório |
| `2` | Entrada ou flags inválidas |
| `3` | Relatório emitido com coleta incompleta |
| `130` | Cancelamento pelo usuário (`Ctrl+C`) |

## Interpretação e limites

`comments` conta comentários gerais; `reviewThreads` conta discussões de código, não mensagens inline individuais. Ambas incluem participantes como bots e autor; threads resolvidas ainda entram no indicador `has_comments`. Texto presente **somente no corpo de uma review** não entra nessa contagem. Thread desatualizada sem resolução ainda conta como pendente. Um total principal de threads zero dispensa enumeração; com total positivo ou desconhecido, a ferramenta enumera todas as páginas necessárias.

`blocked` é uma interpretação conservadora dos campos retornados pela API. Rascunho, conflito e `BLOCKED` podem produzir `true`; `CLEAN` com `MERGEABLE`, sem contradição, pode produzir `false`; dados incompletos, contraditórios ou enum novo produzem `null`. Threads pendentes não estabelecem bloqueio sozinhas. `false` não garante permissão de merge nem avaliação completa de rulesets, bypass ou merge queues. Os enums originais são preservados e valores desconhecidos recebem avisos.

A coleta não é um snapshot transacional: PRs e threads podem mudar entre páginas. `started_at`, `finished_at` e `observed_at` delimitam as observações; contagens principais podem divergir da enumeração posterior de threads. Requisições temporariamente falhas podem ser tentadas mais duas vezes com espera cancelável; rate limits ainda seguem o tratamento genérico de erro da API, sem espera específica. GitHub Enterprise Server, corpos de comentários/reviews, alterações de PR e acompanhamento histórico estão fora deste MVP.

## Desenvolvimento e verificação manual

A suíte padrão usa respostas controladas, sem rede e sem token:

```sh
GOPROXY=off GOTOOLCHAIN=local go test ./...
GOPROXY=off GOTOOLCHAIN=local go test -race ./...
go build -o pr-report ./cmd/pr-report
```

[docs/acceptance-coverage.md](docs/acceptance-coverage.md) associa os 18 critérios de aceitação a testes. Para uma verificação autenticada opcional, execute `gh auth status --hostname github.com` e depois `./pr-report --repo owner/name --format json` em um repositório que você possa acessar. Confirme que stdout contém um objeto JSON e que `complete` e o código de saída refletem a coleta. A consulta usa a autenticação já configurada no `gh`; a ferramenta não abre login nem extrai tokens.
