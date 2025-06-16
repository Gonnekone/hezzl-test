# Hezzle-test

## Технологии
- Go
- PostgreSQL
- Nats
- Redis
- Docker


## Использование
### 📄 Получение списка товаров
**GET** `/goods/list?limit=10&offset=0`

**Пример ответа:**
```json
{
  "meta": {
    "total": 4,
    "removed": 1,
    "limit": 10,
    "offset": 0
  },
  "goods": [
    {
      "id": 2,
      "projectId": 1,
      "name": "Banana",
      "description": "NO DESC",
      "priority": 3,
      "removed": false,
      "createdAt": "2025-06-15T23:30:55.898748Z"
    },
    {
      "id": 3,
      "projectId": 1,
      "name": "Carrot",
      "description": "NO DESC",
      "priority": 4,
      "removed": false,
      "createdAt": "2025-06-15T23:30:59.989213Z"
    },
    {
      "id": 4,
      "projectId": 1,
      "name": "Donut",
      "description": "NO DESC",
      "priority": 1,
      "removed": false,
      "createdAt": "2025-06-15T23:31:03.524348Z"
    }
  ]
}
```

---

### ➕ Добавление товара
**POST** `/goods/create?projectId=1`

**Пример запроса:**
```json
{
  "name": "Mango"
}
```

**Пример ответа:**
```json
{
  "id": 1,
  "projectId": 1,
  "name": "Mango",
  "description": "NO DESC",
  "priority": 1,
  "removed": false,
  "createdAt": "2025-06-16T19:00:41.223684Z"
}
```

---

### ✏️ Редактирование товара
**PATCH** `/good/update?id=1&projectId=1`

**Пример запроса:**
```json
{
  "name": "Apple",
  "description": "yoooo"
}
```

**Пример ответа:**
```json
{
  "id": 1,
  "projectId": 1,
  "name": "Apple",
  "description": "yoooo",
  "priority": 1,
  "removed": false,
  "createdAt": "2025-06-16T19:00:41.223684Z"
}
```

---

### ❌ Удаление товара
**DELETE** `/good/remove?id=1&projectId=1`

**Пример ответа:**
```json
{
  "id": 1,
  "projectId": 1,
  "removed": true
}
```

---

### 🔁 Изменение приоритета товара
**POST** `/good/reprioritize?id=4&projectId=1`

**Пример запроса:**
```json
{
  "newPriority": 1
}
```

**Пример ответа:**
```json
{
  "priorities": [
    {
      "id": 4,
      "priority": 0
    },
    {
      "id": 1,
      "priority": 3
    },
    {
      "id": 2,
      "priority": 4
    },
    {
      "id": 3,
      "priority": 5
    }
  ]
}
```


### Требования:
- [Task](https://taskfile.dev/)
- [Docker](https://www.docker.com/)

### Запуск
#### Core из .core/
```sh
task run
```
```sh
task migrate-up
```

#### Listener из .clickhouse-service/
```sh
task run
```
```sh
task migrate-up
```