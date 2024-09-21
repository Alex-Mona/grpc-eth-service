### Порядок запуска проекта:
```
go run cmd/server/server.go
```
```
go run cmd/client/client.go
```

### Запуск теста 2го метода:
```
go run cmd/server/server.go
```
```
go run test/client_2nd_method.go
```
### 1. Настройка gRPC-сервера
Начнём с создания gRPC-сервера. `google.golang.org/grpc`.

#### Создание proto файла:
1. **Создание `.proto` файла**:
   Определите интерфейсы методов `GetAccount` и `GetAccounts` в `.proto` файле:

```syntax = "proto3";

option go_package = "internal/proto";

service AccountService {
    rpc GetAccount(GetAccountRequest) returns (GetAccountResponse);
    rpc GetAccounts(stream GetAccountsRequest) returns (stream GetAccountsResponse);
}

message GetAccountRequest {
    string ethereum_address = 1;
    string crypto_signature = 2;
}

message GetAccountResponse {
    string gastoken_balance = 1;
    uint64 wallet_nonce = 2;
}

message GetAccountsRequest {
    repeated string ethereum_addresses = 1;
    string erc20_token_address = 2;
}

message GetAccountsResponse {
    string ethereum_address = 1;
    string erc20_balance = 2;
}

```

2. **Сгенерируйте Go код** с помощью `protoc`:
```bash
 protoc --go_out=. --go-grpc_out=. api/account.proto
```
### Генерации Go-кода на основе ABI контракта
Чтобы использовать команду `abigen` для генерации Go-кода на основе ABI контракта, необходимо установить инструмент `abigen`. Этот инструмент поставляется вместе с пакетом `go-ethereum`. Вот шаги для установки:

### 2. Установка abigen

1. **Установите `go-ethereum` и его инструменты:**

   Убедитесь, что у вас установлены все необходимые инструменты Go для работы с Ethereum, включая `abigen`:

   ```bash
   go install github.com/ethereum/go-ethereum/cmd/abigen@latest
   ```

2. **Проверьте установку:**

   После установки убедитесь, что `abigen` успешно установлен. Выполните в терминале:

   ```bash
   abigen --version
   ```

   Если команда выведет версию, значит `abigen` установлен и готов к использованию.

3. **Генерация Go-кода из ABI:**

   Теперь вы можете использовать `abigen` для генерации Go-кода из ABI контракта. Например:

   ```bash
   abigen --abi=erc20.abi --pkg=erc20 --out=erc20.go
   ```

   - `--abi=erc20.abi`: путь к вашему файлу ABI.
   - `--pkg=erc20`: имя пакета, которое будет использоваться в сгенерированном файле.
   - `--out=erc20.go`: файл, в который будет сгенерирован Go-код.

После этого вы сможете использовать сгенерированный Go-код для взаимодействия с контрактом ERC-20 в вашем проекте.
```
abigen --abi=abi/erc20.abi --pkg=erc20 --out=pkg/erc20/erc20.go
```

### 3. cmd/server/server.go - go run cmd/server/server.go запуск сервера
В предоставленном сервере реализованы два метода:

1. **GetAccount()**
   - **Запрос**: `{ ethereum_address, crypto_signature }`
   - **Ответ**: `{ gastoken_balance, wallet_nonce }`
   - **Описание**: 
     Этот метод принимает Ethereum-адрес и криптографическую подпись. Он валидирует подпись на основе адреса и возвращает баланс токенов и nonce кошелька.
   - **Логика**:
     - Подпись клиента передается в виде строки, закодированной в Base64.
     - Подпись декодируется с использованием `base64.StdEncoding.DecodeString`.
     - Метод `isValidSignature` используется для проверки, что подпись соответствует публичному ключу, связанному с указанным Ethereum-адресом.
     - Если подпись корректна, возвращаются фиктивные данные о балансе и nonce (пока что это жестко заданные значения, которые можно заменить реальными).

2. **GetAccounts()** (стриминг, двунаправленный поток)
   - **Запрос**: `{ [ ] ethereum_addresses, erc20_token_address }`
   - **Ответ**: `{ [ ] { ethereum_address, erc20_balance } }`
   - **Описание**: 
     Этот метод получает поток запросов с адресами Ethereum и адресом ERC-20 токена. Для каждого Ethereum-адреса возвращается баланс указанного ERC-20 токена.
   - **Логика**:
     - Получаем поток запросов с адресами Ethereum и токеном ERC-20.
     - Для каждого адреса вызывается функция `getERC20Balance`, которая подключается к Ethereum-узлу и вызывает контракт ERC-20 для получения баланса.
     - Для каждого запроса возвращается соответствующий баланс.

### Объяснение функций сервера:

Давай разберём функции сервера по порядку:

### 1. `Server`
```go
type Server struct {
    pb.UnimplementedAccountServiceServer
}
```
Это структура, которая реализует интерфейс gRPC-сервиса. В данном случае она наследует методы из `pb.UnimplementedAccountServiceServer`, что позволяет добавлять свои методы.

### 2. `GenerateECDSAKeys`
```go
func GenerateECDSAKeys() (*ecdsa.PrivateKey, error) {
    privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
    if err != nil {
        return nil, err
    }
    return privateKey, nil
}
```
Эта функция генерирует новую пару ключей ECDSA (закрытый и открытый) с использованием кривой `S256`. Если генерация успешна, возвращается закрытый ключ, иначе — ошибка.

### 3. `SignMessage`
```go
func SignMessage(privateKey *ecdsa.PrivateKey, msg []byte) ([]byte, error) {
    hash := sha256.Sum256(msg)
    signature, err := crypto.Sign(hash[:], privateKey)
    if err != nil {
        return nil, err
    }
    return signature, nil
}
```
Функция подписывает сообщение с использованием закрытого ключа. Сначала сообщение хешируется с помощью SHA-256, затем создаётся подпись. Возвращает подпись или ошибку.

### 4. `VerifySignature`
```go
func VerifySignature(pubKey *ecdsa.PublicKey, msg []byte, signature []byte) bool {
    hash := sha256.Sum256(msg)
    recoveredPubKey, err := crypto.SigToPub(hash[:], signature)
    if err != nil {
        return false
    }
    return recoveredPubKey.Equal(pubKey)
}
```
Эта функция проверяет подпись, восстанавливая открытый ключ из подписи и сравнивая его с переданным открытым ключом. Если они совпадают, подпись считается действительной.

### 5. `GetAccount`
```go
func (s *Server) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
    ethereumAddress := common.HexToAddress(req.GetEthereumAddress())
    cryptoSignature := req.GetCryptoSignature()
    ...
    return &pb.GetAccountResponse{
        GastokenBalance: gasTokenBalance,
        WalletNonce:     walletNonce,
    }, nil
}
```
Метод обрабатывает запрос на получение информации об аккаунте. Он принимает адрес Ethereum и подпись, декодирует подпись и проверяет её. Возвращает информацию о балансе токена и nonce для аккаунта. Если что-то идёт не так, возвращает ошибку.

### 6. `GetAccounts`
```go
func (s *Server) GetAccounts(stream pb.AccountService_GetAccountsServer) error {
    for {
        req, err := stream.Recv()
        ...
        for _, ethAddress := range req.GetEthereumAddresses() {
            balance := getERC20Balance(ethAddress, req.GetErc20TokenAddress())
            ...
        }
    }
    return nil
}
```
Этот метод позволяет получать балансы для нескольких адресов Ethereum с использованием стриминга. Он принимает запросы в цикле и отправляет ответы для каждого адреса. Вызывает функцию `getERC20Balance` для получения баланса токена.

### 7. `getERC20Balance`
```go
func getERC20Balance(address, tokenAddress string) string {
    ...
    return balance.String()
}
```
Эта функция получает баланс токена ERC-20 для указанного адреса. Она устанавливает соединение с узлом Ethereum, получает экземпляр контракта токена и вызывает метод `balanceOf`, чтобы получить баланс.

### 8. `isValidSignature`
```go
func isValidSignature(address common.Address, signature []byte) bool {
    ...
    return crypto.PubkeyToAddress(*pubKey) == address
}
```
Функция проверяет, соответствует ли восстановленный из подписи открытый ключ заданному адресу Ethereum. Если они совпадают, значит, подпись действительна.

### `main`
```go
func main() {
    ...
    grpcServer := grpc.NewServer()
    pb.RegisterAccountServiceServer(grpcServer, &Server{})
    ...
}
```
Основная функция инициализирует сервер, регистрирует реализованный сервис и запускает gRPC-сервер.

### 4. Подключение к узлу Ethereum через публичные сервисы как Infura

Infura
Infura — это сервис, который предоставляет доступ к узлам сети Ethereum через API. Он позволяет подключаться к сети Ethereum без необходимости самостоятельно запускать полный узел. Это удобно, когда вам нужно взаимодействовать с блокчейном, но вы не хотите поддерживать и управлять своим узлом.

Как использовать Infura?
Создать аккаунт на Infura:

Зайдите на сайт Infura и зарегистрируйтесь. https://app.infura.io/
Создайте новый проект на панели управления. При создании вам будет выдан API ключ (project ID), который нужен для подключения.
Получить свой Infura Project ID:

После создания проекта вы увидите строку с форматом https://mainnet.infura.io/v3/YOUR_INFURA_PROJECT_ID, где YOUR_INFURA_PROJECT_ID — это ваш уникальный идентификатор. Используйте этот URL для подключения к сети Ethereum через Infura.

### 5. cmd/client/client.go в коде клиентского приложения происходит следующее: - go run cmd/client/client.go

1. **Подключение к gRPC серверу**: С помощью `grpc.Dial` устанавливается соединение с сервером, работающим на `localhost:50051`. Используется `grpc.WithInsecure()`, что указывает на отсутствие шифрования (нужно помнить, что в реальном приложении стоит использовать шифрованные соединения).

2. **Генерация Ethereum-кошелька**:
   - С помощью функции `ecdsa.GenerateKey` создается новый приватный ключ, который используется для подписания сообщений.
   - Публичный ключ преобразуется в Ethereum-адрес с использованием `crypto.PubkeyToAddress`.

3. **Создание сообщения для подписи**:
   - Сообщение `"some message"` хэшируется с помощью SHA-256, чтобы подготовить его для подписания. Это стандартная практика для цифровых подписей.

4. **Подписание сообщения**:
   - С помощью функции `crypto.Sign` хэш сообщения подписывается приватным ключом. Подпись — это криптографическое доказательство того, что сообщение было создано владельцем приватного ключа.

5. **Кодирование подписи**:
   - Подпись кодируется в формат Base64 для передачи через gRPC. Это необходимо, поскольку подписи являются бинарными данными, а протоколы передачи сообщений часто требуют текстовых данных.

6. **Вызов метода `GetAccount`**:
   - gRPC клиент вызывает метод `GetAccount` сервера, передавая Ethereum-адрес и закодированную подпись.
   - Если подпись валидна (проверяется на сервере), возвращается баланс токенов и nonce кошелька (в вашем примере это фиктивные данные: `Gastoken Balance: 100, Wallet Nonce: 1`).

### Объяснение работы функций клиента:

### 1. Импортируемые пакеты
Клиент использует несколько пакетов:
- `context`: для управления временем жизни запросов.
- `crypto/ecdsa`, `crypto/rand`, `crypto/sha256`: для генерации ключей и подписания сообщений.
- `encoding/base64`: для кодирования подписи в Base64.
- `log`: для ведения журнала и вывода ошибок.
- `pb`: генерированные файлы протоколов gRPC.
- `github.com/ethereum/go-ethereum/crypto`: библиотека для работы с криптографией Ethereum.
- `google.golang.org/grpc`: библиотека для работы с gRPC.

### 2. `main()`
Это основная функция, где выполняются все действия клиента.

#### 2.1 Подключение к gRPC серверу
```go
conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
```
- Клиент устанавливает соединение с gRPC сервером, который работает на `localhost` на порту `50051`. 
- Используется `grpc.WithInsecure()` для небезопасного соединения (без TLS). Если соединение не удалось, программа завершает работу с выводом ошибки.

#### 2.2 Создание клиента
```go
client := pb.NewAccountServiceClient(conn)
```
- Создается клиент для обращения к сервису `AccountService`, определенному в gRPC протоколе.

#### 2.3 Генерация нового Ethereum кошелька
```go
privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
```
- Генерируется пара ключей ECDSA (закрытый и открытый ключ) с использованием стандартной кривой S256.
- Если возникает ошибка, программа завершает работу.

#### 2.4 Получение адреса Ethereum
```go
address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
```
- Из открытого ключа генерируется Ethereum-адрес.

#### 2.5 Создание и подписание сообщения
```go
message := []byte("some message")
hash := sha256.Sum256(message)
signature, err := crypto.Sign(hash[:], privateKey)
```
- Создается сообщение, которое нужно подписать.
- Сообщение хешируется с помощью SHA-256.
- Хеш подписывается с использованием закрытого ключа. Если возникает ошибка, программа завершает работу.

#### 2.6 Кодирование подписи в Base64
```go
signatureBase64 := base64.StdEncoding.EncodeToString(signature)
```
- Подпись кодируется в формат Base64 для удобной передачи по сети.

#### 2.7 Вызов метода `GetAccount`
```go
resp, err := client.GetAccount(context.Background(), &pb.GetAccountRequest{
    EthereumAddress: address,
    CryptoSignature: signatureBase64,
})
```
- Клиент вызывает метод `GetAccount`, передавая Ethereum-адрес и закодированную подпись. 
- `context.Background()` используется для создания контекста без ограничений по времени.

#### 2.8 Обработка ответа
```go
if err != nil {
    log.Fatalf("error calling GetAccount: %v", err)
}
```
- Если возникает ошибка при вызове метода, программа завершает работу с выводом ошибки.

#### 2.9 Вывод информации
```go
log.Printf("Gastoken Balance: %s, Wallet Nonce: %d", resp.GetGastokenBalance(), resp.GetWalletNonce())
```
- Если все прошло успешно, выводится баланс газа и nonce (счетчик транзакций) из ответа сервера.

### Итог
Клиент создает Ethereum-адрес, подписывает сообщение, отправляет запрос на сервер и обрабатывает ответ, выводя информацию о балансе и nonce.
### Вывод:

Клиент успешно вызывает gRPC метод `GetAccount`, получает и выводит баланс и nonce:

```
Gastoken Balance: 100, Wallet Nonce: 1
```

### Объяснение работы функций - test/client_2nd_method.go:

### 1. `main`
```go
func main() {
    // Connecting to the gRPC server
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewAccountServiceClient(conn)

    // Generation of 100, 1000, 10000 Ethereum addresses for testing
    testAddresses := generateEthereumAddresses(10000) // Let's generate the maximum set

    // Testing the method for 100, 1000 and 10000 addresses
    testGetAccountsPerformance(client, testAddresses[:100], 3)
    testGetAccountsPerformance(client, testAddresses[:1000], 4)
    testGetAccountsPerformance(client, testAddresses[:10000], 5)
}
```
- **Подключение к серверу:** Устанавливает соединение с gRPC-сервером по адресу `localhost:50051`.
- **Создание клиента:** Создаёт новый клиент для взаимодействия с сервисом `AccountService`.
- **Генерация адресов:** Генерирует 10,000 тестовых Ethereum-адресов.
- **Тестирование производительности:** Вызывает функцию для тестирования производительности для разных количеств адресов (100, 1000 и 10000) и различного числа токенов.

### 2. `testGetAccountsPerformance`
```go
func testGetAccountsPerformance(client pb.AccountServiceClient, addresses []string, tokenCount int) {
    start := time.Now()

    // Receiving a stream to send and receive data
    stream, err := client.GetAccounts(context.Background())
    if err != nil {
        log.Fatalf("error opening stream: %v", err)
    }

    // Generating real ERC-20 token addresses
    tokenAddresses := generateRealTokenAddresses(tokenCount)

    // We send requests for each token and all addresses
    for _, tokenAddress := range tokenAddresses {
        err = stream.Send(&pb.GetAccountsRequest{
            EthereumAddresses: addresses,
            Erc20TokenAddress: tokenAddress, // Send one token address at a time
        })
        if err != nil {
            log.Fatalf("failed to send request: %v", err)
        }
    }

    // Finish sending data
    if err := stream.CloseSend(); err != nil {
        log.Fatalf("failed to close stream: %v", err)
    }

    // Receiving responses from the server
    for {
        resp, err := stream.Recv()
        if err != nil {
            log.Printf("Stream finished: %v", err)
            break
        }
        log.Printf("Ethereum Address: %s, ERC20 Balance: %s", resp.GetEthereumAddress(), resp.GetErc20Balance())
    }

    elapsed := time.Since(start)
    log.Printf("Time elapsed for %d addresses and %d tokens: %s", len(addresses), tokenCount, elapsed)
}
```
- **Измерение времени:** Начинает отсчёт времени для измерения производительности.
- **Открытие потока:** Создаёт поток для передачи запросов и получения ответов от сервера.
- **Генерация токенов:** Генерирует реальный набор адресов токенов ERC-20 для тестирования.
- **Отправка запросов:** В цикле отправляет запросы с адресами Ethereum и адресами токенов.
- **Закрытие потока:** Закрывает поток после завершения отправки данных.
- **Получение ответов:** Получает ответы от сервера и выводит баланс для каждого адреса.
- **Вывод времени:** Записывает время, затраченное на обработку запросов.

### 3. `generateEthereumAddresses`
```go
func generateEthereumAddresses(count int) []string {
    addresses := make([]string, count)
    for i := 0; i < count; i++ {
        addresses[i] = randomHex(40) // Random hex string 40 characters long (Ethereum address)
    }
    return addresses
}
```
- **Генерация адресов:** Создаёт массив строк для хранения адресов Ethereum и заполняет его случайными hex-строками длиной 40 символов (что соответствует формату Ethereum-адреса).

### 4. `generateRealTokenAddresses`
```go
func generateRealTokenAddresses(count int) []string {
    // Example of real addresses of popular tokens
    realTokenAddresses := []string{
        "0xdAC17F958D2ee523a2206206994597C13D831ec7", // USDT
        "0x6B175474E89094C44Da98b954EedeAC495271d0F", // DAI
        "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", // USDC
        "0x4fabb145d64652a948d72533023f6e7a623c7c53", // BUSD
        "0x0000000000085d4780B73119b644AE5ecd22b376", // TUSD
    }

    if count > len(realTokenAddresses) {
        count = len(realTokenAddresses)
    }

    return realTokenAddresses[:count] // We use real token addresses
}
```
- **Возврат реальных адресов токенов:** Возвращает массив реальных адресов популярных токенов ERC-20. Если запрашиваемое количество превышает доступное, возвращается максимум доступных адресов.

### 5. `randomHex`
```go
func randomHex(n int) string {
    const letters = "0123456789abcdef"
    result := make([]byte, n)
    for i := range result {
        result[i] = letters[rand.Intn(len(letters))]
    }
    return "0x" + string(result)
}
```
- **Генерация случайной hex-строки:** Создаёт случайную hex-строку заданной длины `n`, добавляя префикс `0x`, что является стандартом для Ethereum-адресов.

Resources: 
- https://goethereumbook.org/en
- https://geth.ethereum.org/docs/tools/abigen
- https://grpc.io/docs/languages/go/quickstart/