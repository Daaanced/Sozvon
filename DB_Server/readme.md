[1] main.go          — запуск сервера, маршруты, точка входа
[2] handlers/         — обработчики HTTP-запросов
[3] db/               — работа с базой данных: подключение + SQL

## HandleFunc

привязывает (регистрирует) URL-маршрут к обработчику (handler)

запоминает, какую функцию нужно вызвать, когда клиент обратится по данному пути

### pattern string - 
Это путь (URL), на который будет реагировать сервер

HandleFunc не различает HTTP-методы (GET, POST, DELETE)

он реагирует на все методы, пока ты сам не проверишь метод внутри обработчика

http.HandleFunc("/users", myHandler)
Теперь любой запрос на /users (GET, POST, PUT…) вызывает myHandler().

### handler func(ResponseWriter, *Request)
Это функция-обработчик, которую сервер вызовет при обращении по указанному пути.

Она всегда должна иметь форму:
func handlerName(w http.ResponseWriter, r *http.Request)

w http.ResponseWriter — куда писать ответ клиенту
Это интерфейс, который позволяет:
писать текст - fmt.Fprintln(w, "Hello!")
отправлять статус-код - w.WriteHeader(201)
отправлять JSON - json.NewEncoder(w).Encode(data)
ставить заголовки - w.Header().Set("Content-Type", "application/json")

### r *http.Request
информация о запросе
Внутри запроса хранится:
| что                       | где             |
| ------------------------- | --------------- |
| HTTP метод                | `r.Method`      |
| параметры URL             | `r.URL.Query()` |
| путь                      | `r.URL.Path`    |
| тело запроса (JSON, Form) | `r.Body`        |
| заголовки                 | `r.Header`      |
| куки                      | `r.Cookie(...)` |
| IP клиента                | `r.RemoteAddr`  |

например:

method := r.Method           // GET / POST / DELETE
name := r.URL.Query().Get("name")

Пример полного обработчика:
func UsersHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case "GET":
        fmt.Fprintf(w, "Get users")
    case "POST":
        fmt.Fprintf(w, "Create user")
    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

Регистрация:
http.HandleFunc("/users", UsersHandler)

## Итог

HandleFunc(pattern, handler) делает следующее:

pattern → определяет по какому пути реагировать

handler → определяет какую функцию вызывать

Go вызывает handler(w, r)

w → позволяет отправить ответ клиенту

r → содержит весь запрос