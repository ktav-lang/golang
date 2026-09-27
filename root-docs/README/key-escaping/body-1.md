>>>>> lang=en
## Key escaping

Since spec 0.6.4 a literal `.` or `:` inside a key segment is written
with a backslash:

```text
## The key is the single segment "a.b".
a\.b: v
## The key is the single segment "a:b".
a\:b: v
## This is the path ["x", "y.z"].
x.y\.z: v
```

A literal backslash in a key is `\\`.

>>>>> lang=ru
## Экранирование в ключах

Начиная со spec 0.6.4 литеральные `.` или `:` внутри сегмента ключа
записываются через backslash:

```text
## Ключ состоит из одного сегмента "a.b".
a\.b: v
## Ключ состоит из одного сегмента "a:b".
a\:b: v
## Это путь ["x", "y.z"].
x.y\.z: v
```

Литеральный backslash в ключе пишется как `\\`.

>>>>> lang=zh
## 键的转义

自 spec 0.6.4 起,键段内的字面量 `.` 或 `:` 通过反斜杠书写:

```text
## 键由单个段 "a.b" 组成。
a\.b: v
## 键由单个段 "a:b" 组成。
a\:b: v
## 这是路径 ["x", "y.z"]。
x.y\.z: v
```

键中的字面量反斜杠写作 `\\`。

