# grpc-event-bus
gRPC сервис на Go, использующий шину событий, работающую по принципу Publisher-Subscriber

# Как установить 
1) Склонируйте проект
```
git clone https://github.com/cihgt/grpc-event-bus
cd grpc-event-bus
``` 
2) Установите зависимости
```
go mod download
go mod tidy
```
# Как запустить

### 1) Через клиент-сервер
1) Запустите в терминале сервер (/cmd/server/main.go)
```
go run main.go
```
2) Запустите в другом терминале клиента (/cmd/client/client.go)
```
go run client.go
```
Клиент шлет запросы на сервер и выводит события
### 2) Через grpcurl

Для этого способа требуется установка grpcurl

1) Запустите в терминале сервер (/cmd/server/main.go)
```
go run main.go
```
2) Во втором терминале выполните команду
```
grpcurl -plaintext   -d '{"key":"mytopic"}'   localhost:50051 PubSub/Subscribe
```
- В первом терминале (где запущен сервер) вы увидите, что появилась новая подписка
3) В третьем терминале выполните команду
```
grpcurl -plaintext   -d '{"key":"mytopic","data":"hello world"}'   localhost:50051 PubSub/Publish
```
- В первом терминале увидите, что сообщение успешно опубликовано
- Во втором терминале вы увидите сообщение hello world
# Примененные паттерны
1) **Graceful Shutdown**  
   Сервер завершается с использованием graceful shutdown (/cmd/server/main.go)
2) **Dependency Injection***  
   Сервис (тип Service) получает реализации шины и логгера через конструктор NewService(bus, logger)
3) **Конфигурации**  
   Параметры конфигурации сервера читаются из файла config.json
4) **Unit-testing**  
   Применены юнит-тесты для проверки корректности работы шины
