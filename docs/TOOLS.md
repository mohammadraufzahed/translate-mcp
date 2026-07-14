# MCP Tools Reference

`translate-mcp` registers the following tools with the MCP host.

## `translate`

Translate text into a target language.

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `text` | string | yes | - | Text to translate |
| `target_language` | string | yes | - | BCP-47 code, e.g. `es`, `zh-CN`, `fr-FR` |
| `source_language` | string | no | `auto` | Source BCP-47 code or `auto` |
| `provider` | string | no | `translation.default_provider` | Provider name |
| `model` | string | no | provider `default_model` | Specific model ID |
| `context` | string | no | `""` | Domain/tone hint |
| `tone` | string | no | `neutral` | `formal`, `informal`, or `neutral` |
| `use_cache` | boolean | no | `true` | Whether to read/write cache |
| `alternatives` | number | no | `0` | Number of alternative translations |

### Example request

```json
{
  "text": "Hello world",
  "target_language": "es",
  "source_language": "auto",
  "provider": "openai",
  "tone": "neutral"
}
```

### Example response

```json
{
  "translation": "Hola mundo",
  "source_language": "en",
  "target_language": "es",
  "provider": "openai",
  "model": "gpt-4o-mini",
  "confidence": 0.98,
  "cached": false,
  "input_tokens": 3,
  "output_tokens": 3
}
```

## `detect_language`

Detect the language of a text.

### Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `text` | string | yes | Text to analyze |
| `provider` | string | no | Provider to use |

### Example request

```json
{
  "text": "Hola mundo"
}
```

### Example response

```json
{
  "language": "es",
  "confidence": 0.97
}
```

## `batch_translate`

Translate many texts in one call.

### One-to-many mode

```json
{
  "text": "Hello",
  "source_language": "en",
  "targets": ["es", "fr", "de"]
}
```

### Many-to-one mode

```json
{
  "items": [
    {"text": "Hello"},
    {"text": "World"}
  ],
  "target_language": "es",
  "source_language": "en"
}
```

### Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `text` | string | no | Source text for one-to-many mode |
| `targets` | array | no | List of target language codes for one-to-many mode |
| `items` | array | no | List of `{text, target_language?}` objects for many-to-one mode |
| `target_language` | string | no | Default target language for many-to-one mode |
| `source_language` | string | no | `auto` or a BCP-47 code |
| `provider` | string | no | Provider name |
| `model` | string | no | Model ID |
| `context` | string | no | Context hint |
| `tone` | string | no | Tone |

## `translate_document`

Translate a document while preserving structure.

### Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `content` | string | yes | Document content |
| `format` | string | yes | `markdown`, `json`, `xml`, `html`, `plain` |
| `target_language` | string | yes | Target BCP-47 code |
| `source_language` | string | no | Source BCP-47 code or `auto` |
| `provider` | string | no | Provider name |
| `model` | string | no | Model ID |
| `context` | string | no | Context hint |
| `tone` | string | no | Tone |

### JSON i18n example

```json
{
  "content": "{\"nav\":{\"home\":\"Home\",\"about\":\"About\"}}",
  "format": "json",
  "target_language": "es"
}
```

## `list_languages`

List supported language codes.

### Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `provider` | string | no | Provider name |

## `add_glossary_entry`

Add a terminology entry to protect terms from translation.

### Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `source_term` | string | yes | Source term |
| `target_language` | string | yes | Target language code |
| `translation` | string | yes | Translation of the term |
| `context` | string | no | Domain context |
| `case_sensitive` | boolean | no | `false` |

## `get_glossary`

Get all glossary entries, optionally filtered by target language.

### Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `target_language` | string | no | Filter by target language |

## `add_translation_memory`

Store a verified translation for future fuzzy matching.

### Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `source_text` | string | yes | Source text |
| `target_text` | string | yes | Target text |
| `source_language` | string | yes | Source language code |
| `target_language` | string | yes | Target language code |
| `domain` | string | no | Domain tag |
| `project` | string | no | Project tag |

## `search_translation_memory`

Search the translation memory for similar previous translations.

### Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `text` | string | yes | Source text to search |
| `source_language` | string | no | Source language filter |
| `target_language` | string | no | Target language filter |
| `threshold` | number | no | Similarity threshold `0-1`, default `0.8` |
