# T01 — Contratos de coleta e interpretação

Decisões normativas que complementam as seções 4–8 da [especificação 1.2](../pr-report-spec.md). Não alteram os campos do JSON nem seu `schema_version: 1`. Os casos ao final devem virar fixtures e testes nas tarefas responsáveis; não representam funcionalidades já implementadas.

## 1. Página inválida versus detalhe incompleto

Validar a página inteira antes de incorporar registros. Diferenciar campo ausente, `null` explícito e valor zero/falso; não deixar a desserialização Go apagar essa distinção. Campos adicionais da API são ignorados.

| Campo ou estrutura consultada | Ausente ou `null` | Valor presente inválido |
|---|---|---|
| Objeto de resposta, `data`, `repository`, `nameWithOwner`, conexão `pullRequests`, array `nodes`, objeto `pageInfo` | Invalidam a página; `repository = null` preserva a ambiguidade inexistente/inacessível | Invalida a página |
| `pageInfo.hasNextPage` | Invalida a página | Deve ser booleano; outro tipo invalida a página |
| `pageInfo.endCursor` | Permitido somente com `hasNextPage = false` | Deve ser string ou `null`; quando há próxima página, string não vazia e ainda não utilizada |
| Cada nó de PR e seus `id`, `number`, `title`, `url`, `isDraft`, `createdAt`, `updatedAt` | Invalidam a página | Tipo/formato inválido invalida a página |
| `author` | Ausente: detalhe incompleto; `null` explícito: autor legitimamente indisponível, detalhe completo | Se objeto, exige `login` string não vazia; outro tipo ou objeto malformado invalida a página |
| `comments.totalCount`, `reviewThreads.totalCount`, incluindo suas conexões | Campo/conexão ausente ou `null`: contagem `null`, detalhe incompleto | Inteiro não negativo obrigatório; outro tipo ou valor invalida a página |
| `reviewDecision` | Ausente: detalhe incompleto; `null` explícito: decisão legitimamente ausente, detalhe completo | String não vazia ou `null`; outro valor invalida a página |
| `mergeable`, `mergeStateStatus` | Ausente ou `null`: campo `null`, detalhe incompleto | String não vazia obrigatória; outro valor invalida a página; enum novo segue seção 2 abaixo |

O nome canônico deve seguir o formato `owner/name` aceito na entrada. `id` é string opaca não vazia, usada internamente. `number` é inteiro positivo; `title` aceita qualquer string, inclusive vazia; `url` é string não vazia; `isDraft` é booleano; datas devem ser strings RFC 3339. Não exigir ordem entre datas nem interpretar uma URL retornada como entrada de repositório.

`nodes: []` é válido; `nodes: null` e elementos `null` não são. Se presente, `errors` deve ser array: qualquer lista não vazia de erros GraphQL invalida a página mesmo com `data`; `errors: []` e ausência de `errors` não constituem erro por si sós. Outro tipo, inclusive `errors: null`, é resposta inválida. Examinar a resposta estruturada mesmo quando o subprocesso falha; exit code diferente de zero continua sendo falha da tentativa, mesmo com dados aparentemente válidos.

Página estruturalmente válida com detalhe anulável ausente é incorporada. Preservar os demais campos e registrar erro `invalid_response`, estágio `list_prs`, repositório e número do PR, identificando os campos afetados na mensagem. Marcar `details_complete = false` nesse PR. Isso não impede avançar o cursor nem, por si só, torna `listing_complete = false`. `author = null` e `reviewDecision = null` explícitos não geram erro. Presença de `reviewDecision` precisa continuar distinguível internamente: ausência e `null` legítimo são ambos emitidos como `review_decision: null`, mas só a ausência é falha.

JSON inválido, tipo inválido ou falta de campo estrutural obrigatório rejeita a página inteira, inclusive registros anteriores ao defeito. Conservar apenas páginas previamente aceitas. Usar `invalid_response` para defeitos de estrutura e os demais códigos da especificação quando a execução/API fornecer causa identificável; não confundir repositório inacessível com vazio.

Na consulta de threads, aplicar as mesmas regras de envelope, `errors` e paginação. O objeto do PR consultado, a conexão `reviewThreads`, `nodes`, cada nó e seus `id` não vazios e `isResolved` booleanos são obrigatórios. PR indisponível ou thread inválida faz falhar a enumeração: manter o PR principal, definir `unresolved_threads_count = null`, marcar detalhe incompleto e registrar erro no estágio `review_threads`. Nunca publicar subtotal de uma enumeração que falhou.

## 2. Dados necessários e precedência no merge

Para classificar `blocked`, são necessários `isDraft`, `mergeable`, `mergeStateStatus` e a presença de `reviewDecision`. `reviewDecision = null` explícito satisfaz esse requisito sem significar aprovação; campo ausente não. `mergeable` e `mergeStateStatus` precisam ser não nulos. Ausência de autor, contagens ou falha na enumeração de threads não impede classificar merge. Portanto, “dados principais incompletos” na tabela da especificação significa dados necessários à classificação, não todo detalhe do PR.

Valores conhecidos para esta interpretação:

- `mergeable`: `MERGEABLE`, `CONFLICTING`, `UNKNOWN`.
- `mergeStateStatus`: `BEHIND`, `BLOCKED`, `CLEAN`, `DIRTY`, `DRAFT`, `HAS_HOOKS`, `UNKNOWN`, `UNSTABLE`.
- `reviewDecision`: `APPROVED`, `CHANGES_REQUESTED`, `REVIEW_REQUIRED`, além de `null` explícito.

Aplicar esta ordem:

1. Campo necessário indisponível, contradição ou qualquer enum desconhecido nos três campos acima produz `blocked = null`.
2. Rascunho (`isDraft = true` ou estado `DRAFT`) produz `true`.
3. `CONFLICTING` ou `DIRTY` produz `true`.
4. Estado `BLOCKED` produz `true`.
5. Estado `CLEAN`, `MERGEABLE` e sem rascunho produz `false`.
6. Demais combinações produzem `null`.

Enum novo prevalece inclusive sobre rascunho, conflito e `BLOCKED`. `UNKNOWN` é um enum conhecido de negócio, não incompatibilidade: `mergeable = UNKNOWN` com estado `BLOCKED` continua bloqueado; com estado `CLEAN` permanece indeterminado.

As contradições consideradas no MVP são `mergeStateStatus = CLEAN` junto de `isDraft = true`, `mergeable = CONFLICTING` ou `reviewDecision = CHANGES_REQUESTED`. Não inferir outras contradições entre campos de significados distintos. Por exemplo, `CLEAN` com `MERGEABLE`, sem rascunho e `REVIEW_REQUIRED` segue a regra de `false`, acompanhado do sinal `review_required`; a classificação não garante permissão de merge. Contradição gera `inconsistent_merge_data`, sem erro de coleta nem incompletude se todos os campos foram consultados.

Preservar cada enum desconhecido e emitir aviso por campo com `code = unknown_enum`, estágio `list_prs`, repositório, número do PR e mensagem identificando campo e valor. Não tornar detalhes ou relatório incompletos apenas por esse aviso. Sinais são independentes da classificação: conservar `draft`, `conflicts`, `merge_blocked` e demais sinais sustentados por valores conhecidos, mesmo se outro campo desconhecido ou uma contradição impedir classificar `blocked`. Não derivar significado de um enum novo. `merge_unknown` indica `mergeable = UNKNOWN`, `mergeStateStatus = UNKNOWN` ou enum desconhecido nesses três campos; não é sinônimo genérico de `blocked = null`. Ordenar e deduplicar os sinais.

## 3. Duplicatas e observação vencedora

A primeira ocorrência em página válida aceita vence integralmente: ordem de páginas da conexão e, dentro da página, ordem em `nodes`. Para PRs, conservar juntos todos os dados principais, contagens, nome canônico e `observed_at` dessa ocorrência. Não mesclar campos, preencher lacunas com duplicatas posteriores nem preferir o maior `updatedAt`. Somente a ocorrência vencedora produz erros de detalhe e avisos de enums. Coletar threads uma vez por PR único. Para threads, conservar o primeiro `isResolved` de cada ID; uma duplicata posterior com resolução diferente não altera a contagem.

Validar a página inteira antes de deduplicar: duplicata estruturalmente malformada ainda invalida sua página. Ocorrências em páginas rejeitadas e tentativas malsucedidas nunca participam da escolha. Deduplicar PRs dentro do repositório e threads dentro do PR. As conexões são sequenciais, então concorrência entre conexões não altera o vencedor. Duplicatas de identidade consistente não geram erro/aviso por serem duplicatas e não contam novamente nos totais.

Identidade conflitante não é atualização: mesmo ID de PR com outro número, ou mesmo `(repo, number)` com outro ID, invalida a página conflitante (`invalid_response`, `list_prs`), preservando páginas anteriores. Mudança de nome canônico entre páginas da mesma listagem também invalida a página; não dividir uma coleta entre dois nomes. Diferenças de título, estado, contagens ou datas em duplicatas de identidade consistente seguem a regra da primeira ocorrência. A validação de conflitos também considera registros da própria página antes de aceitá-la.

## 4. Contagens de threads em momentos diferentes

`conversation_comments_count` e `review_threads_count` são os `totalCount` da observação principal vencedora. `observed_at` é o instante UTC em que essa resposta foi recebida, registrado somente se a página for aceita; todos os PRs novos da página compartilham esse instante. `unresolved_threads_count` conta IDs únicos não resolvidos na enumeração posterior bem-sucedida, aplicando a regra de duplicatas. Não representa necessariamente o mesmo instante. O intervalo `started_at`–`finished_at` delimita essas consultas; o schema 1 não acrescenta timestamp separado para threads.

Não substituir `review_threads_count` pela quantidade enumerada, limitar pendências ao total anterior ou ajustar uma contagem para concordar com outra. Total principal 1 e enumeração posterior de 2 threads pendentes produzem `review_threads_count = 1` e `unresolved_threads_count = 2`. Diferença entre contagens, isoladamente, não gera erro, aviso de incompatibilidade ou incompletude. `has_comments` continua derivado apenas das duas contagens principais, conforme seção 5 da especificação; pendências não são uma terceira entrada desse cálculo.

Com total principal zero conhecido, dispensar a enumeração e definir pendências como zero; essa é a observação principal, não uma garantia de que nenhuma thread surgiu depois. Com total positivo ou desconhecido, enumerar até o fim. Enumeração bem-sucedida pode determinar pendências mesmo com total principal desconhecido, mas não preenche esse total retroativamente nem remove sua incompletude. Enumeração vazia bem-sucedida produz zero; enumeração interrompida produz `null`, independentemente do subtotal observado.

## 5. Completude e falha antes do primeiro PR

No PR, `details_complete` exige todos os dados principais segundo a seção 1 deste documento e pendências conhecidas por enumeração concluída ou total principal zero. `null` legítimo de autor/revisão, enums novos e contradições de negócio não tornam a coleta incompleta. Contagens ou campos necessários ausentes e falhas de threads tornam o detalhe incompleto, mesmo se outros campos permitirem conclusões como `has_comments = true` ou `blocked = true`.

No repositório, `details_complete` é a conjunção dos marcadores dos PRs únicos coletados, antes de qualquer filtro. Sem PRs coletados, essa conjunção é `true`, inclusive quando a listagem falha antes da primeira página válida. Nesse caso, `listing_complete = false`, `prs_collected = 0` e o relatório continua incompleto: detalhes completos de um conjunto vazio não afirmam que o repositório está vazio ou acessível. O nome canônico só é resolvido por página válida aceita; falha antes disso mantém `repo = null` e preserva `requested_repo`.

Com páginas anteriores preservadas, a completude dos detalhes depende somente dos PRs dessas páginas; falha posterior de listagem não a altera automaticamente. Página válida vazia com próxima página seguida de falha conserva o nome canônico já resolvido, detalhes `true` e listagem `false`. Consulta vazia concluída com sucesso tem ambos os marcadores `true`. O relatório é completo somente se ambos os marcadores forem verdadeiros em todos os repositórios; filtro não muda marcadores, erros ou contadores anteriores ao filtro. Falha de todos os repositórios ainda emite relatório incompleto, com saída 3.

## 6. Matriz de fixtures a implementar

Usar respostas controladas e executor falso, sem rede nem token. Cada linha é uma obrigação de cobertura das tarefas indicadas. Onde se diz “demais campos válidos”, incluir todos os campos necessários e concluir a enumeração de threads ou usar total zero. Variar campos individualmente para evitar que um defeito esconda outro.

| Caso | Entrada/variação da fixture | Resultado esperado | Tarefas |
|---|---|---|---|
| C01 | Omitir ou anular cada campo estrutural obrigatório, inclusive `isDraft`; `nodes: null` ou elemento `null` | Rejeitar página inteira, sem valores zero artificiais; listagem incompleta | T05, T08, T09 |
| C02 | Número não positivo, contagem negativa/fracionária, booleano como string, data inválida, autor malformado | Rejeitar página inteira com `invalid_response` | T08 |
| C03 | Omitir/anular cada conexão de contagem ou `totalCount`; omitir autor | Preservar PR, campo `null`, erro contextual; detalhes falsos e listagem concluída se paginação terminar | T05, T08, T17 |
| C04 | `author: null`, `reviewDecision: null`, demais campos válidos | Sem erro por nulidade, detalhes completos; `CLEAN` + `MERGEABLE` sem rascunho resulta em `false` | T05, T08, T15, T17 |
| C05 | Omitir `reviewDecision`, ou omitir/anular cada campo de merge anulável, junto de rascunho/conflito/`BLOCKED` | PR mantido, `blocked: null`, detalhes incompletos e erro contextual | T08, T15 |
| C06 | Autor/contagem ausente ou threads falham; merge completo e `BLOCKED` | `blocked: true`, detalhes incompletos | T13, T15, T17 |
| C07 | Enum novo em cada um dos três campos, combinado separadamente com rascunho, conflito, `BLOCKED` ou `CLEAN` | Raw preservado; `blocked: null`; aviso `unknown_enum`; sinais conhecidos preservados e `merge_unknown`; detalhes completos | T15, T16, T19 |
| C08 | `mergeable: UNKNOWN` com `BLOCKED`, depois com `CLEAN` | Respectivamente `true` e `null`; sinal `merge_unknown`, sem aviso de enum novo ou incompletude | T15, T16, T17 |
| C09 | `CLEAN` junto de cada contradição definida | `blocked: null`, `inconsistent_merge_data` e sinais sustentados; detalhes completos | T15, T16 |
| C10 | `CLEAN`, `MERGEABLE`, sem rascunho, `REVIEW_REQUIRED` | `blocked: false`, `review_required`, sem contradição | T15, T16 |
| C11 | ID de PR repetido na mesma página e entre páginas com título, merge, contagens e `updatedAt` diferentes | Primeiro registro e `observed_at` vencem; uma enumeração de detalhes e um PR nos contadores | T09, T17 |
| C12 | Primeiro PR com detalhe ausente; duplicata posterior completa ou com enum novo | Não preencher lacuna nem emitir aviso da duplicata; incompletude original permanece | T08, T09, T16 |
| C13 | Duplicata estruturalmente inválida, IDs/números conflitantes ou nome canônico alterado | Rejeitar toda a página conflitante; conservar apenas páginas anteriores | T08, T09 |
| C14 | ID de thread repetido na mesma página/entre páginas, primeiro pendente depois resolvido e vice-versa | Primeiro `isResolved` vence; ID conta uma vez | T12 |
| C15 | Total principal 1, enumeração com 2 pendentes; total principal 2, enumeração vazia | Conservar total principal; pendências 2 e 0; nenhum erro/aviso por diferença | T12, T14, T17, T19 |
| C16 | Total principal zero | Nenhuma chamada de threads; pendências zero; observação principal preservada | T12 |
| C17 | Total principal desconhecido, enumeração bem-sucedida com uma pendente, comentários zero | Total `null`, pendências 1, `has_comments: null`, detalhes incompletos e erro original preservado | T12, T14, T17 |
| C18 | Threads com campo obrigatório ausente, PR indisponível ou segunda página falha | PR mantido; pendências `null`, sem subtotal final; erro `review_threads`, detalhes falsos | T12, T13 |
| C19 | Primeira página de PRs falha; todos os repositórios falham | Listagem falsa, detalhes verdadeiros, nome canônico `null`, zero PRs, relatório incompleto, saída 3 | T08, T17, T19, T25 |
| C20 | Página válida vazia seguida de falha; consulta vazia concluída | Primeiro caso: nome resolvido, listagem falsa, detalhes verdadeiros; segundo: ambos verdadeiros | T09, T17 |
| C21 | Primeira página com PRs de detalhes completos/incompletos, segunda falha | Listagem falsa; detalhes refletem apenas PRs preservados; filtro não altera marcadores | T09, T17, T18 |
| C22 | `data` junto de `errors` não vazio, em PRs e threads; erro de processo com JSON válido | Rejeitar tentativa/página; dados dessa tentativa não vencem duplicatas | T08, T09, T13 |
| C23 | Cursor seguinte ausente, nulo, vazio ou repetido; última página com cursor ausente/nulo | Rejeitar os primeiros casos; aceitar os últimos | T08, T09, T12 |
| C24 | `errors` ausente, vazio, nulo ou de tipo incorreto | Ausente/vazio não falham por si; nulo/tipo incorreto invalidam a página | T08, T13 |

T01 está validada pela revisão de coerência desta matriz com as seções 4–9 da especificação. A execução das fixtures depende da implementação posterior; T01 não requer criar comando, modelos Go ou suíte de testes antecipadamente.
