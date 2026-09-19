# homebox-mcp

An MCP server for [Homebox](https://homebox.dev) (v0.26+, `/entities` + `/tags` API), written in Go using the official [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk). Transport: stdio.

Unlike read-only alternatives, it supports the full lifecycle: searching, creating, editing and deleting items, plus uploading photos and files.

## Features (11 tools)

**Read:**

| Tool | Description |
|---|---|
| `search_items` | Text search with tag/location filters and pagination |
| `get_item` | Full item details by UUID |
| `get_item_by_asset_id` | Lookup by asset ID (`000-123`) |
| `list_locations` | All locations (id, name, description) |
| `list_tags` | All tags |
| `list_entity_types` | Entity types (for non-default create) |
| `get_stats` | Group statistics: items, locations, total value, warranty |

**Write:**

| Tool | Description |
|---|---|
| `create_item` | Create an item (name required; parentId = location, tagIds = tags) |
| `update_item` | Partial update: only the fields you pass are changed |
| `delete_item` | Delete an item by UUID |
| `add_attachment` | Upload a photo/document from a local file; `primary=true` makes a photo the item's main image |

## Build

```bash
go build -o homebox-mcp .
```

Requires Go 1.23+.

## Configuration

Environment variables (all required):

| Variable | Example |
|---|---|
| `HOMEBOX_URL` | `https://homebox.example.com` |
| `HOMEBOX_EMAIL` | `you@example.com` |
| `HOMEBOX_PASSWORD` | `secret` |

The server logs in via `POST /api/v1/users/login` and keeps the Bearer token in memory; on a 401 or token expiry it automatically logs in again. All logging goes to stderr (stdout is reserved for the MCP protocol).

> Note: Homebox **v0.26.0 or newer** is required. In v0.26 the API changed from `/items`, `/locations`, `/labels` to the unified `/entities`, `/tags`, `/entity-types` — this server uses only the new API. You can check your instance version at `GET <HOMEBOX_URL>/api/v1/status`.

## Connecting to Claude Code

```bash
claude mcp add homebox \
  --env HOMEBOX_URL=https://homebox.example.com \
  --env HOMEBOX_EMAIL=you@example.com \
  --env HOMEBOX_PASSWORD=secret \
  -- /path/to/homebox-mcp
```

For Claude Desktop, add to `claude_desktop_config.json`:

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

## Example prompts

- "What's in my Tools box?" → `list_locations` + `search_items` with `locationIds`
- "Add a Bosch GBH 2-26 rotary hammer, bought 2026-03-01 for $250, in the Storage Room" → `create_item` + `update_item` (purchaseDate/purchasePrice)
- "Attach a photo of the receipt to the hammer as a warranty document" → `add_attachment` (type=warranty)
- "Mark it as sold: 2026-09-01 for $180 to Bob" → `update_item` (soldDate, soldPrice, soldTo)

## License

MIT

---

<details>
<summary>Русская версия</summary>

MCP-сервер для [Homebox](https://homebox.dev) (v0.26+, API `/entities` + `/tags`), написанный на Go с использованием официального [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk). Транспорт — stdio.

В отличие от read-only аналогов, поддерживает полный цикл: поиск, создание, редактирование, удаление айтемов и загрузку изображений/файлов.

### Возможности (11 инструментов)

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

### Сборка

```bash
go build -o homebox-mcp .
```

Требуется Go 1.23+.

### Конфигурация

Переменные окружения (все обязательны):

| Переменная | Пример |
|---|---|
| `HOMEBOX_URL` | `https://homebox.example.com` |
| `HOMEBOX_EMAIL` | `you@example.com` |
| `HOMEBOX_PASSWORD` | `secret` |

Сервер логинится через `POST /api/v1/users/login` и хранит Bearer-токен в памяти; при 401 или истечении токена автоматически логинится повторно. Весь лог — в stderr (stdout занят протоколом MCP).

> Примечание: нужен Homebox **v0.26.0 или новее**. В v0.26 API сменился с `/items`, `/locations`, `/labels` на единые `/entities`, `/tags`, `/entity-types` — этот сервер использует только новое API. Версию вашего инстанса можно узнать на `GET <HOMEBOX_URL>/api/v1/status`.

### Подключение к Claude Code

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

### Примеры запросов

- «Что у меня лежит в коробке "Инструменты"?» → `list_locations` + `search_items` с `locationIds`
- «Добавь перфоратор Bosch GBH 2-26, куплен 2026-03-01 за 8500 в "Кладовке"» → `create_item` + `update_item` (purchaseDate/purchasePrice)
- «Прикрепи фото чека к перфоратору как гарантию» → `add_attachment` (type=warranty)
- «Продай его: продан 2026-09-01 за 6000 Васе» → `update_item` (soldDate, soldPrice, soldTo)

### Лицензия

MIT

</details>
