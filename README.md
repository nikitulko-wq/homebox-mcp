# homebox-mcp

MCP-сервер для [Homebox](https://homebox.dev) (v0.26+, API `/entities` + `/tags`), написанный на Go с использованием официального [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk). Транспорт — stdio.

В отличие от read-only аналогов, поддерживает полный цикл: поиск, создание, редактирование, удаление айтемов и загрузку изображений/файлов.

## Возможности (11 инструментов)

**Чтение:**

| Инструмент | Описание |
|---|---|
| `search_items` | Поиск по тексту с фильтрами по тегам/локациям, пагинация |
| `get_item` | Полная карточка айтема по UUID |
| `get_item_by_asset_id` | Поиск по asset ID (`000-123`) |
| `list_locations` | Все локации (id, name, description) |
| `list_tags` | Все теги |
| `list_entity_types` | Типы сущностей (для нестандартных create) |
| `get_stats` | Статистика группы: айтемы, локации, сумма, гарантия |

**Запись:**

| Инструмент | Описание |
|---|---|
| `create_item` | Создание айтема (name обязателен; parentId — локация, tagIds — теги) |
| `update_item` | Частичное обновление: меняются только переданные поля |
| `delete_item` | Удаление айтема по UUID |
| `add_attachment` | Загрузка фото/документа из локального файла; `primary=true` делает фото главным |

## Сборка

```bash
go build -o homebox-mcp .
```

Требуется Go 1.23+.

## Конфигурация

Переменные окружения (все обязательны):

| Переменная | Пример |
|---|---|
| `HOMEBOX_URL` | `https://homebox.example.com` |
| `HOMEBOX_EMAIL` | `you@example.com` |
| `HOMEBOX_PASSWORD` | `secret` |

Сервер логинится через `POST /api/v1/users/login` и хранит Bearer-токен в памяти; при 401 или истечении токена автоматически логинится повторно. Весь лог — в stderr (stdout занят протоколом MCP).

> Примечание: нужен Homebox **v0.26.0 или новее**. В v0.26 API сменился с `/items`, `/locations`, `/labels` на единые `/entities`, `/tags`, `/entity-types` — этот сервер использует только новое API. Версию вашего инстанса можно узнать на `GET <HOMEBOX_URL>/api/v1/status`.

## Подключение к Claude Code

```bash
claude mcp add homebox \
  --env HOMEBOX_URL=https://homebox.example.com \
  --env HOMEBOX_EMAIL=you@example.com \
  --env HOMEBOX_PASSWORD=secret \
  -- /path/to/homebox-mcp
```

Для Claude Desktop добавьте в `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "homebox": {
      "command": "/path/to/homebox-mcp",
      "env": {
        "HOMEBOX_URL": "https://homebox.example.com",
        "HOMEBOX_EMAIL": "you@example.com",
        "HOMEBOX_PASSWORD": "secret"
      }
    }
  }
}
```

## Примеры запросов

- «Что у меня лежит в коробке "Инструменты"?» → `list_locations` + `search_items` с `locationIds`
- «Добавь перфоратор Bosch GBH 2-26, куплен 2026-03-01 за 8500 в "Кладовке"» → `create_item` + `update_item` (purchaseDate/purchasePrice)
- «Прикрепи фото чека к перфоратору как гарантию» → `add_attachment` (type=warranty)
- «Продай его: продан 2026-09-01 за 6000 Васе» → `update_item` (soldDate, soldPrice, soldTo)

## Лицензия

MIT
