# throttler-wrap

## Тестовое задание
Реализовать throttler-обёртку для типа Transport из стандартной библиотеки (https://golang.org/pkg/net/http/#Transport).
Обёртка должна реализовывать интерфейс RoundTripper (https://golang.org/pkg/net/http/#RoundTripper) и инициализироваться следующими параметрами:

- RoundTripper, который будет оборачиваться
- Лимит запросов в единицу времени (целое число, если равно 0 то throttling не применяется)
- Единица времени учёта (тип time.Duration (https://golang.org/pkg/time/#Duration))
- Список префиксов исключений URL для которых throttling не будет задействован (если список пуст или nil - то исключений нет)
- Флаг быстрого возврата ошибки

## Примечания
Если частота запросов превышает лимит, то запрос должен быть отложен до момента когда его выполнение не вызовет превышение лимита либо завершён со специальной ошибкой (в зависимости от флага быстрого возврата ошибки).

Для учёта частоты запросов можно считать что они выполняются мгновенно, коды возврата не имеют значения, запросы не подпадающие под условия фильтров не учитываются.

Списки префиксов URL могут содержать * в любой части пути.

Нужно помнить что обёртка может использоваться из многих параллельных горутин, а так же может быть использована в цепочке из нескольких обёрток.

## Пример использования:
```go
throttled := NewThrottler(
    http.DefaultTransport,
    60,
    time.Minute, // 60 rpm
    []string{"/servers/*/status", "/network/"}, // except servers status and network operations
    false, // wait on limit
)

client := http.Client{
    Transport: throttled,
}

// ...
// no throttling
resp, err:= client.GET("http://apidomain.com/network/routes") 
// ...

// ... 
// throttling might be used
req := http.NewRequest("PUT", "http://apidomain.com/images/reload", nil)
resp, err:= client.Do(req) 
// ...

// ...
// no throttling
resp, err:= client.GET("http://apidomain.com/servers/1337/status?simple=true") 
// ...
```
